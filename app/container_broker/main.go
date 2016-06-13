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
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocraft/web"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/trustedanalytics/kubernetes-broker/app/container_broker/api"
	templateRepositoryApi "github.com/trustedanalytics/kubernetes-broker/app/template_repository/api"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/logger"
	"github.com/trustedanalytics/kubernetes-broker/state"
)

type appHandler func(web.ResponseWriter, *web.Request) error

var logger = logger_wrapper.InitLogger("main")
var working = true
var waitGroup = &sync.WaitGroup{}

func main() {
	signalChan := make(chan os.Signal, 1)

	initServices()
	go runJobsProcessor()
	go terminationObserver(signalChan)

	r := web.New(api.Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*api.Context).CheckBrokerConfig)
	r.Middleware((*api.Context).BasicAuthorizeMiddleware)

	r.Put("/service", (*api.Context).CreateServiceInstance)
	r.Delete("/service", (*api.Context).DeleteServiceInstance)
	r.Post("/bind", (*api.Context).Bind)
	r.Post("/unbind", (*api.Context).Unbind)

	port := os.Getenv("CONTAINER_BROKER_PORT")
	logger.Info("Will listen on:", port)

	isSSLEnabled, err := strconv.ParseBool(os.Getenv("CONTAINER_BROKER_SSL_ACTIVE"))
	if err != nil {
		logger.Critical("Couldn't read env CONTAINER_BROKER_SSL_ACTIVE!", err)
	}

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

func initServices() {
	templateRepositoryConnector, err := getTemplateRepositoryConnector()
	if err != nil {
		logger.Fatal("Can't connect with TAP-NG template provider!", err)
	}

	api.BrokerConfig = &api.Config{}
	api.BrokerConfig.StateService = &state.StateMemoryService{}
	api.BrokerConfig.KubernetesApi = k8s.NewK8Fabricator()
	api.BrokerConfig.TemplateRepository = templateRepositoryConnector
	api.BrokerConfig.K8sClusterCredentials = k8s.K8sClusterCredentials{
		Server:   os.Getenv("K8S_API_ADDRESS"),
		Username: os.Getenv("K8S_API_USERNAME"),
		Password: os.Getenv("K8S_API_PASSWORD"),
	}
}

func getTemplateRepositoryConnector() (*templateRepositoryApi.TemplateRepositoryConnector, error) {
	isSSLEnabled, err := strconv.ParseBool(os.Getenv("TEMPLATE_REPOSITORY_SSL_ACTIVE"))
	if err != nil {
		logger.Critical("Couldn't read env TEMPLATE_REPOSITORY_SSL_ACTIVE!", err)
	}
	address := getTemplateRepostoryAddress()
	if isSSLEnabled {
		return templateRepositoryApi.NewTemplateRepositoryCa(
			"https://"+address,
			os.Getenv("TEMPLATE_REPOSITORY_USER"),
			os.Getenv("TEMPLATE_REPOSITORY_PASS"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_CERT_FILE_LOCATION"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_KEY_FILE_LOCATION"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_CA_FILE_LOCATION"),
		)
	} else {
		return templateRepositoryApi.NewTemplateRepositoryBasicAuth(
			"http://"+address,
			os.Getenv("TEMPLATE_REPOSITORY_USER"),
			os.Getenv("TEMPLATE_REPOSITORY_PASS"),
		)
	}
}

func getTemplateRepostoryAddress() string {
	templateRepositoryServiceName := os.Getenv("TEMPLATE_REPOSITORY_KUBERNETES_SERVICE_NAME")
	if templateRepositoryServiceName != "" {
		templateRepositoryHostName := os.Getenv(strings.ToUpper(templateRepositoryServiceName) + "_SERVICE_HOST")
		if templateRepositoryHostName != "" {
			templateRepositoryPort := os.Getenv(strings.ToUpper(templateRepositoryServiceName) + "_SERVICE_PORT")
			return templateRepositoryHostName + ":" + templateRepositoryPort
		}
	}
	return "localhost" + ":" + os.Getenv("TEMPLATE_REPOSITORY_PORT")
}

func runJobsProcessor() {
	maxOrgsNo, err := strconv.Atoi(cfenv.CurrentEnv()["CHECK_JOB_INTERVAL_SEC"])
	if err != nil {
		logger.Fatal("CHECK_JOB_INTERVAL_SEC env not set or incorrect: " + err.Error())
	}

	waitGroup.Add(1)
	for working {
		time.Sleep(time.Duration(maxOrgsNo) * time.Second)
		api.BrokerConfig.KubernetesApi.ProcessJobsResult(api.BrokerConfig.K8sClusterCredentials, api.BrokerConfig.StateService)
	}
	waitGroup.Done()
}

func terminationObserver(signalChan chan os.Signal) {
	signal.Notify(signalChan, os.Interrupt)
	for _ = range signalChan {
		logger.Info("Container-broker is going to be stopped now...")
		working = false
		waitGroup.Wait()
		logger.Info("Container-broker stopped!")
		os.Exit(1)
	}
}
