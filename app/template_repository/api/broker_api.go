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

package api

import (
	"errors"
	"net/http"

	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/state"
	"github.com/trustedanalytics/kubernetes-broker/util"
)

type Config struct {
	StateService state.StateService
}

type Context struct{}

var BrokerConfig *Config

func (c *Context) CheckBrokerConfig(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	if BrokerConfig == nil {
		util.Respond500(rw, errors.New("brokerConfig not set!"))
	}
	next(rw, req)
}

func (c *Context) Catalog(rw web.ResponseWriter, req *web.Request) {
	services := catalog.GetAvailableServicesMetadata()
	util.WriteJson(rw, services, http.StatusOK)
}

type GenerateParsedTemplateRequest struct {
	Uuid                string `json:"uuid"`
	ServiceMetadataUuid string `json:"serviceMetadataUuid"`
	PlanMetadataUuid    string `json:"planMetadataUuid"`
	OrgId               string `json:"orgId"`
	SpaceId             string `json:"spaceId"`
}

func (c *Context) GenerateParsedTemplate(rw web.ResponseWriter, req *web.Request) {
	req_json := GenerateParsedTemplateRequest{}

	err := util.ReadJson(req, &req_json)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	svcMetadata, planMetadata, err := catalog.WhatToCreateByServiceAndPlanId(req_json.ServiceMetadataUuid, req_json.PlanMetadataUuid)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	component, err := catalog.GetParsedKubernetesComponent(catalog.CatalogPath, req_json.Uuid,
		req_json.OrgId, req_json.SpaceId, svcMetadata, planMetadata)
	if err != nil {
		util.Respond500(rw, err)
		return
	}
	util.WriteJson(rw, component, http.StatusOK)
}

type DynamicServiceRequest struct {
	DynamicService catalog.DynamicService `json:"dynamicService"`
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
	util.WriteJson(rw, "", http.StatusCreated)
}

func (c *Context) DeleteAndUnregisterDynamicService(rw web.ResponseWriter, req *web.Request) {
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
	util.WriteJson(rw, "", http.StatusNoContent)

}

func (c *Context) Error(rw web.ResponseWriter, r *web.Request, err interface{}) {
	logger.Error("Respond500: reason: error ", err)
	rw.WriteHeader(http.StatusInternalServerError)
}
