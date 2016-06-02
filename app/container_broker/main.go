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
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

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
	r := web.New(api.Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*api.Context).CheckBrokerConfig)
	r.Middleware((*api.Context).BasicAuthorizeMiddleware)

	r.Put("/service", (*api.Context).CreateServiceInstance)
	r.Delete("/service/:instance_id", (*api.Context).DeleteServiceInstance)

	port := os.Getenv("CONTAINER_BROKER_PORT")
	logger.Info("Will listen on:", port)

	isSSLEnabled, err := strconv.ParseBool(os.Getenv("CONTAINER_BROKER_SSL_ACTIVE"))
	if err != nil {
		logger.Critical("Couldn't read env CONTAINER_BROKER_SSL_ACTIVE!", err)
	}

	initServices(isSSLEnabled)

	if isSSLEnabled {
		err = http.ListenAndServeTLS(":"+port, os.Getenv("CONTAINER_BROKER_SSL_CERT_FILE_LOCATION"),
			os.Getenv("CONTAINER_BROKER_SSL_KEY_FILE_LOCATION"), r)
	} else {
		err = http.ListenAndServe(":"+port, r)
	}

	if err != nil {
		logger.Critical("Couldn't serve app on port:", port, " Error:", err)
	}
}

func initServices(isSSLEnabled bool) {
	var templateRepositoryConnector *templateRepositoryApi.TemplateRepositoryConnector
	var err error

	if isSSLEnabled {
		templateRepositoryConnector, err = templateRepositoryApi.NewTemplateRepositoryCa(
			"https://localhost:"+os.Getenv("TEMPLATE_REPOSITORY_PORT"),
			os.Getenv("TEMPLATE_REPOSITORY_USER"),
			os.Getenv("TEMPLATE_REPOSITORY_PASS"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_CERT_FILE_LOCATION"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_KEY_FILE_LOCATION"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_CA_FILE_LOCATION"),
		)
	} else {
		templateRepositoryConnector, err = templateRepositoryApi.NewTemplateRepositoryBasicAuth(
			"https://localhost:"+os.Getenv("TEMPLATE_REPOSITORY_PORT"),
			os.Getenv("TEMPLATE_REPOSITORY_USER"),
			os.Getenv("TEMPLATE_REPOSITORY_PASS"),
		)
	}

	if err != nil {
		logger.Fatal("Can't connect with TAP-NG template provider!", err)
	}

	api.BrokerConfig = &api.Config{}
	api.BrokerConfig.StateService = &state.StateMemoryService{}
	api.BrokerConfig.KubernetesApi = k8s.NewK8Fabricator()
	api.BrokerConfig.TemplateRepository = templateRepositoryConnector
	api.BrokerConfig.K8sClusterCredential = k8s.K8sClusterCredentials{
		Server:   os.Getenv("K8S_API_ADDRESS"),
		Username: os.Getenv("K8S_API_USERNAME"),
		Password: os.Getenv("K8S_API_PASSWORD"),
	}

	isKubernetesSSLEnabled, err := strconv.ParseBool(os.Getenv("KUBE_SSL_ACTIVE"))
	if err != nil {
		logger.Critical("Couldn't read env KUBE_SSL_ACTIVE!", err)
	}

	if isKubernetesSSLEnabled {
		cert, key, ca, err := loadSSLCertsFromFile(
			os.Getenv("K8S_API_CERT_PEM_STRING"),
			os.Getenv("K8S_API_KEY_PEM_STRING"),
			os.Getenv("K8S_API_CA_PEM_STRING"),
		)
		if err != nil {
			logger.Fatal("Can't load Kuberentes SSL cert files!", err)
		}
		api.BrokerConfig.K8sClusterCredential.AdminCert = cert
		api.BrokerConfig.K8sClusterCredential.AdminKey = key
		api.BrokerConfig.K8sClusterCredential.CaCert = ca
	}
}

//TODO we should rerfactor K8sClusterCredential to accept files instead strings
func loadSSLCertsFromFile(certPemFile, keyPemFile, caPemFile string) (string, string, string, error) {
	certPemByte, err := ioutil.ReadFile(certPemFile)
	if err != nil {
		return "", "", "", err
	}

	keyPemByte, err := ioutil.ReadFile(keyPemFile)
	if err != nil {
		return "", "", "", err
	}

	caPemByte, err := ioutil.ReadFile(caPemFile)
	if err != nil {
		return "", "", "", err
	}
	return string(certPemByte), string(keyPemByte), string(caPemByte), nil
}
