// +build local

/**
 * Copyright (c) 2015 Intel Corporation
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

func (k *K8sCreatorConnector) GetDefaultCluster() K8sClusterCredential {
	k8sCreatorPostClusterResponse := K8sClusterCredential{}
	k8sCreatorPostClusterResponse.Server = "http://localhost:" + k.GetLocalPort()
	k8sCreatorPostClusterResponse.CLusterName = "test-doc"
	k8sCreatorPostClusterResponse.Username = ""
	return k8sCreatorPostClusterResponse
}
func (k *K8sCreatorConnector) DeleteCluster(org string) error {
	return nil
}
func (k *K8sCreatorConnector) GetCluster(org string) (int, K8sClusterCredential, error) {
	logger.Info("Local version OK")
	return 200, k.GetDefaultCluster(), nil
}
func (k *K8sCreatorConnector) PostCluster(org string) (int, error) {
	return 200, nil
}
func (k *K8sCreatorConnector) GetClusters() ([]K8sClusterCredential, error) {
	k8sCreatorGetClustersResponse := []K8sClusterCredential{}
	return k8sCreatorGetClustersResponse, nil
}
func (k *K8sCreatorConnector) GetOrCreateCluster(org string) (K8sClusterCredential, error) {
	status, kresp, err := k.GetCluster(org)
	if status == 200 && err != nil {
		return kresp, nil
	} else {
		return kresp, err
	}
}

func (k *K8sCreatorConnector) GetLocalPort() string {
	port := os.Getenv("K8S_API_PORT")
	if port == "" {
		port = "8080"
	}
	return port
}
