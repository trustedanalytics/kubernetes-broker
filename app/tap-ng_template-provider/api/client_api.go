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

type TapNgTemplateProviderClient interface {
	GetCatalog() (catalog.ServicesMetadata, error)
	CreateAndRegisterDynamicService(dynamicService DynamicServiceRequest) error
	DeleteAndUnregisterDynamicService(dynamicService DynamicServiceRequest) error
}

type TapNgClient struct {
	Address  string
	Username string
	Password string
	Client   *http.Client
}

var logger = logger_wrapper.InitLogger("api")

func NewTapNgTemplateProviderClient(address, username, password string) (*TapNgClient, error) {
	client, _, err := brokerHttp.GetHttpClientWithBasicAuth()
	if err != nil {
		return nil, err
	}
	return &TapNgClient{address, username, password, client}, nil
}

func (t *TapNgClient) GetCatalog() (catalog.ServicesMetadata, error) {
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

func (t *TapNgClient) GenerateParsedTemplate(request GenerateParsedTemplateRequest) error {
	url := t.Address + "/catalog/parsed"
	body, err := json.Marshal(request)
	if err != nil {
		logger.Error("GenerateParsedTemplate error:", err)
		return err
	}

	_, _, err = brokerHttp.RestPOST(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	return err
}

func (t *TapNgClient) CreateAndRegisterDynamicService(dynamicService DynamicServiceRequest) error {
	url := t.Address + "/dynamicservice"
	body, err := json.Marshal(dynamicService)
	if err != nil {
		logger.Error("CreateAndRegisterDynamicService error:", err)
		return err
	}

	_, _, err = brokerHttp.RestPUT(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	return err
}

func (t *TapNgClient) DeleteAndUnregisterDynamicService(dynamicService DynamicServiceRequest) error {
	url := t.Address + "/dynamicservice"
	body, err := json.Marshal(dynamicService)
	if err != nil {
		logger.Error("DeleteAndUnregisterDynamicService error:", err)
		return err
	}

	_, _, err = brokerHttp.RestDELETE(url, string(body), &brokerHttp.BasicAuth{t.Username, t.Password}, t.Client)
	return err
}
