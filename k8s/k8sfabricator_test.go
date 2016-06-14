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
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/kubernetes/pkg/api"
	k8sErrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/extensions"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/state"
	tst "github.com/trustedanalytics/kubernetes-broker/test"
)

func prepareMocksAndRouter(t *testing.T) (fabricator *K8Fabricator, mockStateService *state.MockStateService,
	mockKubernetesRest *KubernetesTestCreator) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStateService = state.NewMockStateService(mockCtrl)

	mockKubernetesRest = &KubernetesTestCreator{}
	fabricator = &K8Fabricator{mockKubernetesRest}
	return
}

const serviceId = "mockKubernetesRest"
const orgHost = "orgHost"
const space = "spaceTest"

var testCreds K8sClusterCredentials = K8sClusterCredentials{Server: orgHost}

func TestFabricateService(t *testing.T) {
	fabricator, mockStateService, mockKubernetesRest := prepareMocksAndRouter(t)

	blueprint := &catalog.KubernetesComponent{
		Deployments: []*extensions.Deployment{&extensions.Deployment{Spec: extensions.DeploymentSpec{
			Template: api.PodTemplateSpec{Spec: api.PodSpec{
				Containers: []api.Container{{}},
			}}}},
		},
		Services:               []*api.Service{&api.Service{}},
		ServiceAccounts:        []*api.ServiceAccount{&api.ServiceAccount{}},
		Secrets:                []*api.Secret{&api.Secret{}},
		PersistentVolumeClaims: []*api.PersistentVolumeClaim{&api.PersistentVolumeClaim{}},
	}

	secretResponse := &api.SecretList{
		Items: []api.Secret{{}},
	}
	pvmResponse := &api.PersistentVolumeClaimList{
		Items: []api.PersistentVolumeClaim{{}},
	}
	deploymentResponse := &extensions.DeploymentList{
		Items: []extensions.Deployment{{}},
	}
	serviceResponse := &api.ServiceList{
		Items: []api.Service{{}},
	}
	serviceAccountResponse := &api.ServiceAccountList{
		Items: []api.ServiceAccount{{}},
	}
	restErrorResponse := getErrorResponseForSpecificResource("*")

	Convey("Test FabricateService", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(secretResponse, pvmResponse, serviceResponse, serviceAccountResponse)
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(deploymentResponse)
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRETS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRET0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIMS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIM0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_DEPLOYMENTS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_DEPLOYMENT0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SVCS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SVC0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_ACCS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_ACC0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_FAB_OK", nil),
			)
			result, err := fabricator.FabricateService(testCreds, space, serviceId, `{"name": "param"}`, mockStateService, blueprint)

			So(err, ShouldBeNil)
			So(result.Url, ShouldEqual, "")
		})

		Convey("Should returns error on Create Secret fail ", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(restErrorResponse)

			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRETS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRET0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "FAILED", gomock.Any()),
			)
			_, err := fabricator.FabricateService(testCreds, space, serviceId, "", mockStateService, blueprint)

			So(err, ShouldNotBeNil)
		})

		Convey("Should returns error on Create Deployments fail ", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(secretResponse, pvmResponse)
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(restErrorResponse)
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRETS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRET0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIMS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIM0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_DEPLOYMENTS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "FAILED", gomock.Any()),
			)
			_, err := fabricator.FabricateService(testCreds, space, serviceId, "", mockStateService, blueprint)

			So(err, ShouldNotBeNil)
		})

		Convey("Should returns error on Create Service fail ", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(secretResponse, pvmResponse, restErrorResponse)
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(deploymentResponse)
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRETS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRET0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIMS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIM0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_DEPLOYMENTS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SVCS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "FAILED", gomock.Any()),
			)
			_, err := fabricator.FabricateService(testCreds, space, serviceId, "", mockStateService, blueprint)

			So(err, ShouldNotBeNil)
		})

		Convey("Should returns error on Create AccountService fail ", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(secretResponse, pvmResponse, serviceResponse, restErrorResponse)
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(deploymentResponse)
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRETS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SECRET0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIMS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_PERSIST_VOL_CLAIM0", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_DEPLOYMENTS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_SVCS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "IN_PROGRESS_CREATING_ACCS", nil),
				mockStateService.EXPECT().ReportProgress(serviceId, gomock.Any(), nil),
				mockStateService.EXPECT().ReportProgress(serviceId, "FAILED", gomock.Any()),
			)
			_, err := fabricator.FabricateService(testCreds, space, serviceId, "", mockStateService, blueprint)

			So(err, ShouldNotBeNil)
		})
		Convey("Should returns error when extra paramaters are wrong", func() {
			_, err := fabricator.FabricateService(testCreds, space, serviceId, `BAD_PARAMETER`, mockStateService, blueprint)

			So(err, ShouldNotBeNil)
		})

	})
}

func TestCheckKubernetesServiceHealthByServiceInstanceId(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	Convey("Test CheckKubernetesServiceHealthByServiceInstanceId", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction()

			response, err := fabricator.CheckKubernetesServiceHealthByServiceInstanceId(testCreds, space, serviceId)
			So(err, ShouldBeNil)
			So(response, ShouldBeTrue)
		})

		//todo this test not works because of the bug in Kubernetes test API - NPE when try to return error from PodList
		/*Convey("Should returns error on Get pods fail", func() {
			mockKubernetesRest.Init(getErrorResponseForSpecificResource("PodList"))

			response, err := fabricator.CheckKubernetesServiceHealthByServiceInstanceId(testCreds, space, serviceId)

			So(err, ShouldNotBeNil)
			So(response, ShouldBeFalse)
		})*/
	})
}

func TestDeleteAllByServiceId(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	Convey("Test DeleteAllByServiceId", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction()
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient()

			err := fabricator.DeleteAllByServiceId(testCreds, serviceId)
			So(err, ShouldBeNil)
		})

		Convey("Should returns error on List ServiceAccounts fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("ServiceAccountList"))

			err := fabricator.DeleteAllByServiceId(testCreds, serviceId)
			So(err, ShouldNotBeNil)
		})

		Convey("Should returns error on List Services fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("ServiceList"))

			err := fabricator.DeleteAllByServiceId(testCreds, serviceId)
			So(err, ShouldNotBeNil)
		})

		/*Convey("Should returns error on List Deployments fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction()
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(getErrorResponseForSpecificExtensionsResource("DeploymentList"))

			err := fabricator.DeleteAllByServiceId(testCreds, serviceId)
			So(err, ShouldNotBeNil)
		})*/

		Convey("Should returns error on List Secret fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("SecretList"))

			err := fabricator.DeleteAllByServiceId(testCreds, serviceId)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetAllPodsEnvsByServiceId(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	Convey("Test GetAllPodsEnvsByServiceId", t, func() {
		/*
		Convey("Should returns proper response", func() {
			env_name := "name"
			env_val := "val"

			deploymentResponse := &extensions.DeploymentList{
				Items: []extensions.Deployment{
					{Spec: extensions.DeploymentSpec{
						Selector: &unversioned.LabelSelector{MatchLabels: map[string]string{serviceIdLabel: serviceId, managedByLabel: "TAP"}},
						Template: api.PodTemplateSpec{
							Spec: api.PodSpec{
								Containers: []api.Container{
									{Env: []api.EnvVar{{Name: env_name, Value: env_val}}},
								},
							}}}},
				},
			}
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient(deploymentResponse)

			result, err := fabricator.GetAllPodsEnvsByServiceId(testCreds, space, serviceId)
			So(err, ShouldBeNil)
			So(result, ShouldNotBeEmpty)
			So(result, ShouldHaveLength, 1)
			So(result[0].Containers[0].Envs[env_name], ShouldEqual, env_val)
		})

		Convey("Should returns error on List ReplicationControllers fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("ReplicationControllerList"))

			_, err := fabricator.GetAllPodsEnvsByServiceId(testCreds, space, serviceId)
			So(err, ShouldNotBeNil)
		})*/

		Convey("Should returns error when no items in respone", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction()
			mockKubernetesRest.LoadSimpleResponsesWithSameActionForExtensionsClient()

			_, err := fabricator.GetAllPodsEnvsByServiceId(testCreds, space, serviceId)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "No deployments associated with the service: "+serviceId)
		})
	})
}

func TestGetSecret(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	secret := tst.GetTestSecret()

	Convey("Test GetSecret", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(&secret)
			result, err := fabricator.GetSecret(testCreds, tst.TestSecretName)

			So(err, ShouldBeNil)
			So(result.Name, ShouldEqual, tst.TestSecretName)
			So(result.Data[tst.TestSecretName], ShouldResemble, tst.GetTestSecretData())
		})

		Convey("Should returns error on SecretsGet fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("Secret"))
			_, err := fabricator.GetSecret(testCreds, tst.TestSecretName)

			So(err, ShouldNotBeNil)
		})
	})
}

func TestCreateSecret(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	secret := tst.GetTestSecret()

	Convey("Test CreateSecret", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(&secret)
			err := fabricator.CreateSecret(testCreds, secret)

			So(err, ShouldBeNil)
		})

		Convey("Should returns error on SecretsCreate fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("Secret"))
			err := fabricator.CreateSecret(testCreds, secret)

			So(err, ShouldNotBeNil)
		})
	})
}

func TestUpdateSecret(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	secret := tst.GetTestSecret()

	Convey("Test UpdateSecret", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(&secret)
			err := fabricator.UpdateSecret(testCreds, secret)

			So(err, ShouldBeNil)
		})

		Convey("Should returns error on SecretsGet fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("Secret"))
			err := fabricator.UpdateSecret(testCreds, secret)

			So(err, ShouldNotBeNil)
		})
	})
}

func TestDeleteSecret(t *testing.T) {
	fabricator, _, mockKubernetesRest := prepareMocksAndRouter(t)

	secret := tst.GetTestSecret()

	Convey("Test DeleteSecret", t, func() {
		Convey("Should returns proper response", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(&secret)
			err := fabricator.DeleteSecret(testCreds, tst.TestSecretName)

			So(err, ShouldBeNil)
		})

		Convey("Should returns error on SecretsGet fail", func() {
			mockKubernetesRest.LoadSimpleResponsesWithSameAction(getErrorResponseForSpecificResource("Secret"))
			err := fabricator.DeleteSecret(testCreds, tst.TestSecretName)

			So(err, ShouldNotBeNil)
		})
	})
}

func getErrorResponseForSpecificResource(resourceName string) runtime.Object {
	return &api.List{
		Items: []runtime.Object{
			&(k8sErrors.NewForbidden(api.Resource(resourceName), "", nil).(*k8sErrors.StatusError).ErrStatus),
		},
	}
}

func getErrorResponseForSpecificExtensionsResource(resourceName string) runtime.Object {
	return &api.List{
		Items: []runtime.Object{
			&(k8sErrors.NewForbidden(extensions.Resource(resourceName), "", nil).(*k8sErrors.StatusError).ErrStatus),
		},
	}
}
