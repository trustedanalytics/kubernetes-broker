// +build local

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
	"os"
)

func (k *K8sCreatorConnector) GetDefaultCluster() K8sClusterCredentials {
	k8sCreatorPostClusterResponse := K8sClusterCredentials{}
	k8sCreatorPostClusterResponse.Server = k.GetLocalAddress()
	k8sCreatorPostClusterResponse.CLusterName = "test-doc"
	k8sCreatorPostClusterResponse.Username = ""
	k8sCreatorPostClusterResponse.AdminCert = os.Getenv("KUBERNETES_CERT_PEM_STRING")
	k8sCreatorPostClusterResponse.AdminKey = os.Getenv("KUBERNETES_KEY_PEM_STRING")
	k8sCreatorPostClusterResponse.CaCert = os.Getenv("KUBERNETES_CA_PEM_STRING")
	return k8sCreatorPostClusterResponse
}
func (k *K8sCreatorConnector) DeleteCluster(org string) error {
	return nil
}
func (k *K8sCreatorConnector) GetCluster(org string) (int, K8sClusterCredentials, error) {
	logger.Info("Local version OK")
	return 200, k.GetDefaultCluster(), nil
}
func (k *K8sCreatorConnector) PostCluster(org string) (int, error) {
	return 200, nil
}
func (k *K8sCreatorConnector) GetClusters() ([]K8sClusterCredentials, error) {
	k8sCreatorGetClustersResponse := []K8sClusterCredentials{}
	return k8sCreatorGetClustersResponse, nil
}
func (k *K8sCreatorConnector) GetOrCreateCluster(org string) (K8sClusterCredentials, error) {
	status, kresp, err := k.GetCluster(org)
	if status == 200 && err != nil {
		return kresp, nil
	} else {
		return kresp, err
	}
}

func (k *K8sCreatorConnector) GetLocalAddress() string {
	address := os.Getenv("K8S_API_ADDRESS")
	if address == "" {
		address = "http://localhost:8080"
	}
	return address
}
