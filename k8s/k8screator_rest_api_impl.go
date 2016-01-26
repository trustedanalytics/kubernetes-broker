// +build !local

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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
)

func (k *K8sCreatorConnector) DeleteCluster(org string) error {
	k8sCreatorPostClusterResponse := K8sClusterCredential{}
	_, resp, err := brokerHttp.RestDELETE(k.Server+"/clusters/"+org, &brokerHttp.BasicAuth{k.Username, k.Password}, k.Client)
	err = json.Unmarshal(resp, &k8sCreatorPostClusterResponse)
	if err != nil {
		logger.Error("[DeleteCluster] Error: ", err)
		return err
	}
	return nil
}
func (k *K8sCreatorConnector) GetCluster(org string) (int, K8sClusterCredential, error) {
	url := k.Server + "/clusters/" + org
	k8sCreatorPostClusterResponse := K8sClusterCredential{}

	logger.Info("[GetCluster] GetCluster on url: ", url)
	status, resp, err := brokerHttp.RestGET(url, &brokerHttp.BasicAuth{k.Username, k.Password}, k.Client)

	if status == 404 || status == 204 {
		return status, K8sClusterCredential{}, nil
	}

	err = json.Unmarshal(resp, &k8sCreatorPostClusterResponse)
	if err != nil {
		return status, K8sClusterCredential{}, err
	}
	return status, k8sCreatorPostClusterResponse, nil
}
func (k *K8sCreatorConnector) PostCluster(org string) (int, error) {
	err := k.checkIfClustersQuotaNotExeeded()
	if err != nil {
		return -1, err
	}

	url := k.Server + "/clusters/" + org
	logger.Info("[PostCluster] PostCluster on url: ", url)
	status, _, err := brokerHttp.RestPUT(url, "", &brokerHttp.BasicAuth{k.Username, k.Password}, k.Client)

	if err != nil {
		return -1, err
	}
	return status, nil
}

func (k *K8sCreatorConnector) checkIfClustersQuotaNotExeeded() error {
	clusters, err := k.GetClusters()
	if err != nil {
		return err
	}

	if len(clusters) > k.OrgQuota {
		return errors.New(fmt.Sprintf("Clusters quota exceeded! Max allowed level is: %d", k.OrgQuota))
	} else {
		return nil
	}
}

func (k *K8sCreatorConnector) GetClusters() ([]K8sClusterCredential, error) {
	k8sCreatorGetClustersResponse := []K8sClusterCredential{}

	_, resp, err := brokerHttp.RestGET(k.Server+"/clusters", &brokerHttp.BasicAuth{k.Username, k.Password}, k.Client)
	logger.Debug("RESP: ", string(resp))
	err = json.Unmarshal(resp, &k8sCreatorGetClustersResponse)
	if err != nil {
		logger.Error("[GetClusters] Error: ", err)
		return []K8sClusterCredential{}, err
	}
	return k8sCreatorGetClustersResponse, nil
}

func (k *K8sCreatorConnector) GetOrCreateCluster(org string) (K8sClusterCredential, error) {
	wasCreated := false
	for {
		status, kresp, err := k.GetCluster(org)

		if err != nil {
			logger.Error("[GetOrCreateCluster] ERROR: GetCluster! We will not fetch/create requested cluster!", err)
			return K8sClusterCredential{}, err
		}

		if status == 200 && k.IsApiWorking(kresp) {
			logger.Warning("[GetOrCreateCluster] Cluster already created for org:", org)
			return kresp, nil
		} else if status == 404 {
			if !wasCreated {
				logger.Info("[GetOrCreateCluster] Creating cluster for org:", org)
				status, err = k.PostCluster(org)
				if err != nil {
					logger.Error("[GetOrCreateCluster] ERROR: PostCluster", err)
					return K8sClusterCredential{}, err
				} else if status == 409 {
					return K8sClusterCredential{}, errors.New("UnExpected Cluster conflict")
				}
				wasCreated = true
			} else {
				return K8sClusterCredential{}, errors.New("After creating CLuster bad response received")
			}
		} else if status == 204 {
			logger.Info("[GetOrCreateCluster] Waiting for cluster to finish creating for org:", org)
		}
		time.Sleep(30 * time.Second)
	}
}
