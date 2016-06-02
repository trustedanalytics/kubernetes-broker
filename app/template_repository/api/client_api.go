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
	"encoding/json"
	"net/http"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
	"github.com/trustedanalytics/kubernetes-broker/logger"
)

type TemplateRepository interface {
	GetCatalog() (catalog.ServicesMetadata, error)
	GenerateParsedTemplate(request GenerateParsedTemplateRequest) (catalog.KubernetesComponent, error)
	CreateAndRegisterDynamicService(dynamicService DynamicServiceRequest) error
	DeleteAndUnregisterDynamicService(dynamicService DynamicServiceRequest) error
}

type TemplateRepositoryConnector struct {
	Address  string
	Username string
	Password string
	Client   *http.Client
}

var logger = logger_wrapper.InitLogger("api")

func NewTemplateRepositoryBasicAuth(address, username, password string) (*TemplateRepositoryConnector, error) {
	client, _, err := brokerHttp.GetHttpClientWithBasicAuth()
	if err != nil {
		return nil, err
	}
	return &TemplateRepositoryConnector{address, username, password, client}, nil
}

func NewTemplateRepositoryCa(address, username, password, certPemFile, keyPemFile, caPemFile string) (*TemplateRepositoryConnector, error) {
	client, _, err := brokerHttp.GetHttpClientWithCertAndCaFromFile(certPemFile, keyPemFile, caPemFile)
	if err != nil {
		return nil, err
	}
	return &TemplateRepositoryConnector{address, username, password, client}, nil
}

func (t *TemplateRepositoryConnector) GetCatalog() (catalog.ServicesMetadata, error) {
	url := t.Address + "/catalog"
	_, body, err := brokerHttp.RestGET(url, &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)

	services := catalog.ServicesMetadata{}
	err = json.Unmarshal(body, &services)
	if err != nil {
		logger.Error("GetCatalog error:", err)
		return services, err
	}
	return services, nil
}

func (t *TemplateRepositoryConnector) GenerateParsedTemplate(request GenerateParsedTemplateRequest) (catalog.KubernetesComponent, error) {
	component := catalog.KubernetesComponent{}

	url := t.Address + "/catalog/parsed"
	body, err := json.Marshal(request)
	if err != nil {
		logger.Error("GenerateParsedTemplate marshall request error:", err)
		return component, err
	}

	_, body, err = brokerHttp.RestPOST(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	err = json.Unmarshal(body, &component)
	if err != nil {
		logger.Error("GenerateParsedTemplate unmarshall response error:", err)
		return component, err
	}
	return component, nil
}

func (t *TemplateRepositoryConnector) CreateAndRegisterDynamicService(dynamicService DynamicServiceRequest) error {
	url := t.Address + "/dynamicservice"
	body, err := json.Marshal(dynamicService)
	if err != nil {
		logger.Error("CreateAndRegisterDynamicService error:", err)
		return err
	}

	_, _, err = brokerHttp.RestPUT(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	return err
}

func (t *TemplateRepositoryConnector) DeleteAndUnregisterDynamicService(dynamicService DynamicServiceRequest) error {
	url := t.Address + "/dynamicservice"
	body, err := json.Marshal(dynamicService)
	if err != nil {
		logger.Error("DeleteAndUnregisterDynamicService error:", err)
		return err
	}

	_, _, err = brokerHttp.RestDELETE(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	return err
}
