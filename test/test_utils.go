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
package test

import (
	"os"
	"strings"

	"k8s.io/kubernetes/pkg/api"

	"github.com/trustedanalytics/kubernetes-broker/logger"
)

const TestServiceId string = "testServiceId"
const TestPlanId string = "testPlanId"
const TestOrgGuid string = "testOrgGuid"
const TestOrgHost string = "testOrgHost"
const TestSpaceGuid string = "testSpaceGuid"

const TestSecretName string = "testSecretName"

const TestInternalServiceId = "consul"
const TestInternalPlanId = "simple"

const testSecretName = "testSecretName"

var logger = logger_wrapper.InitLogger("test")

func GetTestCatalogPath(nameToTrim string) string {
	pwd, err := os.Getwd()
	if err != nil {
		logger.Fatal("Unable to get working directory!")
	}
	return "/" + strings.Trim(pwd, nameToTrim) + "/test/catalog/"
}

func GetTestSecretData() []byte {
	return []byte("dGVzdA==")
}

func GetTestSecret() api.Secret {
	secret := api.Secret{}
	secret.Name = testSecretName
	data := make(map[string][]byte)
	data[testSecretName] = GetTestSecretData()
	secret.Data = data
	return secret
}
