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

package k8s

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/util/sets"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/consul"
	"github.com/trustedanalytics/kubernetes-broker/state"
)

type KubernetesApi interface {
	FabricateService(creds K8sClusterCredential, space, cf_service_id, parameters string, ss state.StateService,
		component *catalog.KubernetesComponent) (FabricateResult, error)
	CheckKubernetesServiceHealthByServiceInstanceId(creds K8sClusterCredential, space, instance_id string) (bool, error)
	DeleteAllByServiceId(creds K8sClusterCredential, space, service_id string) error
	DeleteAllPersistentVolumes(creds K8sClusterCredential) error
	GetAllPodsEnvsByServiceId(creds K8sClusterCredential, space, service_id string) ([]PodEnvs, error)
	GetServiceVisibility(creds K8sClusterCredential, org, space, service_id string) ([]K8sServiceInfo, error)
	GetServicesVisibility(creds K8sClusterCredential, org, space string) ([]K8sServiceInfo, error)
	GetServiceCredentials(creds K8sClusterCredential, space, service_id string) ([]ServiceCredential, error)
	GetServices(creds K8sClusterCredential, org string) ([]api.Service, error)
	GetQuota(creds K8sClusterCredential, space string) (*api.ResourceQuotaList, error)
	GetClusterWorkers(creds K8sClusterCredential) ([]string, error)
	GetPodsStateByServiceId(creds K8sClusterCredential, service_id string) ([]PodStatus, error)
	GetPodsStateForAllServices(creds K8sClusterCredential) (map[string][]PodStatus, error)
	SetServicePublicVisibilityByServiceId(creds K8sClusterCredential, org, space, service_id string, shouldBePublic bool) ([]K8sServiceInfo, error)
	ListReplicationControllers(creds K8sClusterCredential, space string) (*api.ReplicationControllerList, error)
	GetSecret(creds K8sClusterCredential, key string) (*api.Secret, error)
	CreateSecret(creds K8sClusterCredential, secret api.Secret) error
	DeleteSecret(creds K8sClusterCredential, key string) error
	UpdateSecret(creds K8sClusterCredential, secret api.Secret) error
}

type K8Fabricator struct {
	KubernetesClient KubernetesClientCreator
	ConsulApi        consul.ConsulService
	Domain           string
}

type FabricateResult struct {
	Url string
	Env map[string]string
}

type K8sServiceInfo struct {
	ServiceId string   `json:"serviceId"`
	Org       string   `json:"org"`
	Space     string   `json:"space"`
	Name      string   `json:"name"`
	TapPublic bool     `json:"tapPublic"`
	Uri       []string `json:"uri"`
}

func NewK8Fabricator(domain string) *K8Fabricator {
	return &K8Fabricator{KubernetesClient: &KubernetesRestCreator{}, ConsulApi: &consul.ConsulConnector{}, Domain: domain}
}

func (k *K8Fabricator) FabricateService(creds K8sClusterCredential, space, cf_service_id, parameters string,
	ss state.StateService, component *catalog.KubernetesComponent) (FabricateResult, error) {
	result := FabricateResult{"", map[string]string{}}

	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return result, err
	}

	extraEnvironments := []api.EnvVar{{Name: "TAP_K8S", Value: "true"}}
	if parameters != "" {
		extraUserParam := api.EnvVar{}
		err := json.Unmarshal([]byte(parameters), &extraUserParam)
		if err != nil {
			logger.Error("[FabricateService] Unmarshalling extra user parameters error!", err)
			return result, err
		}

		if extraUserParam.Name != "" {
			// kubernetes env name validation:
			// "must be a C identifier (matching regex [A-Za-z_][A-Za-z0-9_]*): e.g. \"my_name\" or \"MyName\"","
			extraUserParam.Name = extraUserParam.Name + "_" + space
			extraUserParam.Name = strings.Replace(extraUserParam.Name, "_", "__", -1) //name_1 --> name__1__SpaceGUID
			extraUserParam.Name = strings.Replace(extraUserParam.Name, "-", "_", -1)  //name-1 --> name_1__SpaceGUID

			extraEnvironments = append(extraEnvironments, extraUserParam)
		}
		logger.Debug("[FabricateService] Extra parameters value:", extraEnvironments)
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_SECRETS", nil)
	for idx, sc := range component.Secrets {
		ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_SECRET"+strconv.Itoa(idx), nil)
		_, err = c.Secrets(api.NamespaceDefault).Create(sc)
		if err != nil {
			ss.ReportProgress(cf_service_id, "FAILED", err)
			return result, err
		}
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIMS", nil)
	for idx, claim := range component.PersistentVolumeClaim {
		ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIM"+strconv.Itoa(idx), nil)
		_, err = c.PersistentVolumeClaims(api.NamespaceDefault).Create(claim)
		if err != nil {
			ss.ReportProgress(cf_service_id, "FAILED", err)
			return result, err
		}
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_RCS", nil)
	for idx, rc := range component.ReplicationControllers {
		ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_RC"+strconv.Itoa(idx), nil)
		for i, container := range rc.Spec.Template.Spec.Containers {
			rc.Spec.Template.Spec.Containers[i].Env = append(container.Env, extraEnvironments...)
		}

		_, err = NewReplicationControllerManager(c).Create(rc)
		if err != nil {
			ss.ReportProgress(cf_service_id, "FAILED", err)
			return result, err
		}
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_SVCS", nil)
	for idx, svc := range component.Services {
		ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_SVC"+strconv.Itoa(idx), nil)
		_, err = c.Services(api.NamespaceDefault).Create(svc)
		if err != nil {
			ss.ReportProgress(cf_service_id, "FAILED", err)
			return result, err
		}
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_ACCS", nil)
	for idx, acc := range component.ServiceAccounts {
		ss.ReportProgress(cf_service_id, "IN_PROGRESS_CREATING_ACC"+strconv.Itoa(idx), nil)
		_, err = c.ServiceAccounts(api.NamespaceDefault).Create(acc)
		if err != nil {
			ss.ReportProgress(cf_service_id, "FAILED", err)
			return result, err
		}
	}

	ss.ReportProgress(cf_service_id, "IN_PROGRESS_FAB_OK", nil)
	return result, nil
}

func (k *K8Fabricator) CheckKubernetesServiceHealthByServiceInstanceId(creds K8sClusterCredential, space, instance_id string) (bool, error) {
	logger.Info("[CheckKubernetesServiceHealthByServiceInstanceId] serviceId:", instance_id)
	// http://kubernetes.io/v1.1/docs/user-guide/liveness/README.html

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, instance_id)
	if err != nil {
		return false, err
	}

	pods, err := c.Pods(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[CheckKubernetesServiceHealthByServiceInstanceId] Getting pods failed:", err)
		return false, err
	}
	logger.Debug("[CheckKubernetesServiceHealthByServiceInstanceId] PODS:", pods)

	// for each pod check if healthy
	// if all healthy return true
	// else return false

	return true, nil
}

func (k *K8Fabricator) DeleteAllByServiceId(creds K8sClusterCredential, space, service_id string) error {
	logger.Info("[DeleteAllByServiceId] serviceId:", service_id)

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return err
	}

	accs, err := c.ServiceAccounts(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[DeleteAllByServiceId] List service accounts failed:", err)
		return err
	}
	var name string
	for _, i := range accs.Items {
		name = i.ObjectMeta.Name
		logger.Debug("[DeleteAllByServiceId] Delete service account:", name)
		err = c.ServiceAccounts(api.NamespaceDefault).Delete(name)
		if err != nil {
			logger.Error("[DeleteAllByServiceId] Delete service account failed:", err)
			return err
		}
	}

	svcs, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[DeleteAllByServiceId] List services failed:", err)
		return err
	}

	for _, i := range svcs.Items {
		name = i.ObjectMeta.Name
		logger.Debug("[DeleteAllByServiceId] Delete service:", name)
		err = c.Services(api.NamespaceDefault).Delete(name)
		if err != nil {
			logger.Error("[DeleteAllByServiceId] Delete service failed:", err)
			return err
		}
	}

	if err = NewReplicationControllerManager(c).DeleteAll(selector); err != nil {
		logger.Error("[DeleteAllByServiceId] Delete replication controller failed:", err)
		return err
	}

	secrets, err := c.Secrets(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[DeleteAllByServiceId] List secrets failed:", err)
		return err
	}

	for _, i := range secrets.Items {
		name = i.ObjectMeta.Name
		logger.Debug("[DeleteAllByServiceId] Delete secret:", name)
		err = c.Secrets(api.NamespaceDefault).Delete(name)
		if err != nil {
			logger.Error("[DeleteAllByServiceId] Delete secret failed:", err)
			return err
		}
	}

	pvcs, err := c.PersistentVolumeClaims(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[DeleteAllByServiceId] List PersistentVolumeClaims failed:", err)
		return err
	}

	for _, i := range pvcs.Items {
		name = i.ObjectMeta.Name
		logger.Debug("[DeleteAllByServiceId] Delete PersistentVolumeClaims:", name)
		err = c.PersistentVolumeClaims(api.NamespaceDefault).Delete(name)
		if err != nil {
			logger.Error("[DeleteAllByServiceId] Delete PersistentVolumeClaims failed:", err)
			return err
		}
	}

	return nil
}

func (k *K8Fabricator) DeleteAllPersistentVolumes(creds K8sClusterCredential) error {

	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return err
	}

	pvList, err := c.PersistentVolumes().List(api.ListOptions{
		LabelSelector: labels.NewSelector(),
	})
	if err != nil {
		logger.Error("[DeleteAllPersistentVolumes] List PersistentVolume failed:", err)
		return err
	}

	var errorFound bool = false
	for _, i := range pvList.Items {
		name := i.ObjectMeta.Name
		logger.Debug("[DeleteAllPersistentVolumes] Delete PersistentVolume:", name)
		err = c.PersistentVolumes().Delete(name)
		if err != nil {
			logger.Error("[DeleteAllPersistentVolumes] Delete PersistentVolume: "+name+" failed!", err)
			errorFound = true
		}
	}

	if errorFound {
		return errors.New("Error on deleting PersistentVolume!")
	} else {
		return nil
	}
}

func (k *K8Fabricator) GetServiceVisibility(creds K8sClusterCredential, org, space, service_id string) ([]K8sServiceInfo, error) {
	logger.Info("[GetServiceVisibility]:", service_id)
	response := []K8sServiceInfo{}

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return response, err
	}

	services, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[GetServiceVisibility] Get services list failed:", err)
		return response, err
	}

	servicesPublicTags, err := k.ConsulApi.GetServicesListWithPublicTagStatus(creds.ConsulEndpoint)
	if err != nil {
		return response, err
	}

	response = createServiceInfoList(org, space, k.Domain, services.Items, servicesPublicTags)
	return response, nil
}

func (k *K8Fabricator) GetServicesVisibility(creds K8sClusterCredential, org, space string) ([]K8sServiceInfo, error) {
	logger.Info("[GetServicesVisibility] orgId:", org)
	response := []K8sServiceInfo{}

	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		logger.Error("[GetServicesVisibility] GetNewClient error", err)
		return response, err
	}
	selector, err := getSelectorForManagedByLabel()
	if err != nil {
		logger.Error("[GetServicesVisibility] GetSelectorForManagedByLabel error", err)
		return response, err
	}

	services, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[GetServicesVisibility] ListServices failed:", err)
		return response, err
	}

	servicesPublicTags, err := k.ConsulApi.GetServicesListWithPublicTagStatus(creds.ConsulEndpoint)
	if err != nil {
		logger.Error("[GetServicesVisibility]  ConsulApi.GetServicesListWithPublicTagStatus error", err)
		return response, err
	}

	response = createServiceInfoList(org, space, k.Domain, services.Items, servicesPublicTags)
	return response, nil
}

func (k *K8Fabricator) GetServices(creds K8sClusterCredential, org string) ([]api.Service, error) {
	logger.Info("[GetServices] orgId:", org)
	response := []api.Service{}

	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		logger.Error("[GetServices] GetNewClient error", err)
		return response, err
	}
	selector, err := getSelectorForManagedByLabel()
	if err != nil {
		logger.Error("[GetServices] GetSelectorForManagedByLabel error", err)
		return response, err
	}

	serviceList, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[GetServices] ListServices failed:", err)
		return response, err
	}
	return serviceList.Items, nil
}

func createServiceInfoList(org, space, domain string, services []api.Service, servicesPublicTags map[string]bool) []K8sServiceInfo {
	result := []K8sServiceInfo{}
	for _, service := range services {
		svc := K8sServiceInfo{
			ServiceId: service.ObjectMeta.Labels["service_id"],
			Org:       org,
			Space:     space,
			Name:      service.ObjectMeta.Name,
			TapPublic: readTapPublic(service.ObjectMeta.Name, servicesPublicTags),
		}

		for _, port := range service.Spec.Ports {
			svc.Uri = append(svc.Uri, GetServiceExternalAddress(domain, port))
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

func GetServiceExternalAddress(domain string, port api.ServicePort) string {
	return strings.ToLower(string(port.Protocol)) + "." + domain + ":" + strconv.Itoa(int(port.NodePort))
}

func GetServiceInternalHost(port api.ServicePort, service api.Service) string {
	return GetConsulServiceName(port, service) + ".service.consul"
}

func GetConsulServiceName(port api.ServicePort, service api.Service) string {
	portName := ""
	if port.Name != "" {
		portName = "-" + port.Name
	}
	return service.ObjectMeta.Name + portName
}

func (k *K8Fabricator) GetQuota(creds K8sClusterCredential, space string) (*api.ResourceQuotaList, error) {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return nil, err
	}

	return c.ResourceQuotas(api.NamespaceDefault).List(api.ListOptions{})
}

func (k *K8Fabricator) SetServicePublicVisibilityByServiceId(creds K8sClusterCredential, org, space, service_id string, shouldBePublic bool) ([]K8sServiceInfo, error) {
	logger.Info("[SetServicePublicVisibilityByServiceId] orgId, serviceId:", org, service_id)
	response := []K8sServiceInfo{}
	consulData := []consul.ConsulServiceParams{}

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return response, err
	}

	services, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[SetServicePublicVisibilityByServiceId] List services failed:", err)
		return []K8sServiceInfo{}, err
	}

	for _, service := range services.Items {
		svc := K8sServiceInfo{
			ServiceId: service_id,
			Org:       org,
			Space:     space,
			Name:      service.ObjectMeta.Name,
			TapPublic: shouldBePublic,
			Uri:       []string{},
		}

		for _, port := range service.Spec.Ports {
			if port.Protocol != api.ProtocolUDP {
				param := consul.ConsulServiceParams{
					Name:     GetConsulServiceName(port, service),
					IsPublic: shouldBePublic,
					Port:     port.NodePort,
				}
				consulData = append(consulData, param)

				svc.Uri = append(svc.Uri, GetServiceExternalAddress(k.Domain, port))
			}
		}

		err := k.ConsulApi.UpdateServiceTag(consulData, creds.ConsulEndpoint)
		if err != nil {
			logger.Error("[SetServicePublicVisibilityByServiceId] Consul UpdateServiceTag failed:", err)
			return response, err
		}
		response = append(response, svc)
	}
	return response, nil
}

func (k *K8Fabricator) GetClusterWorkers(creds K8sClusterCredential) ([]string, error) {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return []string{}, err
	}

	nodes, err := c.Nodes().List(api.ListOptions{})
	if err != nil {
		logger.Error("[GetClusterWorkers] GetNodes error:", err)
		return []string{}, err
	}

	workers := []string{}
	for _, i := range nodes.Items {
		workers = append(workers, i.Spec.ExternalID)
	}
	return workers, nil
}

func (k *K8Fabricator) ListReplicationControllers(creds K8sClusterCredential, space string) (*api.ReplicationControllerList, error) {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return nil, err
	}
	selector, err := getSelectorForManagedByLabel()
	if err != nil {
		return nil, err
	}

	return NewReplicationControllerManager(c).List(selector)
}

type PodStatus struct {
	PodName       string
	ServiceId     string
	Status        api.PodPhase
	StatusMessage string
}

func (k *K8Fabricator) GetPodsStateByServiceId(creds K8sClusterCredential, service_id string) ([]PodStatus, error) {
	result := []PodStatus{}

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return result, err
	}

	pods, err := c.Pods(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return result, err
	}

	for _, pod := range pods.Items {
		podStatus := PodStatus{
			pod.Name, service_id, pod.Status.Phase, pod.Status.Message,
		}
		result = append(result, podStatus)
	}
	return result, nil
}

func (k *K8Fabricator) GetPodsStateForAllServices(creds K8sClusterCredential) (map[string][]PodStatus, error) {
	result := map[string][]PodStatus{}

	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return result, err
	}
	selector, err := getSelectorForManagedByLabel()
	if err != nil {
		return result, err
	}

	pods, err := c.Pods(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return result, err
	}

	for _, pod := range pods.Items {
		service_id := pod.Labels["service_id"]
		if service_id != "" {
			podStatus := PodStatus{
				pod.Name, service_id, pod.Status.Phase, pod.Status.Message,
			}
			result[service_id] = append(result[service_id], podStatus)
		}
	}
	return result, nil
}

type ServiceCredential struct {
	Name  string
	Host  string
	Ports []api.ServicePort
}

func (k *K8Fabricator) GetServiceCredentials(creds K8sClusterCredential, space, service_id string) ([]ServiceCredential, error) {
	logger.Info("[GetServiceCredentials] serviceId:", service_id)
	result := []ServiceCredential{}

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return result, err
	}

	svcs, err := c.Services(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return result, err
	}
	if len(svcs.Items) < 1 {
		return result, errors.New("No services associated with the serviceId: " + service_id)
	}

	for _, svc := range svcs.Items {
		svcCred := ServiceCredential{}
		svcCred.Name = svc.Name
		svcCred.Host = GetServiceInternalHostByFirstTCPPort(svc)

		for _, p := range svc.Spec.Ports {
			svcCred.Ports = append(svcCred.Ports, p)
		}
		result = append(result, svcCred)
	}
	return result, nil
}

func GetServiceInternalHostByFirstTCPPort(service api.Service) string {
	for _, port := range service.Spec.Ports {
		if port.Protocol == api.ProtocolTCP {
			return GetServiceInternalHost(port, service)
		}
	}
	return ""
}

func (k *K8Fabricator) GetSecret(creds K8sClusterCredential, key string) (*api.Secret, error) {
	secret := &api.Secret{}
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return secret, err
	}
	result, err := c.Secrets(api.NamespaceDefault).Get(key)
	if err != nil {
		return secret, err
	}
	return result, nil
}

func (k *K8Fabricator) CreateSecret(creds K8sClusterCredential, secret api.Secret) error {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return err
	}
	_, err = c.Secrets(api.NamespaceDefault).Create(&secret)
	if err != nil {
		return err
	}
	return nil
}

func (k *K8Fabricator) DeleteSecret(creds K8sClusterCredential, key string) error {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return err
	}
	err = c.Secrets(api.NamespaceDefault).Delete(key)
	if err != nil {
		return err
	}
	return nil
}

func (k *K8Fabricator) UpdateSecret(creds K8sClusterCredential, secret api.Secret) error {
	c, err := k.KubernetesClient.GetNewClient(creds)
	if err != nil {
		return err
	}
	_, err = c.Secrets(api.NamespaceDefault).Update(&secret)
	if err != nil {
		return err
	}
	return nil
}

type PodEnvs struct {
	RcName     string
	Containers []ContainerSimple
}

type ContainerSimple struct {
	Name string
	Envs map[string]string
}

func (k *K8Fabricator) GetAllPodsEnvsByServiceId(creds K8sClusterCredential, space, service_id string) ([]PodEnvs, error) {
	logger.Info("[GetEnvFromReplicationControllerByServiceIdLabel] serviceId:", service_id)
	result := []PodEnvs{}

	c, selector, err := k.getKubernetesClientWithServiceIdSelector(creds, service_id)
	if err != nil {
		return result, err
	}

	rcs, err := NewReplicationControllerManager(c).List(selector)

	if err != nil {
		return result, err
	}
	if len(rcs.Items) < 1 {
		return result, errors.New("No replication controllers associated with the service: " + service_id)
	}

	secrets, err := c.Secrets(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		logger.Error("[GetEnvFromReplicationControllerByServiceIdLabel] List secrets failed:", err)
		return result, err
	}

	for _, rc := range rcs.Items {
		pod := PodEnvs{}
		pod.RcName = rc.Name
		pod.Containers = []ContainerSimple{}

		for _, container := range rc.Spec.Template.Spec.Containers {
			simpelContainer := ContainerSimple{}
			simpelContainer.Name = container.Name
			simpelContainer.Envs = map[string]string{}

			for _, env := range container.Env {
				if env.Value == "" {
					logger.Debug("Empty env value, searching env variable in secrets")
					simpelContainer.Envs[env.Name] = findSecretValue(secrets, envNameToSecretKey(env.Name))
				} else {
					simpelContainer.Envs[env.Name] = env.Value
				}

			}
			pod.Containers = append(pod.Containers, simpelContainer)
		}
		result = append(result, pod)
	}
	return result, nil
}

func envNameToSecretKey(env_name string) string {
	lower_case_string := strings.ToLower(env_name)
	return strings.Replace(lower_case_string, "_", "-", -1)
}

func findSecretValue(secrets *api.SecretList, secret_key string) string {
	for _, i := range secrets.Items {
		for key, value := range i.Data {
			if key == secret_key {
				return string((value))
			}
		}
	}
	logger.Info("Secret key not found: ", secret_key)
	return ""
}

func (k *K8Fabricator) getKubernetesClientWithServiceIdSelector(creds K8sClusterCredential, serviceId string) (KubernetesClient, labels.Selector, error) {
	selector, err := getSelectorForServiceIdLabel(serviceId)
	if err != nil {
		return nil, selector, err
	}

	c, err := k.KubernetesClient.GetNewClient(creds)
	return c, selector, err
}

func getSelectorForServiceIdLabel(serviceId string) (labels.Selector, error) {
	selector := labels.NewSelector()
	managedByReq, err := labels.NewRequirement("managed_by", labels.EqualsOperator, sets.NewString("TAP"))
	if err != nil {
		return selector, err
	}
	serviceIdReq, err := labels.NewRequirement("service_id", labels.EqualsOperator, sets.NewString(serviceId))
	if err != nil {
		return selector, err
	}
	return selector.Add(*managedByReq, *serviceIdReq), nil
}

func getSelectorForManagedByLabel() (labels.Selector, error) {
	selector := labels.NewSelector()
	managedByReq, err := labels.NewRequirement("managed_by", labels.EqualsOperator, sets.NewString("TAP"))
	if err != nil {
		return selector, err
	}
	return selector.Add(*managedByReq), nil
}
