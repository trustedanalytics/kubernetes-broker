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
package catalog

import (
	"encoding/json"
	"sync"

	"github.com/nu7hatch/gouuid"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

type DynamicService struct {
	ServiceName        string                 `json:"serviceName"`
	PlanName           string                 `json:"planName"`
	IsPlanFree         bool                   `json:"isPlanFree"`
	Containers         []api.Container        `json:"containers"`
	ServicePorts       []api.ServicePort      `json:"servicePorts"`
	CredentialMappings map[string]interface{} `json:"credentialMappings"`
}

const templateCatalogPath string = "./template/"
const templateServiceMetaInternalId string = "service"
const templatePlanMetaInternalId string = "simple"

var catalog_mutex sync.RWMutex

func CreateDynamicService(dynamicService DynamicService) (KubernetesBlueprint, PlanMetadata, ServiceMetadata, error) {
	//todo validation for service/plan name? (no spaces etc.)?

	logger.Info("[CreateDynamicService] Creating dynamic service with params:", dynamicService)
	result := KubernetesBlueprint{}

	plan, err := getDynamicPlanMetadata(dynamicService)
	if err != nil {
		return result, plan, ServiceMetadata{}, err
	}

	service, err := getDynamicServiceMetadata(dynamicService, plan)
	if err != nil {
		return result, plan, service, err
	}

	blueprintTemplate, err := GetKubernetesBlueprint(
		templateCatalogPath, templateServiceMetaInternalId, templatePlanMetaInternalId, "",
	)
	if err != nil {
		return result, plan, service, err
	}

	componentTemplate, err := CreateKubernetesComponentFromBlueprint(blueprintTemplate, false)
	if err != nil {
		return result, plan, service, err
	}

	result, err = getParsedKubernetesBlueprint(*componentTemplate, blueprintTemplate, dynamicService)

	// TODO now save "result" in postgres as json - we will be using it to initiate new services
	// TODO save it in postgres! and then add to catalog! -> after save, reload catalog
	logger.Info("We should save created dynamic service result to DB now!", plan, service, result)
	return result, plan, service, err
}

func RegisterOfferingInCatalog(service ServiceMetadata, blueprint KubernetesBlueprint) {
	//todo THIS is not persisted registration - we need to save it in DB to keep it persisted
	// first add to catalog
	catalog_mutex.Lock()
	GLOBAL_SERVICES_METADATA.Services = append(GLOBAL_SERVICES_METADATA.Services, service)
	// then register dynamic blueprint
	TEMP_DYNAMIC_BLUEPRINTS[service.Id] = blueprint
	catalog_mutex.Unlock()
}

func UnregisterOfferingFromCatalog(service ServiceMetadata) {
	// TODO no persited version - change it together with RegisterOfferingInCatalog()
	// first remove from catalog
	catalog_mutex.Lock()

	for i, svc := range GLOBAL_SERVICES_METADATA.Services {
		if svc.Name == service.Name {
			GLOBAL_SERVICES_METADATA.Services = append(GLOBAL_SERVICES_METADATA.Services[:i], GLOBAL_SERVICES_METADATA.Services[i+1:]...)
			break
		}
	}

	// then unregister dynamic blueprint
	delete(TEMP_DYNAMIC_BLUEPRINTS, service.Id)
	catalog_mutex.Unlock()
}

func getDynamicPlanMetadata(dynamicService DynamicService) (PlanMetadata, error) {
	planId, err := uuid.NewV4()
	if err != nil {
		return PlanMetadata{}, err
	}

	return PlanMetadata{
		Id:          planId.String(),
		Name:        dynamicService.PlanName,
		Description: dynamicService.PlanName,
		Free:        dynamicService.IsPlanFree,
		InternalId:  "dynamic" + dynamicService.PlanName,
	}, nil
}

func getDynamicServiceMetadata(dynamicService DynamicService, plan PlanMetadata) (ServiceMetadata, error) {
	serviceId, err := uuid.NewV4()
	if err != nil {
		return ServiceMetadata{}, err
	}

	return ServiceMetadata{
		Id:          serviceId.String(),
		Name:        dynamicService.ServiceName,
		Description: dynamicService.ServiceName,
		Bindable:    true,
		Tags:        []string{dynamicService.ServiceName},
		Plans:       []PlanMetadata{plan},
		InternalId:  "dynamic" + dynamicService.ServiceName,
	}, nil
}

func getParsedKubernetesBlueprint(componentTemplate KubernetesComponent, blueprintTemplate KubernetesBlueprint, dynamicService DynamicService) (KubernetesBlueprint, error) {
	result := KubernetesBlueprint{}
	deploymentJson, err := getParsedDeploymentJson(*componentTemplate.Deployments[0], dynamicService.Containers)
	if err != nil {
		return result, err
	}
	result.DeploymentJson = append(result.DeploymentJson, deploymentJson)

	serviceJson, err := getParsedServiceJson(*componentTemplate.Services[0], dynamicService.ServicePorts)
	if err != nil {
		return result, err
	}
	result.ServiceJson = append(result.ServiceJson, serviceJson)

	result.ServiceAcccountJson = blueprintTemplate.ServiceAcccountJson

	//todo remove it from dynamicService - it supposed to be build dynamicly from container envs
	jsonCredMapping, err := json.Marshal(dynamicService.CredentialMappings)
	if err != nil {
		logger.Error("[CreateDynamicService] Marshaling credential mapping error!", err)
		return result, err
	}
	result.CredentialsMapping = string(jsonCredMapping)
	return result, nil
}

func getParsedDeploymentJson(template extensions.Deployment, conteinersToParse []api.Container) (string, error) {
	template.Spec.Template.Spec.Containers = getParsedContainers(template.Spec.Template.Spec.Containers[0], conteinersToParse)
	jsonRc, err := json.Marshal(template)
	if err != nil {
		logger.Error("Marshaling deployment error!", err)
		return "", err
	}
	return string(jsonRc), nil
}

func getParsedContainers(template api.Container, conteinersToParse []api.Container) []api.Container {
	configuredContainers := []api.Container{}
	for _, userContainer := range conteinersToParse {
		container := template
		container.Ports = userContainer.Ports
		container.Image = userContainer.Image
		container.Name = userContainer.Name //todo validate name value!:
		//Respond500: reason: error  ReplicationController "x2f7af44d0c614" is invalid: spec.template.spec.containers[0].name: invalid value 'postgres:9.4', Details: must be a DNS label (at most 63 characters, matching regex [a-z0-9]([-a-z0-9]*[a-z0-9])?): e.g. "my-name"
		container.Env = append(container.Env, userContainer.Env...)
		configuredContainers = append(configuredContainers, container)
	}
	return configuredContainers
}

func getParsedServiceJson(template api.Service, servicePortsToParse []api.ServicePort) (string, error) {
	template.Spec.Ports = servicePortsToParse
	jsonSvc, err := json.Marshal(template)
	if err != nil {
		logger.Error("[getParsedServiceJson] Marshaling service error!", err)
		return "", err
	}
	return string(jsonSvc), nil
}
