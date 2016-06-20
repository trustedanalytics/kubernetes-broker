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
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/consul"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/logger"
	"github.com/trustedanalytics/kubernetes-broker/state"
)

type Context struct{}

type appHandler func(web.ResponseWriter, *web.Request) error

var logger = logger_wrapper.InitLogger("main")

func main() {
	catalog.GetAvailableServicesMetadata()

	rand.Seed(time.Now().UnixNano())

	cfApp, err := cfenv.Current()
	if err != nil {
		logger.Fatal("CF Env vars gathering failed. Running locally, probably.\n", err)
	}
	logger.Debug("CF ENV: ", cfApp)
	logger.Info("Starting. Working directory is: ", cfApp.WorkingDir)

	initServices(cfApp)
	removeNotUsedClusters()

	r := web.New(Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*Context).CheckBrokerConfig)
	r.Error((*Context).Error)

	r.Get("/", (*Context).Index)

	jwtRouter := r.Subrouter(Context{}, "/rest")
	jwtRouter.Middleware((*Context).JWTAuthorizeMiddleware)

	basicAuthRouter := r.Subrouter(Context{}, "/v2")
	basicAuthRouter.Middleware((*Context).BasicAuthorizeMiddleware)

	jwtRouter.Get("/kubernetes/:org_id/:space_id/service/:instance_id", (*Context).GetService)
	jwtRouter.Get("/kubernetes/:org_id/:space_id/services", (*Context).GetServices)
	jwtRouter.Post("/kubernetes/service/visibility", (*Context).SetServiceVisibility)

	jwtRouter.Get("/kubernetes/:org_id/secret/:key", (*Context).GetSecret)
	jwtRouter.Post("/kubernetes/:org_id/secret/:key", (*Context).CreateSecret)
	jwtRouter.Delete("/kubernetes/:org_id/secret/:key", (*Context).DeleteSecret)
	jwtRouter.Put("/kubernetes/:org_id/secret/:key", (*Context).UpdateSecret)
	jwtRouter.Get("/kubernetes/catalog/:service_id", (*Context).GetServiceDetails)

	basicAuthRouter.Get("/catalog", (*Context).Catalog)
	basicAuthRouter.Put("/service_instances/:instance_id", (*Context).ServiceInstancesPut)
	basicAuthRouter.Get("/service_instances/:instance_id/last_operation", (*Context).ServiceInstancesGetLastOperation)
	basicAuthRouter.Delete("/service_instances/:instance_id", (*Context).ServiceInstancesDelete)
	basicAuthRouter.Put("/service_instances/:instance_id/service_bindings/:binding_id", (*Context).ServiceBindingsPut)
	basicAuthRouter.Delete("/service_instances/:instance_id/service_bindings/:binding_id", (*Context).ServiceBindingsDelete)

	basicAuthRouter.Put("/dynamicservice", (*Context).CreateAndRegisterDynamicService)
	basicAuthRouter.Delete("/dynamicservice", (*Context).DeleteAndUnRegisterDynamicService)
	basicAuthRouter.Get("/:org_id/service/:instance_id/status", (*Context).CheckPodsStatusForService)
	basicAuthRouter.Get("/:org_id/services/status", (*Context).CheckPodsStatusForAllServicesInOrg)

	logger.Info("Will listen on:", cfApp.Host, cfApp.Port)
	err = http.ListenAndServe(cfApp.Host+":"+strconv.Itoa(cfApp.Port), r)
	if err != nil {
		logger.Critical("Couldn't serve app on port ", cfApp.Port, " Application will be closed now.")
	}
}

func initServices(cfApp *cfenv.App) {
	brokerConfig = &BrokerConfig{}

	serviceDomain := "DOMAIN_NOT_SET"
	if len(cfApp.ApplicationURIs) > 0 {
		serviceDomain = cfApp.ApplicationURIs[0]
		serviceDomain = strings.TrimPrefix(serviceDomain, "kubernetes-broker.")
	}
	brokerConfig.Domain = serviceDomain

	sso, err := cfApp.Services.WithName("sso")
	if err != nil {
		logger.Fatal("SSO service can't be found!", err)
	}
	brokerConfig.CloudProvider = NewCFApiClient(
		sso.Credentials["clientId"].(string),
		sso.Credentials["clientSecret"].(string),
		sso.Credentials["tokenUri"].(string),
		sso.Credentials["apiEndpoint"].(string),
	)
	TokenKeyURL = sso.Credentials["tokenKey"].(string)

	kubeCreds, err := cfApp.Services.WithName("kubernetes-creator-credentials")
	if err != nil {
		logger.Fatal("kubernetes-creator-credentials service can't be found!", err)
	}
	maxOrgsNo, err := strconv.Atoi(cfenv.CurrentEnv()["MAX_ORG_QUOTA"])
	if err != nil {
		logger.Fatal("MAX_ORG_QUOTA env not set or incorrect: " + err.Error())
	}
	brokerConfig.CreatorConnector = k8s.NewK8sCreatorConnector(
		kubeCreds.Credentials["url"].(string),
		kubeCreds.Credentials["username"].(string),
		kubeCreds.Credentials["password"].(string),
		maxOrgsNo,
	)

	brokerConfig.StateService = &state.StateMemoryService{}
	brokerConfig.KubernetesApi = k8s.NewK8Fabricator()
	brokerConfig.ConsulApi = &consul.ConsulConnector{}

	waitBeforeNextPVCheckSec, err := strconv.Atoi(cfenv.CurrentEnv()["WAIT_BEFORE_NEXT_PV_CHECK_SEC"])
	if err != nil {
		logger.Fatal("WAIT_BEFORE_NEXT_PV_CHECK_SEC env not set or incorrect: " + err.Error())
	}
	waitBeforeRemoveClusterSec, err := strconv.Atoi(cfenv.CurrentEnv()["WAIT_BEFORE_REMOVE_CLUSTER_SEC"])
	if err != nil {
		logger.Fatal("WAIT_BEFORE_REMOVE_CLUSTER_SEC env not set or incorrect: " + err.Error())
	}
	brokerConfig.CheckPVbeforeRemoveClusterIntervalSec = time.Second * time.Duration(waitBeforeNextPVCheckSec)
	brokerConfig.WaitBeforeRemoveClusterIntervalSec = time.Second * time.Duration(waitBeforeRemoveClusterSec)
}

func removeNotUsedClusters() {
	clusters, err := brokerConfig.CreatorConnector.GetClusters()
	if err != nil {
		logger.Error("[removeNotUsedClusters] GetClusters error:", err)
		return
	}
	for _, cluster := range clusters {
		go removeCluster(cluster, cluster.CLusterName)
	}
}
