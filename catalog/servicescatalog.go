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
	"errors"
	"io/ioutil"

	"github.com/trustedanalytics/kubernetes-broker/logger"
)

type ServicesMetadata struct {
	Services []ServiceMetadata `json:"services"`
}

type ServiceMetadata struct {
	Id          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Bindable    bool           `json:"bindable"`
	Tags        []string       `json:"tags"`
	Plans       []PlanMetadata `json:"plans"`
	InternalId  string         `json:"-"`
}

type PlanMetadata struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Free        bool   `json:"free"`
	InternalId  string `json:"-"`
}

var CatalogPath string = "./catalogData/"
var logger = logger_wrapper.InitLogger("catalog")

func WhatToCreateByServiceAndPlanId(service_id, plan_id string) (ServiceMetadata, PlanMetadata, error) {
	//todo we need to check in postgres for dynamic services - we need to only add result to GetAvailableServicesMetadata()
	svcMeta, err := GetServiceMetadataByServiceId(service_id)
	if err != nil {
		logger.Error(err)
		return ServiceMetadata{}, PlanMetadata{}, err
	}
	logger.Info("Found service:", svcMeta)
	planMeta, err := GetPlanMetadataByPlanIdInServiceMetadata(svcMeta, plan_id)
	if err != nil {
		logger.Error(err)
		return svcMeta, PlanMetadata{}, err
	}
	logger.Info("Found plan:", planMeta)

	return svcMeta, planMeta, nil
}

func GetPlanMetadataByPlanIdInServiceMetadata(svc_metadata ServiceMetadata, plan_id string) (PlanMetadata, error) {
	for _, plan := range svc_metadata.Plans {
		if plan.Id == plan_id {
			return plan, nil
		}
	}
	return PlanMetadata{}, errors.New("No such plan by ID: " + plan_id)
}

func GetServiceMetadataByServiceId(service_id string) (ServiceMetadata, error) {
	for _, svc := range GetAvailableServicesMetadata().Services {
		if svc.Id == service_id {
			return svc, nil
		}
	}
	return ServiceMetadata{}, errors.New("No such service by ID: " + service_id)
}

func CheckIfServiceAlreadyExist(serviceName string) bool {
	for _, svc := range GetAvailableServicesMetadata().Services {
		if svc.Name == serviceName {
			return true
		}
	}
	return false
}

func GetServiceByName(serviceName string) (ServiceMetadata, error) {
	for _, svc := range GetAvailableServicesMetadata().Services {
		if svc.Name == serviceName {
			return svc, nil
		}
	}
	return ServiceMetadata{}, errors.New("service not exist!")
}

// add mutex... or return a deep copy (prefered).
var GLOBAL_SERVICES_METADATA *ServicesMetadata

func GetAvailableServicesMetadata() ServicesMetadata {
	if GLOBAL_SERVICES_METADATA != nil {
		logger.Debug("GetAvailableServicesMetadata - already exists.")
		return *GLOBAL_SERVICES_METADATA
	} else {
		logger.Debug("GetAvailableServicesMetadata - need to parse catalog/ directory.")
		services_metadata := ServicesMetadata{}
		catalog_file_info, err := ioutil.ReadDir(CatalogPath)
		if err != nil {
			logger.Panic(err)
		}
		for _, svcdir := range catalog_file_info {
			if svcdir.IsDir() {
				svcdirname := CatalogPath + svcdir.Name()
				logger.Debug(" => ", svcdir.Name(), svcdirname)

				plans_file_info, err := ioutil.ReadDir(svcdirname)
				if err != nil {
					logger.Panic(err)
				}
				var svc_meta ServiceMetadata
				var plan_metas []PlanMetadata
				for _, plandir := range plans_file_info {
					plan_dir_full_name := svcdirname + "/" + plandir.Name()
					var plan_meta PlanMetadata
					if plandir.IsDir() {
						logger.Debug(" ====> ", plandir.Name(), plan_dir_full_name)
						plans_content_file_info, err := ioutil.ReadDir(plan_dir_full_name)
						if err != nil {
							logger.Panic(err)
						}

						for _, plan_details := range plans_content_file_info {
							plan_details_dir_full_name := plan_dir_full_name + "/" + plan_details.Name()
							if plan_details.IsDir() {
								logger.Debug("Skipping directory:", plan_details_dir_full_name)
							} else if plan_details.Name() == "plan.json" {
								logger.Debug(" -----------> PLAN.JSON: ", plan_details.Name(), plan_details_dir_full_name)
								plan_metadata_file_content, err := ioutil.ReadFile(plan_details_dir_full_name)
								if err != nil {
									logger.Fatal("Error reading file: ", plan_details_dir_full_name, err)
								}
								b := []byte(plan_metadata_file_content)
								err = json.Unmarshal(b, &plan_meta)
								if err != nil {
									logger.Fatal("Error parsing json from file: ", plan_details_dir_full_name, err)
								}
								logger.Debug("PLAN.JSON parsed as: ", plan_meta)
								plan_meta.InternalId = plandir.Name()
								plan_metas = append(plan_metas, plan_meta)
							} else {
								logger.Debug(" -----------> ", plan_details.Name(), plan_details_dir_full_name)
							}

						}

					} else if plandir.Name() == "service.json" {
						logger.Debug(" ----> SERVICE.JSON: ", plandir.Name())
						// LOAD SERVICE METADATA

						svc_metadata_file_content, err := ioutil.ReadFile(plan_dir_full_name)
						if err != nil {
							logger.Fatal("Error reading file: ", plan_dir_full_name, err)
						}
						b := []byte(svc_metadata_file_content)
						err = json.Unmarshal(b, &svc_meta)
						if err != nil {
							logger.Fatal("Error parsing json from file: ", plan_dir_full_name, err)
						}
						logger.Debug("SERVICE.JSON parsed as: ", svc_meta)

					} else {
						logger.Debug("Skipping file: ", plan_dir_full_name)
					}
				}
				svc_meta.InternalId = svcdir.Name()
				svc_meta.Plans = plan_metas
				services_metadata.Services = append(services_metadata.Services, svc_meta)

			}
		}

		logger.Debug("PARSED: services_metadata: ", services_metadata)
		GLOBAL_SERVICES_METADATA = &services_metadata
		return *GLOBAL_SERVICES_METADATA
	}
}
