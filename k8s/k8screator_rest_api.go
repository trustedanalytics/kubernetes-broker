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
package k8s

import (
	"net/http"

	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
)

type K8sCreatorRest interface {
	DeleteCluster(org string) error
	GetCluster(org string) (int, K8sClusterCredential, error)
	GetOrCreateCluster(org string) (K8sClusterCredential, error)
	PostCluster(org string) (int, error)
	GetClusters() ([]K8sClusterCredential, error)
}

type K8sCreatorConnector struct {
	ApiVersion       string
	Server           string
	Username         string
	Password         string
	Client           *http.Client
	OrgQuota         int
	KubernetesClient KubernetesClientCreator
}

type K8sClusterCredential struct {
	CLusterName    string `json:"cluster_name"`
	Server         string `json:"api_server"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	CaCert         string `json:"ca_cert"`
	AdminKey       string `json:"admin_key"`
	AdminCert      string `json:"admin_cert"`
	ConsulEndpoint string `json:"consul_http_api"`
}

func NewK8sCreatorConnector(server, user, pass string, maxOrgQuota int) *K8sCreatorConnector {
	clientCreator, _, err := brokerHttp.GetHttpClientWithBasicAuth()
	if err != nil {
		logger.Panic("Can't get http client!", err)
	}

	return &K8sCreatorConnector{
		Server:           server,
		Username:         user,
		Password:         pass,
		Client:           clientCreator,
		OrgQuota:         maxOrgQuota,
		KubernetesClient: &KubernetesRestCreator{},
	}
}

func (k *K8sCreatorConnector) IsApiWorking(credential K8sClusterCredential) bool {
	req_url := credential.Server + "/api/v1"
	statusCde, _, err := brokerHttp.RestGET(req_url, &brokerHttp.BasicAuth{credential.Username, credential.Password}, k.Client)

	if err != nil {
		logger.Error("[IsApiWorking] Error: ", err)
		return false
	}
	return statusCde == 200
}
