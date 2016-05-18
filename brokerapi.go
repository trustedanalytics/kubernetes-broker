/**
 * Copyright (c) 2015 Intel Corporation
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
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/state"
	"k8s.io/kubernetes/pkg/api"
)

var cloudProvider CloudApi
var stateService state.StateService
var kubernetesApi k8s.KubernetesApi
var creatorConnector k8s.K8sCreatorRest

// here we can inject specfic implementation for our services. This is for test purpose.
// Normally we should inject it directly into Context during inicialization, but gocraft/web dosen't allow for this.
func (c *Context) SetupContext(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	c.CloudProvider = cloudProvider
	c.KubernetesApi = kubernetesApi
	c.StateService = stateService
	c.CreatorConnector = creatorConnector
	next(rw, req)
}

func (c *Context) Index(rw web.ResponseWriter, req *web.Request) {
	WriteJson(rw, "I'm OK", http.StatusOK)
}

// http://docs.cloudfoundry.org/services/api.html#catalog-mgmt
func (c *Context) Catalog(rw web.ResponseWriter, req *web.Request) {
	services := catalog.GetAvailableServicesMetadata()
	WriteJson(rw, services, http.StatusOK)
}

func (c *Context) GetServiceDetails(rw web.ResponseWriter, req *web.Request) {
	service_id := req.PathParams["service_id"]

	service, err := catalog.GetServiceMetadataByServiceId(service_id)
	if err != nil {
		Respond404(rw, err)
	}
	WriteJson(rw, service, http.StatusOK)
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

	err := ReadJson(req, &req_json)
	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
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

	c.StateService.ReportProgress(instance_id, "IN_PROGRESS_STARTED", nil)
	svc_meta, plan_meta, err := catalog.WhatToCreateByServiceAndPlanId(serviceId, planId)
	if err != nil {
		c.StateService.ReportProgress(instance_id, "FAILED", err)
		Respond500(rw, err)
		return
	}
	c.StateService.ReportProgress(instance_id, "IN_PROGRESS_METADATA_OK", nil)
	fabrication_function := func() {
		logger.Info("[ServiceInstancesPut] Creating ", svc_meta.Name, " with plan: ", plan_meta.Name)
		c.StateService.ReportProgress(instance_id, "IN_PROGRESS_IN_BACKGROUND_JOB", nil)
		component, err := catalog.GetParsedKubernetesComponent(catalog.CatalogPath, instance_id, org, space, svc_meta, plan_meta)
		if err != nil {
			c.StateService.ReportProgress(instance_id, "FAILED", err)
			if !async {
				logger.Error(err)
			}
			Respond500(rw, err)
			return
		}
		c.StateService.ReportProgress(instance_id, "IN_PROGRESS_BLUEPRINT_OK", nil)

		creds, err := c.CreatorConnector.GetOrCreateCluster(org)
		if err != nil {
			Respond500(rw, err)
			return
		}

		_, err = c.KubernetesApi.FabricateService(creds, space, instance_id, string(req_json.Parameters), c.StateService, component)
		if err != nil {
			c.StateService.ReportProgress(instance_id, "FAILED", err)
			if !async {
				logger.Error(err)
			}
			Respond500(rw, err)
			return
		}
		c.StateService.ReportProgress(instance_id, "IN_PROGRESS_KUBERNETES_OK", nil)
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
		WriteJson(rw, ret, http.StatusAccepted)
	} else {
		WriteJson(rw, ret, http.StatusCreated)
	}

}

func (c *Context) GetQuota(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceInstancesPutRequest{}
	logger.Info("getting quota")
	err := ReadJson(req, &req_json)
	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
		return
	}

	_, creds, err := c.CreatorConnector.GetCluster(req_json.OrganizationGuid)
	if err != nil {
		Respond500(rw, err)
		return
	}

	quotaResource, err := c.KubernetesApi.GetQuota(creds, req_json.SpaceGuid)

	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
		return
	}

	WriteJson(rw, quotaResource.Items[0].Status.Used.Memory, http.StatusAccepted)

}

func (c *Context) GetService(rw web.ResponseWriter, req *web.Request) {
	logger.Info("Fetching service")
	org := req.PathParams["org_id"]
	space := req.PathParams["space_id"]
	service_id := req.PathParams["instance_id"]

	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}

	services, err := c.KubernetesApi.GetServiceVisibility(creds, org, space, service_id)

	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
		return
	}

	WriteJson(rw, services, http.StatusAccepted)

}

func (c *Context) GetServices(rw web.ResponseWriter, req *web.Request) {
	logger.Info("Fetching serice")
	org := req.PathParams["org_id"]
	space := req.PathParams["space_id"]

	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		//Respond500(rw, err)
		WriteJson(rw, []interface{}{}, http.StatusAccepted)
		return
	}

	controllers, err := c.KubernetesApi.GetServicesVisibility(creds, org, space)
	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		//Respond500(rw, err)
		WriteJson(rw, []interface{}{}, http.StatusAccepted)
		return
	}

	WriteJson(rw, controllers, http.StatusAccepted)

}

func (c *Context) SetServiceVisibility(rw web.ResponseWriter, req *web.Request) {
	req_json := ServiceInstancesPutRequest{}
	logger.Info("Setting service visibility")
	err := ReadJson(req, &req_json)
	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
		return
	}

	_, creds, err := c.CreatorConnector.GetCluster(req_json.OrganizationGuid)
	if err != nil {
		Respond500(rw, err)
		return
	}

	replicationControllerItem, err := c.KubernetesApi.SetServicePublicVisibilityByServiceId(creds, req_json.OrganizationGuid,
		req_json.SpaceGuid, req_json.ServiceId, req_json.Visibility)

	if err != nil {
		c.StateService.ReportProgress("1", "FAILED", err)
		Respond500(rw, err)
		return
	}
	WriteJson(rw, replicationControllerItem, http.StatusAccepted)
}

type ServiceInstancesGetLastOperationResponse struct {
	State       string  `json:"state"` // in progress, succeeded, failed
	Description *string `json:"description"`
}

// http://docs.cloudfoundry.org/services/api.html#asynchronous-operations
func (c *Context) ServiceInstancesGetLastOperation(rw web.ResponseWriter, req *web.Request) {
	instance_id := req.PathParams["instance_id"]

	org, space, err := c.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}

	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}

	var stateValue string
	var description string

	if c.StateService.HasProgressRecords(instance_id) {
		ts, description, e := c.StateService.ReadProgress(instance_id)
		if e != nil || strings.HasPrefix(description, "FAIL") {
			stateValue = "failed"
			logger.Error("[ServiceInstancesGetLastOperation] Error found! Status set to:", stateValue, err)
		} else if time.Since(ts) > (time.Duration(20) * time.Minute) {
			stateValue = "failed"
			logger.Error("[ServiceInstancesGetLastOperation] creating service takes too long! Status set to:", stateValue)
		} else if description == "IN_PROGRESS_KUBERNETES_OK" {
			healthy, err := c.KubernetesApi.CheckKubernetesServiceHealthByServiceInstanceId(creds, space, instance_id)
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
	WriteJson(rw, ServiceInstancesGetLastOperationResponse{stateValue, &description}, http.StatusOK)
}

type ServiceInstancesDeleteResponse struct {
}

// DELETE /v2/service_instances/:instance_id?plan_id=ddd3fc74-8b8d-422b-8217-4a8eb6b6cddd&service_id=dddf9a19-a193-4a86-b449-b448350dbddd
func (c *Context) ServiceInstancesDelete(rw web.ResponseWriter, req *web.Request) {
	instance_id := req.PathParams["instance_id"]
	plan_id := req.URL.Query().Get("plan_id")
	service_id := req.URL.Query().Get("service_id")
	logger.Debug("ServiceInstancesDelete instance:", instance_id, "plan:", plan_id, "service", service_id)

	org, space, err := c.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}

	status, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		if status != 200 {
			WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusGone)
			return
		}
		Respond500(rw, err)
		return
	}

	if status == 404 || status == 204 {
		logger.Error("Cluster not exist! We can't remove service, service_id:", service_id)
		WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusGone)
		return
	}

	err = c.KubernetesApi.DeleteAllByServiceId(creds, space, instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}

	//check if there is more services, if not then remove cluster
	//todo currently we use hardcoded "deault" space
	//todo -> in the feature we should check if other spaces in organization also don't contain any services
	services, err := c.KubernetesApi.GetServices(creds, org)
	if err != nil {
		WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusGone)
		logger.Error(err)
		return
	}

	controllers, err := c.KubernetesApi.ListReplicationControllers(creds, space)
	if err != nil {
		Respond500(rw, err)
		return
	}

	if len(services) == 0 || len(controllers.Items) == 0 {
		logger.Info("There is no more services in the org. Cluster will be removed now...")

		err = c.KubernetesApi.DeleteAllPersistentVolumes(creds)
		if err != nil {
			Respond500(rw, err)
			return
		}

		err = c.CreatorConnector.DeleteCluster(org)
		if err != nil {
			Respond500(rw, err)
			return
		}
	}
	logger.Info("Service DELETED. Id:", service_id)
	WriteJson(rw, ServiceInstancesDeleteResponse{}, http.StatusOK)
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
	ReadJson(req, &req_json)
	instance_id := req.PathParams["instance_id"] // already provisioned instance
	binding_id := req.PathParams["binding_id"]   // used for unbinding

	if req_json.ServiceId == nil || req_json.PlanId == nil {
		Respond500(rw, errors.New("service id or plan id is nil - at this stage, we won't continue. TODO: ask CF to retrieve those from API, by instance_id"))
		return
	} else {
		logger.Debug(req_json, instance_id, binding_id, "ServiceID=", *req_json.ServiceId, "PlanID=", *req_json.PlanId)
	}

	svc_meta, plan_meta, err := catalog.WhatToCreateByServiceAndPlanId(*req_json.ServiceId, *req_json.PlanId)
	if err != nil {
		Respond500(rw, err)
		return
	}
	logger.Info("Binding, found blueprint name: ", svc_meta.Name, " with plan: ", plan_meta.Name)

	org, space, err := c.CloudProvider.GetOrgIdAndSpaceIdFromCfByServiceInstanceId(instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}
	logger.Debug("org: ", org, "space: ", space)

	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}

	podsEnvs, err := c.KubernetesApi.GetAllPodsEnvsByServiceId(creds, space, instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}

	svcCreds, err := c.KubernetesApi.GetServiceCredentials(creds, space, instance_id)
	if err != nil {
		Respond500(rw, err)
		return
	}

	blueprint, err := catalog.GetKubernetesBlueprintByServiceAndPlan(catalog.CatalogPath, svc_meta, plan_meta)
	if err != nil {
		Respond500(rw, err)
		return
	}
	logger.Debug("CredentialMappings: ", blueprint.CredentialsMapping)

	mapping, err := ParseCredentialMappingAdvanced(svc_meta.Name, svcCreds, podsEnvs, blueprint)
	if err != nil {
		Respond500(rw, err)
		return
	}

	ret := `{ "credentials": ` + mapping + ` }`
	logger.Info("[ServiceBindingsPut] Responding with parsed credential JSON: ", ret)
	rw.WriteHeader(http.StatusCreated)
	rw.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(rw, "%s", ret)
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

	WriteJson(rw, ServiceBindingsDeleteResponse{}, http.StatusGone)
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

	err := ReadJson(req, &req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}

	if catalog.CheckIfServiceAlreadyExist(req_json.DynamicService.ServiceName) {
		Respond500(rw, errors.New("Service with name: "+req_json.DynamicService.ServiceName+" already exists!"))
		return
	}

	blueprint, _, service, err := catalog.CreateDynamicService(req_json.DynamicService)
	if err != nil {
		logger.Error("[CreateAndRegisterDynamicService] CreateDynamicService fail!", err)
		Respond500(rw, err)
		return
	}

	catalog.RegisterOfferingInCatalog(service, blueprint)

	if req_json.UpdateBroker {
		_, err = cloudProvider.UpdateServiceBroker()
		if err != nil {
			Respond500(rw, err)
			return
		}

		//now register service using cli:
		// cf enable-service-access your-service-name
	}
	WriteJson(rw, "", http.StatusCreated)
}

func (c *Context) DeleteAndUnRegisterDynamicService(rw web.ResponseWriter, req *web.Request) {
	req_json := DynamicServiceRequest{}

	err := ReadJson(req, &req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}

	service, err := catalog.GetServiceByName(req_json.DynamicService.ServiceName)
	if err != nil {
		logger.Error("[DeleteAndUnRegisterDynamicService] Delete DynamicService fail!", err)
		Respond500(rw, err)
		return
	}

	catalog.UnregisterOfferingFromCatalog(service)

	//TODO we not persist copy of dynamic services yet, but remember to remove it in the future

	if req_json.UpdateBroker {
		_, err = cloudProvider.UpdateServiceBroker()
		if err != nil {
			Respond500(rw, err)
			return
		}
	}
	WriteJson(rw, "", http.StatusGone)

}

func (c *Context) CheckPodsStatusForService(rw web.ResponseWriter, req *web.Request) {
	instanceId := req.PathParams["instance_id"]
	orgId := req.PathParams["org_id"]

	_, creds, err := c.CreatorConnector.GetCluster(orgId)
	if err != nil {
		Respond500(rw, err)
		return
	}

	podsStates, err := c.KubernetesApi.GetPodsStateByServiceId(creds, instanceId)
	if err != nil {
		Respond500(rw, err)
		return
	}
	WriteJson(rw, podsStates, http.StatusOK)
}

func (c *Context) CheckPodsStatusForAllServicesInOrg(rw web.ResponseWriter, req *web.Request) {
	orgId := req.PathParams["org_id"]

	_, creds, err := c.CreatorConnector.GetCluster(orgId)
	if err != nil {
		Respond500(rw, err)
		return
	}

	podsStates, err := c.KubernetesApi.GetPodsStateForAllServices(creds)
	if err != nil {
		Respond500(rw, err)
		return
	}
	WriteJson(rw, podsStates, http.StatusOK)
}

func (c *Context) GetSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	key := req.PathParams["key"]
	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}
	secret, err := c.KubernetesApi.GetSecret(creds, key)
	if err != nil {
		Respond500(rw, err)
		return
	}
	WriteJson(rw, secret, http.StatusOK)
}

func (c *Context) CreateSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}
	req_json := api.Secret{}
	err = ReadJson(req, &req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}
	err = c.KubernetesApi.CreateSecret(creds, req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}
	return
}

func (c *Context) DeleteSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	key := req.PathParams["key"]
	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}
	err = c.KubernetesApi.DeleteSecret(creds, key)
	if err != nil {
		Respond500(rw, err)
		return
	}
	return
}

func (c *Context) UpdateSecret(rw web.ResponseWriter, req *web.Request) {
	org := req.PathParams["org_id"]
	_, creds, err := c.CreatorConnector.GetCluster(org)
	if err != nil {
		Respond500(rw, err)
		return
	}
	req_json := api.Secret{}
	err = ReadJson(req, &req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}
	err = c.KubernetesApi.UpdateSecret(creds, req_json)
	if err != nil {
		Respond500(rw, err)
		return
	}
	return
}
