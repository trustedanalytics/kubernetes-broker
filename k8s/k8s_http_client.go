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
	"strconv"

	"github.com/cloudfoundry-community/go-cfenv"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	"k8s.io/kubernetes/pkg/client/restclient"
	k8sClient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"

	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
	"github.com/trustedanalytics/kubernetes-broker/logger"
)

// we need this redundant interface to be able to inject TestClient in Test class
type KubernetesClient interface {
	ReplicationControllers(namespace string) k8sClient.ReplicationControllerInterface
	Nodes() k8sClient.NodeInterface
	Events(namespace string) k8sClient.EventInterface
	Endpoints(namespace string) k8sClient.EndpointsInterface
	Pods(namespace string) k8sClient.PodInterface
	PodTemplates(namespace string) k8sClient.PodTemplateInterface
	Services(namespace string) k8sClient.ServiceInterface
	LimitRanges(namespace string) k8sClient.LimitRangeInterface
	ResourceQuotas(namespace string) k8sClient.ResourceQuotaInterface
	ServiceAccounts(namespace string) k8sClient.ServiceAccountsInterface
	Secrets(namespace string) k8sClient.SecretsInterface
	Namespaces() k8sClient.NamespaceInterface
	PersistentVolumes() k8sClient.PersistentVolumeInterface
	PersistentVolumeClaims(namespace string) k8sClient.PersistentVolumeClaimInterface
	ComponentStatuses() k8sClient.ComponentStatusInterface
	ConfigMaps(namespace string) k8sClient.ConfigMapsInterface
}

type KubernetesClientCreator interface {
	GetNewClient(creds K8sClusterCredentials) (KubernetesClient, error)
}

type KubernetesRestCreator struct {
}

type KubernetesTestCreator struct {
	testClient *testclient.Fake
}

var logger = logger_wrapper.InitLogger("k8s")

func (k *KubernetesRestCreator) GetNewClient(creds K8sClusterCredentials) (KubernetesClient, error) {
	return getKubernetesClient(creds)
}

func getKubernetesClient(creds K8sClusterCredentials) (KubernetesClient, error) {
	sslActive, parseError := strconv.ParseBool(cfenv.CurrentEnv()["KUBE_SSL_ACTIVE"])
	if parseError != nil {
		logger.Error("KUBE_SSL_ACTIVE env probably not set!")
		return nil, parseError
	}

	var transport *http.Transport
	var err error

	if sslActive {
		_, transport, err = brokerHttp.GetHttpClientWithCertAndCa(creds.AdminCert, creds.AdminKey, creds.CaCert)
	} else {
		_, transport, err = brokerHttp.GetHttpClientWithBasicAuth()
	}

	if err != nil {
		return nil, err
	}

	config := &restclient.Config{
		Host:      creds.Server,
		Username:  creds.Username,
		Password:  creds.Password,
		Transport: transport,
	}
	return k8sClient.New(config)
}

func (k *KubernetesTestCreator) GetNewClient(creds K8sClusterCredentials) (KubernetesClient, error) {
	return k.testClient, nil
}

/*
	Objects will be returned in provided order
	All objects should do same action e.g. list/update/create
*/
func (k *KubernetesTestCreator) LoadSimpleResponsesWithSameAction(responeObjects ...runtime.Object) {
	k.testClient = testclient.NewSimpleFake(responeObjects...)
}

type KubernetesTestAdvancedParams struct {
	Verb            string
	Resource        string
	ResponceObjects []runtime.Object
}

/*
	This method allow to inject response object dependly of their action
*/
func (k *KubernetesTestCreator) LoadAdvancedResponses(params []KubernetesTestAdvancedParams) {
	fakeClient := &testclient.Fake{}

	for _, param := range params {
		o := testclient.NewObjects(api.Scheme, api.Codecs.UniversalDecoder())
		for _, obj := range param.ResponceObjects {
			if err := o.Add(obj); err != nil {
				panic(err)
			}
		}
		verb := param.Verb
		if param.Verb == "" {
			verb = "*"
		}

		resource := param.Resource
		if param.Resource == "" {
			resource = "*"
		}
		fakeClient.AddReactor(verb, resource, testclient.ObjectReaction(o, registered.RESTMapper()))
	}

	fakeClient.AddWatchReactor("*", testclient.DefaultWatchReactor(watch.NewFake(), nil))
	k.testClient = fakeClient
}
