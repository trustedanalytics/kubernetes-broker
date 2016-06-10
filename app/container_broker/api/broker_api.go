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

	"github.com/trustedanalytics/kubernetes-broker/app/template_repository/api"
	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/logger"
	"github.com/trustedanalytics/kubernetes-broker/state"
	"github.com/trustedanalytics/kubernetes-broker/util"
)

type Config struct {
	StateService          state.StateService
	KubernetesApi         k8s.KubernetesApi
	TemplateRepository    api.TemplateRepository
	K8sClusterCredentials k8s.K8sClusterCredentials
}

type Context struct{}

var BrokerConfig *Config
var logger = logger_wrapper.InitLogger("api")

func (c *Context) CheckBrokerConfig(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	if BrokerConfig == nil {
		util.Respond500(rw, errors.New("BrokerConfig not set!"))
	}
	next(rw, req)
}

type ServiceInstanceRequest struct {
	Uuid       string `json:"uuid"`
	TemplateId string `json:"templateId"`
	OrgId      string `json:"orgId"`
	SpaceId    string `json:"spaceId"`
}

func (c *Context) CreateServiceInstance(rw web.ResponseWriter, req *web.Request) {
	req_json, templateRequest, err := ParseServiceInstanceRequest(req)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "IN_PROGRESS_STARTED", nil)
	template, err := BrokerConfig.TemplateRepository.GenerateParsedTemplate(templateRequest)
	if err != nil {
		BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "FAILED", err)
		util.Respond500(rw, err)
		return
	}
	BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "IN_PROGRESS_BLUEPRINT_OK", nil)

	_, err = BrokerConfig.KubernetesApi.FabricateService(BrokerConfig.K8sClusterCredentials, req_json.SpaceId,
		req_json.Uuid, "", BrokerConfig.StateService, &template.Body)
	if err != nil {
		BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "FAILED", err)
		util.Respond500(rw, err)
		return
	}

	BrokerConfig.KubernetesApi.CreateJobsByType(BrokerConfig.K8sClusterCredentials, template.Hooks, req_json.Uuid,
		catalog.JobTypeOnCreateInstance, BrokerConfig.StateService)

	BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "IN_PROGRESS_KUBERNETES_OK", nil)
	util.WriteJson(rw, "", http.StatusAccepted)
}

func (c *Context) DeleteServiceInstance(rw web.ResponseWriter, req *web.Request) {
	req_json, templateRequest, err := ParseServiceInstanceRequest(req)
	if err != nil {
		util.Respond500(rw, err)
		return
	}

	template, err := BrokerConfig.TemplateRepository.GenerateParsedTemplate(templateRequest)
	if err != nil {
		BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "Delete FAILED", err)
		util.Respond500(rw, err)
		return
	}
	BrokerConfig.KubernetesApi.CreateJobsByType(BrokerConfig.K8sClusterCredentials, template.Hooks, req_json.Uuid,
		catalog.JobTypeOnDeleteInstance, BrokerConfig.StateService)

	err = BrokerConfig.KubernetesApi.DeleteAllByServiceId(BrokerConfig.K8sClusterCredentials, req_json.Uuid)
	if err != nil {
		BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "Delete FAILED", err)
		util.Respond500(rw, err)
		return
	}

	BrokerConfig.StateService.NotifyCatalog(req_json.Uuid, "Delete SUCCESS", err)
	util.WriteJson(rw, "", http.StatusOK)
}

func ParseServiceInstanceRequest(req *web.Request) (ServiceInstanceRequest, api.GenerateParsedTemplateRequest, error) {
	req_json := ServiceInstanceRequest{}
	err := util.ReadJson(req, &req_json)
	templateRequest := api.GenerateParsedTemplateRequest{
		Uuid:       req_json.Uuid,
		TemplateId: req_json.TemplateId,
		OrgId:      req_json.OrgId,
		SpaceId:    req_json.SpaceId,
	}
	return req_json, templateRequest, err
}

func (c *Context) Error(rw web.ResponseWriter, r *web.Request, err interface{}) {
	logger.Error("Respond500: reason: error ", err)
	rw.WriteHeader(http.StatusInternalServerError)
}
