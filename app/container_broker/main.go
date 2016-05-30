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
	"net/http"
	"os"

	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/app/container_broker/api"
	templateRepositoryApi "github.com/trustedanalytics/kubernetes-broker/app/template_repository/api"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/logger"
	"github.com/trustedanalytics/kubernetes-broker/state"
)

type appHandler func(web.ResponseWriter, *web.Request) error

var logger = logger_wrapper.InitLogger("main")

func main() {
	initServices()

	r := web.New(api.Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*api.Context).CheckBrokerConfig)
	r.Middleware((*api.Context).BasicAuthorizeMiddleware)

	r.Put("/service", (*api.Context).CreateServiceInstance)
	r.Delete("/service/:instance_id", (*api.Context).DeleteServiceInstance)

	address := os.Getenv("CONTAINER_BROKER_ADDRESS")
	logger.Info("Will listen on:", address)
	err := http.ListenAndServe(address, r)
	if err != nil {
		logger.Critical("Couldn't serve app on address: ", address, " Application will be closed now.")
	}
}

func initServices() {
	templateRepositoryConnector, err := templateRepositoryApi.NewTemplateRepository(
		os.Getenv("TEMPLATE_REPOSITORY_ADDRESS"),
		os.Getenv("TEMPLATE_REPOSITORY_USER"),
		os.Getenv("TEMPLATE_REPOSITORY_PASS"),
	)
	if err != nil {
		logger.Fatal("Can't connect with TAP-NG template provider!", err)
	}

	api.BrokerConfig = &api.Config{}
	api.BrokerConfig.StateService = &state.StateMemoryService{}
	api.BrokerConfig.KubernetesApi = k8s.NewK8Fabricator()
	api.BrokerConfig.TemplateRepository = templateRepositoryConnector
	api.BrokerConfig.K8sClusterCredential = k8s.K8sClusterCredentials{
		Server:    os.Getenv("K8S_API_ADDRESS"),
		Username:  os.Getenv("K8S_API_USERNAME"),
		Password:  os.Getenv("K8S_API_PASSWORD"),
		AdminCert: os.Getenv("K8S_API_CERT_PEM_STRING"),
		AdminKey:  os.Getenv("K8S_API_KEY_PEM_STRING"),
		CaCert:    os.Getenv("K8S_API_CA_PEM_STRING"),
	}
}
