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
package consul

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"

	"github.com/trustedanalytics/kubernetes-broker/logger"
)

var logger = logger_wrapper.InitLogger("consul")

const publicTag string = "Public"

type ConsulService interface {
	UpdateServiceTag(params []ConsulServiceParams, endpoint string) error
	GetServicesListWithPublicTagStatus(endpoint string) (map[string]bool, error)
}

type ConsulConnector struct{}

type ConsulServiceParams struct {
	Name     string
	IsPublic bool
	Port     int
}

func NewConsulClient(endpoint string) *api.Client {
	config := api.DefaultConfig()
	config.Address = strings.Replace(endpoint, "http://", "", -1)
	logger.Debug("Consul Client config:", config)
	// currently we not use any authorization
	config.HttpAuth = nil

	client, err := api.NewClient(config)
	if err != nil {
		logger.Panic("Could not connect to Consul!", err)
	}
	return client
}

func (c *ConsulConnector) GetServicesListWithPublicTagStatus(endpoint string) (map[string]bool, error) {
	client := NewConsulClient(endpoint)
	result := map[string]bool{}

	services, err := client.Agent().Services()
	if err != nil {
		return result, err
	}

	for _, service := range services {
		result[service.Service] = isPubicTagIncluded(service.Tags)
	}
	return result, nil
}

func isPubicTagIncluded(tags []string) bool {
	for _, tag := range tags {
		if publicTag == tag {
			return true
		}
	}
	return false
}

func (c *ConsulConnector) UpdateServiceTag(params []ConsulServiceParams, endpoint string) error {
	client := NewConsulClient(endpoint)

	for _, param := range params {
		var asr *api.AgentServiceRegistration
		serviceFound := false

		services, err := client.Agent().Services()
		if err != nil {
			return err
		}

		if len(services) == 0 {
			return errors.New(fmt.Sprintf("Can't find service: %s in consul catalog!", param.Name))
		} else {
			asr, serviceFound = prepareUpdatedService(services, param)
			if !serviceFound {
				return errors.New(fmt.Sprintf("Can't find service: %s with port: %d in consul catalog!", param.Name, param.Port))
			}
		}

		logger.Debug("[UpdateServiceTag] Service tags before update: ", asr.Tags)
		tags := []string{}
		for _, tag := range asr.Tags {
			if tag != publicTag {
				tags = append(tags, tag)
			}
		}
		if param.IsPublic {
			tags = append(tags, publicTag)
		}
		asr.Tags = tags

		logger.Debug("[UpdateServiceTag] Consul re-registration service param:", asr)
		err = client.Agent().ServiceRegister(asr)
		if err != nil {
			return err
		}
	}
	return nil
}

func prepareUpdatedService(services map[string]*api.AgentService, param ConsulServiceParams) (*api.AgentServiceRegistration, bool) {
	var asr *api.AgentServiceRegistration
	serviceFound := false

servicesLoop:
	for _, service := range services {
		if service.Service == param.Name && service.Port == param.Port {
			logger.Info(fmt.Sprintf("[ConsulConnector] Preparing service %s with port %d for update...", param.Name, param.Port))
			asr = &api.AgentServiceRegistration{
				ID:                service.ID,
				Name:              service.Service,
				Port:              service.Port,
				Address:           service.Address,
				Tags:              service.Tags,
				EnableTagOverride: true,
			}
			serviceFound = true
			break servicesLoop
		}
	}
	return asr, serviceFound
}
