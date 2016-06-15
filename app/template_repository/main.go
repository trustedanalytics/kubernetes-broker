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
	"strconv"
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

	catalog.LoadAvailableTemplates()
	initServices()

	r := web.New(api.Context{})
	r.Middleware(web.LoggerMiddleware)
	r.Middleware((*api.Context).CheckBrokerConfig)

	basicAuthRouter := r.Subrouter(api.Context{}, "/api/v1")
	basicAuthRouter.Middleware((*api.Context).BasicAuthorizeMiddleware)

	jwtRouter := r.Subrouter(api.Context{}, "/api/v1")
	jwtRouter.Middleware((*api.Context).JWTAuthorizeMiddleware)

	basicAuthRouter.Get("/templates", (*api.Context).Templates)
	basicAuthRouter.Post("/templates", (*api.Context).CreateCustomTemplate)

	basicAuthRouter.Post("/parsed_template/:templateId/", (*api.Context).GenerateParsedTemplate)

	jwtRouter.Get("/template/:templateId", (*api.Context).GetCustomTemplate)
	jwtRouter.Delete("/template/:templateId", (*api.Context).DeleteCustomTemplate)

	port := os.Getenv("TEMPLATE_REPOSITORY_PORT")
	logger.Info("Will listen on:", port)

	isSSLEnabled, err := strconv.ParseBool(os.Getenv("TEMPLATE_REPOSITORY_SSL_ACTIVE"))
	if err != nil {
		logger.Critical("Couldn't read env TEMPLATE_REPOSITORY_SSL_ACTIVE!", err)
	}

	if isSSLEnabled {
		err = http.ListenAndServeTLS(":"+port, os.Getenv("TEMPLATE_REPOSITORY_SSL_CERT_FILE_LOCATION"),
			os.Getenv("TEMPLATE_REPOSITORY_SSL_KEY_FILE_LOCATION"), r)
	} else {
		err = http.ListenAndServe(":"+port, r)
	}

	if err != nil {
		logger.Critical("Couldn't serve app on port:", port, " Error:", err)
	}
}

func initServices() {
	api.BrokerConfig = &api.Config{}
	api.BrokerConfig.StateService = &state.StateMemoryService{}
}
