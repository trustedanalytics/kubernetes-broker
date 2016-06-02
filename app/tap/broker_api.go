/**
 * Copyright (c) 2016 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/gocraft/web"
	"k8s.io/kubernetes/pkg/api"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/consul"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/state"
	"github.com/trustedanalytics/kubernetes-broker/util"
)

type BrokerConfig struct {
	CheckPVbeforeRemoveClusterIntervalSec time.Duration
	WaitBeforeRemoveClusterIntervalSec    time.Duration
	Domain                                string
	CloudProvider                         CloudApi
	StateService                          state.StateService
	KubernetesApi                         k8s.KubernetesApi
	CreatorConnector                      k8s.K8sCreatorRest
	ConsulApi                             consul.ConsulService
}

var brokerConfig *BrokerConfig

func (c *Context) CheckBrokerConfig(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	if brokerConfig == nil {
		util.Respond500(rw, errors.New("brokerConfig not set!"))
	}
	next(rw, req)
}

func (c *Context) Index(rw web.ResponseWriter, req *web.Request) {
	util.WriteJson(rw, "I'm OK", http.StatusOK)
}

// http://docs.cloudfoundry.org/services/api.html#catalog-mgmt
func (c *Context) Catalog(rw web.ResponseWriter, req *web.Request) {
	services := catalog.GetAvailableServicesMetadata()
	util.WriteJson(rw, services, http.StatusOK)
}

func (c *Context) GetServiceDetails(rw web.ResponseWriter, req *web.Request) {
	service_id := req.PathParams["service_id"]

	service, err := catalog.GetServiceMetadataByServiceId(service_id)
	if err != nil {
		util.Respond404(rw, err)
	}
	util.WriteJson(rw, service, http.StatusOK)
}

type ServiceInstancesPutRequest struct {
	OrganizationGuid string          `json:"organization_guid"`
	PlanId           string          `json:"plan_id"`
	ServiceId        string          `json:"service_id"`
	SpaceGuid        string          `json:"space_guid"`
	Parameters       json.RawMessage `json:"parameters"`
	Visibility       bool            `json:"visibility"`
}

type ServiceInstancesPutResponse struct {
	DashboardUrl *string `json:"dashboard_url,omitempty"`
	Error        *string `json:"error,omitempty"`
}

// http://docs.cloudfoundry.org/services/api.html#provisioning
func (c *Context) ServiceInstancesPut(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceInstancesPutRequest{}

	err := util.ReadJson(req, &req_json)
	if err != nil {
		brokerConfig.StateService.ReportProgress("1", "FAILED", err)
		util.Respond500(rw, err)
		return
	}
	instance_id := req.PathParams["instance_id"]
	serviceId := req_json.ServiceId
	org := req_json.OrganizationGuid
	space := req_json.SpaceGuid
	planId := req_json.PlanId

	async := false
	if val, exist := cfenv.CurrentEnv()["ACCEPT_INCOMPLETE"]; exist && val == "true" {
		async = true
	}

	brokerConfig.StateService.ReportProgress(instance_id, "IN_PROGRESS_STARTED", nil)
	svc_meta, plan_meta, err := catalog.WhatToCreateByServiceAndPlanId(serviceId, planId)
	if err != nil {
		brokerConfig.StateService.ReportProgress(instance_id, "FAILED", err)
		util.Respond500(rw, err)
		return
	}
	brokerConfig.StateService.ReportProgress(instance_id, "IN_PROGRESS_METADATA_OK", nil)
	fabrication_function := func() {
		logger.Info("[ServiceInstancesPut] Creating ", svc_meta.Name, " with plan: ", plan_meta.Name)
		brokerConfig.StateService.ReportProgress(instance_id, "IN_PROGRESS_IN_BACKGROUND_JOB", nil)
		component, err := catalog.GetParsedKubernetesComponent(catalog.CatalogPath, instance_id, org, space, svc_meta, plan_meta)
		if err != nil {
			brokerConfig.StateService.ReportProgress(instance_id, "FAILED", err)
			if !async {
				logger.Error(err)
			}
			util.Respond500(rw, err)
			return
		}
		brokerConfig.StateService.ReportProgress(instance_id, "IN_PROGRESS_BLUEPRINT_OK", nil)

		creds, err := brokerConfig.CreatorConnector.GetOrCreateCluster(org)
		if err != nil {
			util.Respond500(rw, err)
			return
		}

		_, err = brokerConfig.KubernetesApi.FabricateService(creds, space, instance_id, string(req_json.Parameters), brokerConfig.StateService, component)
		if err != nil {
			brokerConfig.StateService.ReportProgress(instance_id, "FAILED", err)
			if !async {
				logger.Error(err)
			}
			util.Respond500(rw, err)
			return
		}
		brokerConfig.StateService.ReportProgress(instance_id, "IN_PROGRESS_KUBERNETES_OK", nil)
	}
	if async {
		go fabrication_function()
	} else {
		fabrication_function()
	}

	ret := ServiceInstancesPutResponse{nil, nil}
	url := "UrlNotYetSupported"
	ret.DashboardUrl = &url
	if async {
		util.WriteJson(rw, ret, http.StatusAccepted)
	} else {
		util.WriteJson(rw, ret, http.StatusCreated)
	}

}

func (c *Context) GetQuota(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceInstancesPutRequest{}
	logger.Info("getting quota")
	err := util.ReadJson(req, &req_json)
	if err != nil {
		brokerConfig.StateService.ReportProgress("1", "FAILED", err)
		util.Respond500(rw, err)
		return
	}

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(req_json.OrganizationGuid)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	quotaResource, err := brokerConfig.KubernetesApi.GetQuota(creds, req_json.SpaceGuid)

	if err != nil {
		brokerConfig.StateService.ReportProgress("1", "FAILED", err)
		util.Respond500(rw, err)
		return
	}

	util.WriteJson(rw, quotaResource.Items[0].Status.Used.Memory, http.StatusAccepted)

}

type ServiceInfoResponse struct {
	ServiceId string   `json:"serviceId"`
	Org       string   `json:"org"`
	Space     string   `json:"space"`
	Name      string   `json:"name"`
	TapPublic bool     `json:"tapPublic"`
	Uri       []string `json:"uri"`
}

func (c *Context) GetService(rw web.ResponseWriter, req *web.Request) {
	logger.Info("Fetching service info")
	org := req.PathParams["org_id"]
	space := req.PathParams["space_id"]
	service_id := req.PathParams["instance_id"]

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	services, err := brokerConfig.KubernetesApi.GetService(creds, org, service_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	servicesPublicTags, err := brokerConfig.ConsulApi.GetServicesListWithPublicTagStatus(creds.ConsulEndpoint)
	if err != nil {
		util.Respond500(rw, err)
	}

	response := createServiceInfoList(org, space, services, servicesPublicTags)
	util.WriteJson(rw, response, http.StatusAccepted)

}

func (c *Context) GetServices(rw web.ResponseWriter, req *web.Request) {
	logger.Info("Fetching services info")
	org := req.PathParams["org_id"]
	space := req.PathParams["space_id"]

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	services, err := brokerConfig.KubernetesApi.GetServices(creds, org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	servicesPublicTags, err := brokerConfig.ConsulApi.GetServicesListWithPublicTagStatus(creds.ConsulEndpoint)
	if err != nil {
		util.Respond500(rw, err)
	}

	response := createServiceInfoList(org, space, services, servicesPublicTags)
	util.WriteJson(rw, response, http.StatusAccepted)
}

func (c *Context) SetServiceVisibility(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceInstancesPutRequest{}
	logger.Info("Setting service visibility")
	err := util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(req_json.OrganizationGuid)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	services, err := brokerConfig.KubernetesApi.GetService(creds, req_json.OrganizationGuid, req_json.ServiceId)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	response := []ServiceInfoResponse{}
	consulData := []consul.ConsulServiceParams{}
	for _, service := range services {
		svc := ServiceInfoResponse{
			ServiceId: req_json.ServiceId,
			Org:       req_json.OrganizationGuid,
			Space:     req_json.SpaceGuid,
			Name:      service.ObjectMeta.Name,
			TapPublic: req_json.Visibility,
			Uri:       []string{},
		}

		for _, port := range service.Spec.Ports {
			if port.Protocol != api.ProtocolUDP {
				param := consul.ConsulServiceParams{
					Name:     getConsulServiceName(port, service),
					IsPublic: req_json.Visibility,
					Port:     port.NodePort,
				}
				consulData = append(consulData, param)
				svc.Uri = append(svc.Uri, getServiceExternalAddress(port))
			}
		}

		err := brokerConfig.ConsulApi.UpdateServiceTag(consulData, creds.ConsulEndpoint)
		if err != nil {
			util.Respond500(rw, err)
			return
		}
		response = append(response, svc)
	}
	util.WriteJson(rw, response, http.StatusAccepted)
}

func createServiceInfoList(org, space string, services []api.Service, servicesPublicTags map[string]bool) []ServiceInfoResponse {
	result := []ServiceInfoResponse{}
	for _, service := range services {
		svc := ServiceInfoResponse{
			ServiceId: service.ObjectMeta.Labels["service_id"],
			Org:       org,
			Space:     space,
			Name:      service.ObjectMeta.Name,
			TapPublic: readTapPublic(service.ObjectMeta.Name, servicesPublicTags),
		}

		for _, port := range service.Spec.Ports {
			svc.Uri = append(svc.Uri, getServiceExternalAddress(port))
		}

		result = append(result, svc)
	}
	return result
}

func readTapPublic(serviceName string, servicesPublicTags map[string]bool) bool {
	for k, v := range servicesPublicTags {
		if strings.Contains(k, serviceName) {
			return v
		}
	}
	return false
}

func getServiceExternalAddress(port api.ServicePort) string {
	return strings.ToLower(string(port.Protocol)) + "." + brokerConfig.Domain + ":" + strconv.Itoa(int(port.NodePort))
}

func getServiceInternalHost(port api.ServicePort, service api.Service) string {
	return getConsulServiceName(port, service) + ".service.consul"
}

func getConsulServiceName(port api.ServicePort, service api.Service) string {
	portName := ""
	if len(service.Spec.Ports) > 1 {
		if port.Name != "" {
			portName = "-" + port.Name
		}
	}
	return service.ObjectMeta.Name + portName
}

type ServiceInstancesGetLastOperationResponse struct {
	State       string  `json:"state"` // in progress, succeeded, failed
	Description *string `json:"description"`
}

// http://docs.cloudfoundry.org/services/api.html#asynchronous-operations
func (c *Context) ServiceInstancesGetLastOperation(rw web.ResponseWriter, req *web.Request) {
	instance_id := req.PathParams["instance_id"]

	org, space, err := brokerConfig.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	var stateValue string
	var description string

	if brokerConfig.StateService.HasProgressRecords(instance_id) {
		ts, description, e := brokerConfig.StateService.ReadProgress(instance_id)
		if e != nil || strings.HasPrefix(description, "FAIL") {
			stateValue = "failed"
			logger.Error("[ServiceInstancesGetLastOperation] Error found! Status set to:", stateValue, err)
		} else if time.Since(ts) > (time.Duration(20) * time.Minute) {
			stateValue = "failed"
			logger.Error("[ServiceInstancesGetLastOperation] creating service takes too long! Status set to:", stateValue)
		} else if description == "IN_PROGRESS_KUBERNETES_OK" {
			healthy, err := brokerConfig.KubernetesApi.CheckKubernetesServiceHealthByServiceInstanceId(creds, space, instance_id)
			if err != nil {
				stateValue = "in progress"
			} else if healthy {
				stateValue = "succeeded"
			} else {
				stateValue = "in progress"
			}
		}
	} else {
		// FIXME - this state should only happen when state is being stored in-memory. It should occur only during the initial platform deployment stage...
		stateValue = "failed"
		logger.Error("[ServiceInstancesGetLastOperation] No service data in StateService! Status set to:", stateValue)
	}

	logger.Info("[ServiceInstancesGetLastOperation] result: ", stateValue, "serviceId: ", instance_id, "org: ", org, "space: ", space)
	util.WriteJson(rw, ServiceInstancesGetLastOperationResponse{stateValue, &description}, http.StatusOK)
}

type ServiceInstancesDeleteResponse struct {
}

// DELETE /v2/service_instances/:instance_id?plan_id=ddd3fc74-8b8d-422b-8217-4a8eb6b6cddd&service_id=dddf9a19-a193-4a86-b449-b448350dbddd
func (c *Context) ServiceInstancesDelete(rw web.ResponseWriter, req *web.Request) {
	instance_id := req.PathParams["instance_id"]
	plan_id := req.URL.Query().Get("plan_id")
	service_id := req.URL.Query().Get("service_id")
	logger.Debug("ServiceInstancesDelete instance:", instance_id, "plan:", plan_id, "service", service_id)

	org, _, err := brokerConfig.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	status, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		if status != 200 {
			util.WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusGone)
			return
		}
		util.Respond500(rw, err)
		return
	}

	if status == 404 || status == 204 {
		logger.Error("Cluster not exist! We can't remove service, service_id:", service_id)
		util.WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusGone)
		return
	}

	err = brokerConfig.KubernetesApi.DeleteAllByServiceId(creds, instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	go removeCluster(creds, org)

	logger.Info("Service DELETED. Id:", service_id)
	util.WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusOK)
}

func removeCluster(creds k8s.K8sClusterCredentials, org string) {
	time.Sleep(brokerConfig.WaitBeforeRemoveClusterIntervalSec)

	for {
		services, err := brokerConfig.KubernetesApi.GetServices(creds, org)
		if err != nil {
			logger.Error("[removeCluster] GetServices error. Org:", org, err)
			return
		}

		controllers, err := brokerConfig.KubernetesApi.ListReplicationControllers(creds)
		if err != nil {
			logger.Error("[removeCluster] ListReplicationControllers error. Org:", org, err)
			return
		}

		if len(services) == 0 && len(controllers.Items) == 0 {
			err = brokerConfig.KubernetesApi.DeleteAllPersistentVolumeClaims(creds)
			if err != nil {
				logger.Error("[removeCluster] DeleteAllPersistentVolumeClaims error. Org:", org, err)
				return
			}

			pvList, err := brokerConfig.KubernetesApi.GetAllPersistentVolumes(creds)
			if err != nil {
				logger.Error("[removeCluster] GetAllPersistentVolumes error. Org:", org, err)
				return
			}

			if len(pvList) == 0 {
				logger.Info(fmt.Sprintf("[removeCluster] There is no more Services and PersistentVolumes for the org: %s. Cluster will be removed now...", org))
				err = brokerConfig.CreatorConnector.DeleteCluster(org)
				if err != nil {
					logger.Error(err)
					return
				}
				logger.Info("[removeCluster] Cluster removed successfully! Org:", org)
				return
			} else {
				logger.Warning(fmt.Sprintf("[removeCluster] There are still some PersistentVolumes for the org: %s. Waiting for EBS to delete them...", org))
				time.Sleep(brokerConfig.CheckPVbeforeRemoveClusterIntervalSec)
			}
		} else {
			logger.Warning("[removeCluster] Some ervices exist! Removing cluster stopped! Org:", org)
			return
		}
	}
}

type ServiceBindingsPutRequest struct {
	ServiceId    *string                `json:"service_id,omitempty"`
	PlanId       *string                `json:"plan_id,omitempty"`
	AppGuid      string                 `json:"app_guid"`
	BindResource interface{}            `json:"bind_resource"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// http://docs.cloudfoundry.org/services/api.html#binding
func (c *Context) ServiceBindingsPut(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceBindingsPutRequest{}
	util.ReadJson(req, &req_json)
	instance_id := req.PathParams["instance_id"] // already provisioned instance
	binding_id := req.PathParams["binding_id"]   // used for unbinding

	if req_json.ServiceId == nil || req_json.PlanId == nil {
		util.Respond500(rw, errors.New("service id or plan id is nil - at this stage, we won't continue. TODO: ask CF to retrieve those from API, by instance_id"))
		return
	} else {
		logger.Debug(req_json, instance_id, binding_id, "ServiceID=", *req_json.ServiceId, "PlanID=", *req_json.PlanId)
	}

	svc_meta, plan_meta, err := catalog.WhatToCreateByServiceAndPlanId(*req_json.ServiceId, *req_json.PlanId)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	logger.Info("Binding, found blueprint name: ", svc_meta.Name, " with plan: ", plan_meta.Name)

	org, space, err := brokerConfig.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	logger.Debug("org: ", org, "space: ", space)

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	podsEnvs, err := brokerConfig.KubernetesApi.GetAllPodsEnvsByServiceId(creds, space, instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	svcCreds, err := getServiceCredentials(creds, space, instance_id)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	blueprint, err := catalog.GetKubernetesBlueprintByServiceAndPlan(catalog.CatalogPath, svc_meta, plan_meta)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	logger.Debug("CredentialMappings: ", blueprint.CredentialsMapping)

	mapping, err := ParseCredentialMappingAdvanced(svc_meta.Name, svcCreds, podsEnvs, blueprint)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	ret := `{ "credentials": ` + mapping + ` }`
	logger.Info("[ServiceBindingsPut] Responding with parsed credential JSON: ", ret)
	rw.WriteHeader(http.StatusCreated)
	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, "%s", ret)
}

type ServiceCredential struct {
	Name  string
	Host  string
	Ports []api.ServicePort
}

func getServiceCredentials(creds k8s.K8sClusterCredentials, org, serviceId string) ([]ServiceCredential, error) {
	logger.Info("[GetServiceCredentials] serviceId:", serviceId)
	result := []ServiceCredential{}

	services, err := brokerConfig.KubernetesApi.GetService(creds, org, serviceId)
	if err != nil {
		return result, err
	}
	if len(services) < 1 {
		return result, errors.New("No services associated with the serviceId: " + serviceId)
	}

	for _, svc := range services {
		svcCred := ServiceCredential{}
		svcCred.Name = svc.Name
		svcCred.Host = getServiceInternalHostByFirstTCPPort(svc)

		for _, p := range svc.Spec.Ports {
			svcCred.Ports = append(svcCred.Ports, p)
		}
		result = append(result, svcCred)
	}
	return result, nil
}

func getServiceInternalHostByFirstTCPPort(service api.Service) string {
	for _, port := range service.Spec.Ports {
		if port.Protocol == api.ProtocolTCP {
			return getServiceInternalHost(port, service)
		}
	}
	return ""
}

type ServiceBindingsDeleteResponse struct {
}

// http://docs.cloudfoundry.org/services/api.html#binding
// DELETE /v2/service_instances/:instance_id/service_bindings/:binding_id
func (c *Context) ServiceBindingsDelete(rw web.ResponseWriter, req *web.Request) {
	instance_id := req.PathParams["instance_id"]
	binding_id := req.PathParams["binding_id"]
	plan_id := req.URL.Query().Get("plan_id")
	service_id := req.URL.Query().Get("service_id")
	logger.Info("ServiceBindingsDelete instance:", instance_id, "binding:", binding_id, "plan:", plan_id, "service", service_id)

	util.WriteJson(rw, ServiceBindingsDeleteResponse{}, http.StatusGone)
}

type DynamicServiceRequest struct {
	OrganizationGuid string                 `json:"organization_guid"`
	SpaceGuid        string                 `json:"space_guid"`
	Parameters       json.RawMessage        `json:"parameters"`
	UpdateBroker     bool                   `json:"updateBroker"`
	DynamicService   catalog.DynamicService `json:"dynamicService"`
}

func (c *Context) CreateAndRegisterDynamicService(rw web.ResponseWriter, req *web.Request) {
	req_json := DynamicServiceRequest{}

	err := util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	if catalog.CheckIfServiceAlreadyExist(req_json.DynamicService.ServiceName) {
		util.Respond500(rw, errors.New("Service with name: "+req_json.DynamicService.ServiceName+" already exists!"))
		return
	}

	blueprint, _, service, err := catalog.CreateDynamicService(req_json.DynamicService)
	if err != nil {
		logger.Error("[CreateAndRegisterDynamicService] CreateDynamicService fail!", err)
		util.Respond500(rw, err)
		return
	}

	catalog.RegisterOfferingInCatalog(service, blueprint)

	if req_json.UpdateBroker {
		_, err = brokerConfig.CloudProvider.UpdateServiceBroker()
		if err != nil {
			util.Respond500(rw, err)
			return
		}

		//now register service using cli:
		// cf enable-service-access your-service-name
	}
	util.WriteJson(rw, "", http.StatusCreated)
}

func (c *Context) DeleteAndUnRegisterDynamicService(rw web.ResponseWriter, req *web.Request) {
	req_json := DynamicServiceRequest{}

	err := util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	service, err := catalog.GetServiceByName(req_json.DynamicService.ServiceName)
	if err != nil {
		logger.Error("[DeleteAndUnRegisterDynamicService] Delete DynamicService fail!", err)
		util.WriteJson(rw, "", http.StatusGone)
		return
	}

	catalog.UnregisterOfferingFromCatalog(service)

	//TODO we not persist copy of dynamic services yet, but remember to remove it in the future

	if req_json.UpdateBroker {
		_, err = brokerConfig.CloudProvider.UpdateServiceBroker()
		if err != nil {
			util.Respond500(rw, err)
			return
		}
	}
	util.WriteJson(rw, "", http.StatusNoContent)

}

func (c *Context) CheckPodsStatusForService(rw web.ResponseWriter, req *web.Request) {
	instanceId := req.PathParams["instance_id"]
	orgId := req.PathParams["org_id"]

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(orgId)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	podsStates, err := brokerConfig.KubernetesApi.GetPodsStateByServiceId(creds, instanceId)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	util.WriteJson(rw, podsStates, http.StatusOK)
}

func (c *Context) CheckPodsStatusForAllServicesInOrg(rw web.ResponseWriter, req *web.Request) {
	orgId := req.PathParams["org_id"]

	_, creds, err := brokerConfig.CreatorConnector.GetCluster(orgId)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	podsStates, err := brokerConfig.KubernetesApi.GetPodsStateForAllServices(creds)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	util.WriteJson(rw, podsStates, http.StatusOK)
}

func (c *Context) GetSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	key := req.PathParams["key"]
	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	secret, err := brokerConfig.KubernetesApi.GetSecret(creds, key)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	util.WriteJson(rw, secret, http.StatusOK)
}

func (c *Context) CreateSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	req_json := api.Secret{}
	err = util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	err = brokerConfig.KubernetesApi.CreateSecret(creds, req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	return
}

func (c *Context) DeleteSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	key := req.PathParams["key"]
	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	err = brokerConfig.KubernetesApi.DeleteSecret(creds, key)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	return
}

func (c *Context) UpdateSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	_, creds, err := brokerConfig.CreatorConnector.GetCluster(org)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	req_json := api.Secret{}
	err = util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	err = brokerConfig.KubernetesApi.UpdateSecret(creds, req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	return
}

func (c *Context) Error(rw web.ResponseWriter, r *web.Request, err interface{}) {
	logger.Error("Respond500: reason: error ", err)
	rw.WriteHeader(http.StatusInternalServerError)
}
