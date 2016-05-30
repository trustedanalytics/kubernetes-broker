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
	"os"
	"time"

	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/app/template_repository/api"
	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/logger"
	"github.com/trustedanalytics/kubernetes-broker/state"
)

type appHandler func(web.ResponseWriter, *web.Request) error

var logger = logger_wrapper.InitLogger("main")

func main() {
	rand.Seed(time.Now().UnixNano())

	catalog.GetAvailableServicesMetadata()
	initServices()

	r := web.New(api.Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*api.Context).CheckBrokerConfig)
	r.Middleware((*api.Context).BasicAuthorizeMiddleware)

	r.Get("/catalog", (*api.Context).Catalog)
	r.Post("/catalog/parsed", (*api.Context).GenerateParsedTemplate)
	r.Put("/dynamicservice", (*api.Context).CreateAndRegisterDynamicService)
	r.Delete("/dynamicservice", (*api.Context).DeleteAndUnregisterDynamicService)

	address := os.Getenv("TEMPLATE_REPOSITORY_ADDRESS")
	logger.Info("Will listen on:", address)
	err := http.ListenAndServe(address, r)
	if err != nil {
		logger.Critical("Couldn't serve app on address: ", address, " Application will be closed now.")
	}
}

func initServices() {
	api.BrokerConfig = &api.Config{}
	api.BrokerConfig.StateService = &state.StateMemoryService{}
}
