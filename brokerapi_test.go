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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gocraft/web"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"k8s.io/kubernetes/pkg/api"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
	"github.com/trustedanalytics/kubernetes-broker/state"
	tst "github.com/trustedanalytics/kubernetes-broker/test"
)

const URLcatalogPath = "/v2/catalog"
const URLrestCatalogPath = "/rest/kubernetes/catalog"
const URLserviceDetailsPath = "/rest/kubernetes/catalog/:service_id"
const URLservicePath = "/rest/kubernetes/:org_id/:space_id/service/:instance_id"
const URLservicesPath = "/rest/kubernetes/:org_id/:space_id/services"
const URLsecretPath = "/rest/kubernetes/:org_id/secret/:key"
const URLquotaPath = "/rest/quota"
const URLserviceInstancePath = "/v2/service_instances/"
const URLserviceInstanceIdPath = "/v2/service_instances/:instance_id"
const URLlastOperationPath = "/v2/service_instances/:instance_id/last_operation"
const URLserviceBindingsPath = "/v2/service_instances/:instance_id/service_bindings/:binding_id"

var testCatalogPath = tst.GetTestCatalogPath()
var testError error = errors.New("New Errror")
var testCreds k8s.K8sClusterCredential = k8s.K8sClusterCredential{tst.TestOrgHost, tst.TestOrgHost, "", "", ""}

func prepareMocksAndRouter(t *testing.T) (r *web.Router, mockCloudAPi *MockCloudApi,
	mockKubernetesApi *k8s.MockKubernetesApi, mockStateService *state.MockStateService, mockCreatorConnector *k8s.MockK8sCreatorRest) {

	// setup Catalog for example test files
	catalog.CatalogPath = testCatalogPath

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockCloudAPi = NewMockCloudApi(mockCtrl)
	mockKubernetesApi = k8s.NewMockKubernetesApi(mockCtrl)
	mockStateService = state.NewMockStateService(mockCtrl)
	mockCreatorConnector = k8s.NewMockK8sCreatorRest(mockCtrl)

	brokerConfig = &BrokerConfig{
		CloudProvider:                         mockCloudAPi,
		KubernetesApi:                         mockKubernetesApi,
		StateService:                          mockStateService,
		CreatorConnector:                      mockCreatorConnector,
		WaitBeforeRemoveClusterIntervalSec:    time.Millisecond,
		CheckPVbeforeRemoveClusterIntervalSec: time.Second,
	}

	r = web.New(Context{})
	return
}

func TestServiceInstancesPut(t *testing.T) {
	request := ServiceInstancesPutRequest{ServiceId: tst.TestServiceId, PlanId: tst.TestPlanId,
		OrganizationGuid: tst.TestOrgGuid, SpaceGuid: tst.TestSpaceGuid}

	instanceId := "4324324324324324324234234"

	r, _, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Put(URLserviceInstanceIdPath, (*Context).ServiceInstancesPut)

	Convey("Test ServiceInstancesPut", t, func() {
		Convey("Should returns proper response", func() {
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_STARTED", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_METADATA_OK", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_IN_BACKGROUND_JOB", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_BLUEPRINT_OK", nil),
				mockCreatorConnector.EXPECT().GetOrCreateCluster(tst.TestOrgGuid).Return(testCreds, nil),
				mockKubernetesApi.EXPECT().FabricateService(testCreds, tst.TestSpaceGuid, instanceId,
					gomock.Any(), mockStateService, gomock.Any()).
					Return(k8s.FabricateResult{}, nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_KUBERNETES_OK", nil),
			)

			rr := sendRequest("PUT", URLserviceInstancePath+instanceId, marshallToJson(t, request), r)
			assertResponse(rr, "", 201)
		})

		Convey("Should returns proper response when async is active", func() {
			// this is because we need to wait for asynchronous call inside ServiceInstancesPut
			var wg sync.WaitGroup

			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_STARTED", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_METADATA_OK", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_IN_BACKGROUND_JOB", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_BLUEPRINT_OK", nil),
				mockCreatorConnector.EXPECT().GetOrCreateCluster(tst.TestOrgGuid).Return(testCreds, nil),
				mockKubernetesApi.EXPECT().FabricateService(testCreds, tst.TestSpaceGuid, instanceId,
					gomock.Any(), mockStateService, gomock.Any()).
					Return(k8s.FabricateResult{}, nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_KUBERNETES_OK", nil).
					Do(func(arg0, arg1, arg2 interface{}) {
						wg.Done()
					}),
			)

			os.Setenv("ACCEPT_INCOMPLETE", "true")
			wg.Add(1)
			rr := sendRequest("PUT", URLserviceInstancePath+instanceId, marshallToJson(t, request), r)
			wg.Wait()
			assertResponse(rr, "", 202)
			os.Unsetenv("ACCEPT_INCOMPLETE")
		})

		Convey("Should returns error when service not exist", func() {
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_STARTED", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any()),
			)

			serviceInstance := ServiceInstancesPutRequest{ServiceId: "FakeServiceId"}
			rr := sendRequest("PUT", URLserviceInstancePath+instanceId, marshallToJson(t, serviceInstance), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error on kubernetes error", func() {
			kubernetesError := errors.New("KUBERNETES ERROR")
			gomock.InOrder(
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_STARTED", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_METADATA_OK", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_IN_BACKGROUND_JOB", nil),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "IN_PROGRESS_BLUEPRINT_OK", nil),
				mockCreatorConnector.EXPECT().GetOrCreateCluster(tst.TestOrgGuid).Return(testCreds, nil),
				mockKubernetesApi.EXPECT().FabricateService(testCreds, tst.TestSpaceGuid, instanceId,
					gomock.Any(), mockStateService, gomock.Any()).
					Return(k8s.FabricateResult{}, kubernetesError),
				mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", kubernetesError),
			)

			rr := sendRequest("PUT", URLserviceInstancePath+instanceId, marshallToJson(t, request), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when incorete request body", func() {
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())

			rr := sendRequest("PUT", URLserviceInstancePath+instanceId, []byte("{WrongJson]"), r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestGetQuota(t *testing.T) {
	r, _, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Get(URLquotaPath, (*Context).GetQuota)

	Convey("Test GetQuota", t, func() {
		//todo fixme
		/*Convey("Should returns proper response", func() {
			req_body := ServiceInstancesPutRequest{OrganizationGuid: tst.TestOrgGuid, SpaceGuid: tst.TestSpaceGuid}

			memory_value := "memory OK"
			quotaList := *api.ResourceQuotaList{Items: []api.ResourceQuota{api.ResourceQuotaSpec{
				Status: k8s.K8sResourceQuotaStatus{Used: k8s.K8sQuotaElements{Memory: memory_value}}}}}

			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetQuota(testCreds, tst.TestSpaceGuid).Return(quotaList, nil)
			rr := sendRequest("GET", URLquotaPath, marshallToJson(t, req_body), r)

			assertResponse(rr, memory_value, 202)
		})*/

		Convey("Should returns error when incorete request body", func() {
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())

			rr := sendRequest("GET", URLquotaPath, []byte("{WrongJson]"), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error on kubernetes error", func() {
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())

			req_body := ServiceInstancesPutRequest{OrganizationGuid: tst.TestOrgGuid, SpaceGuid: tst.TestSpaceGuid}
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetQuota(testCreds, tst.TestSpaceGuid).Return(&api.ResourceQuotaList{}, errors.New("KUBERNETES ERROR"))
			rr := sendRequest("GET", URLquotaPath, marshallToJson(t, req_body), r)

			assertResponse(rr, "", 500)
		})

	})
}

func TestGetCatalog(t *testing.T) {
	r, _, _, _, _ := prepareMocksAndRouter(t)
	r.Get(URLcatalogPath, (*Context).Catalog)

	Convey("Test GetCatalog", t, func() {
		Convey("Should returns proper response", func() {
			rr := sendRequest("GET", URLcatalogPath, nil, r)
			assertResponse(rr, "", 200)
		})
	})
}

func TestGetServiceDetails(t *testing.T) {
	r, _, _, _, _ := prepareMocksAndRouter(t)
	r.Get(URLserviceDetailsPath, (*Context).GetServiceDetails)

	Convey("Test GetServiceDetails", t, func() {
		Convey("Should returns proper response", func() {
			rr := sendRequest("GET", URLrestCatalogPath+"/testServiceId", nil, r)
			assertResponse(rr, "", 200)
		})

		Convey("Should returns 404", func() {
			rr := sendRequest("GET", URLrestCatalogPath+"/non-existentTestServiceId", nil, r)
			assertResponse(rr, "", 404)
		})
	})
}

func TestServiceInstancesGetLastOperation(t *testing.T) {
	testId := "1223"
	requestPath := URLserviceInstancePath + testId + "/last_operation"

	r, mockCloudAPi, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Get(URLlastOperationPath, (*Context).ServiceInstancesGetLastOperation)

	Convey("Test ServiceInstancesGetLastOperation", t, func() {
		Convey("Should returns succeeded response", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockStateService.EXPECT().HasProgressRecords(testId).Return(true),
				mockStateService.EXPECT().ReadProgress(testId).Return(time.Now(), "IN_PROGRESS_KUBERNETES_OK", nil),
				mockKubernetesApi.EXPECT().CheckKubernetesServiceHealthByServiceInstanceId(testCreds, tst.TestSpaceGuid, testId).Return(true, nil),
			)

			rr := sendRequest("GET", requestPath, nil, r)
			response := ServiceInstancesGetLastOperationResponse{}
			err := readJson(rr, &response)

			assertResponse(rr, "", 200)
			So(err, ShouldBeNil)
			So(response.State, ShouldEqual, "succeeded")
		})

		Convey("Should returns failed response", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockStateService.EXPECT().HasProgressRecords(testId).Return(false),
			)

			rr := sendRequest("GET", requestPath, nil, r)
			response := ServiceInstancesGetLastOperationResponse{}
			err := readJson(rr, &response)

			assertResponse(rr, "", 200)
			So(err, ShouldBeNil)
			So(response.State, ShouldEqual, "failed")
		})

		Convey("Should returns catch error response from cloud", func() {
			mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return("", "", errors.New("Error Test"))
			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestServiceInstancesDelete(t *testing.T) {
	testId := "1223"

	r, mockCloudAPi, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Delete(URLserviceInstanceIdPath, (*Context).ServiceInstancesDelete)

	Convey("Test ServiceInstancesDelete", t, func() {
		Convey("Should returns succeeded response", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().DeleteAllByServiceId(testCreds, testId).Return(nil),
				mockKubernetesApi.EXPECT().GetServices(testCreds, tst.TestOrgGuid).Return(nil, nil),
				mockKubernetesApi.EXPECT().ListReplicationControllers(testCreds).Return(&api.ReplicationControllerList{}, nil),
				mockKubernetesApi.EXPECT().DeleteAllPersistentVolumeClaims(testCreds).Return(nil),
				mockKubernetesApi.EXPECT().GetAllPersistentVolumes(testCreds).Return(nil, nil),
				mockCreatorConnector.EXPECT().DeleteCluster(tst.TestOrgGuid).Return(nil),
			)

			rr := sendRequest("DELETE", URLserviceInstancePath+testId, nil, r)
			time.Sleep(time.Second * 3)
			assertResponse(rr, "", 200)
		})

		Convey("Should wait until all PV will be removed and then remove cluster", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().DeleteAllByServiceId(testCreds, testId).Return(nil),
				mockKubernetesApi.EXPECT().GetServices(testCreds, tst.TestOrgGuid).Return(nil, nil),
				mockKubernetesApi.EXPECT().ListReplicationControllers(testCreds).Return(&api.ReplicationControllerList{}, nil),
				mockKubernetesApi.EXPECT().DeleteAllPersistentVolumeClaims(testCreds).Return(nil),
				// return one PV to fore waitingon EBS action
				mockKubernetesApi.EXPECT().GetAllPersistentVolumes(testCreds).Return([]api.PersistentVolume{{}}, nil),

				mockKubernetesApi.EXPECT().GetServices(testCreds, tst.TestOrgGuid).Return(nil, nil),
				mockKubernetesApi.EXPECT().ListReplicationControllers(testCreds).Return(&api.ReplicationControllerList{}, nil),
				mockKubernetesApi.EXPECT().DeleteAllPersistentVolumeClaims(testCreds).Return(nil),
				mockKubernetesApi.EXPECT().GetAllPersistentVolumes(testCreds).Return(nil, nil),
				mockCreatorConnector.EXPECT().DeleteCluster(tst.TestOrgGuid).Return(nil),
			)

			rr := sendRequest("DELETE", URLserviceInstancePath+testId, nil, r)
			time.Sleep(time.Second * 3)
			assertResponse(rr, "", 200)
		})

		Convey("Should break removoving cluster if service occur", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().DeleteAllByServiceId(testCreds, testId).Return(nil),
				mockKubernetesApi.EXPECT().GetServices(testCreds, tst.TestOrgGuid).Return([]api.Service{api.Service{}}, nil),
				mockKubernetesApi.EXPECT().ListReplicationControllers(testCreds).Return(&api.ReplicationControllerList{}, nil),
			)

			rr := sendRequest("DELETE", URLserviceInstancePath+testId, nil, r)
			time.Sleep(time.Second * 3)
			assertResponse(rr, "", 200)
		})

		Convey("Should returns error on kubernetes error", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().DeleteAllByServiceId(testCreds, testId).
					Return(errors.New("KUBERNETES ERROR")),
			)

			rr := sendRequest("DELETE", URLserviceInstancePath+testId, nil, r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error on cloud error", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testId).
					Return("", "", errors.New("CLOUD error")),
			)

			rr := sendRequest("DELETE", URLserviceInstancePath+testId, nil, r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestServiceBindingsPut(t *testing.T) {
	testInstanceId, testBindingId := "instanceId", "bindId"
	requestPath := URLserviceInstancePath + testInstanceId + "/service_bindings/" + testBindingId

	r, mockCloudAPi, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Put(URLserviceBindingsPath, (*Context).ServiceBindingsPut)

	//http://stackoverflow.com/questions/10535743/address-of-a-temporary-in-go
	tmpTestServiceId := tst.TestServiceId
	tmpTestPlanId := tst.TestPlanId

	Convey("Test ServiceBindingsPut", t, func() {
		Convey("Should returns proper response", func() {
			port := 8500

			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testInstanceId).
					Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().GetAllPodsEnvsByServiceId(testCreds, tst.TestSpaceGuid, testInstanceId).
					Return([]k8s.PodEnvs{
						{Containers: []k8s.ContainerSimple{{Envs: map[string]string{"foo": "bar"}}}},
					}, nil),
				mockKubernetesApi.EXPECT().GetServiceCredentials(testCreds, tst.TestSpaceGuid, testInstanceId).
					Return([]k8s.ServiceCredential{}, nil),
			)

			putRequestBody := ServiceBindingsPutRequest{ServiceId: &tmpTestServiceId, PlanId: &tmpTestPlanId}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, fmt.Sprint(port), 201)
		})

		Convey("Should returns error when ServiceId is empty", func() {
			putRequestBody := ServiceBindingsPutRequest{}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when ServiceId is incorrect", func() {
			tmpTestServiceId := "FakeService"

			putRequestBody := ServiceBindingsPutRequest{ServiceId: &tmpTestServiceId, PlanId: &tmpTestPlanId}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when org not exist", func() {
			mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testInstanceId).
				Return("", "", errors.New("No Org"))

			putRequestBody := ServiceBindingsPutRequest{ServiceId: &tmpTestServiceId, PlanId: &tmpTestPlanId}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when env for service not exist", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testInstanceId).
					Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().GetAllPodsEnvsByServiceId(testCreds, tst.TestSpaceGuid, testInstanceId).
					Return([]k8s.PodEnvs{}, errors.New("No env")),
			)

			putRequestBody := ServiceBindingsPutRequest{ServiceId: &tmpTestServiceId, PlanId: &tmpTestPlanId}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when host or port not exist", func() {
			gomock.InOrder(
				mockCloudAPi.EXPECT().GetOrgIdAndSpaceIdFromCfByServiceInstanceId(testInstanceId).
					Return(tst.TestOrgGuid, tst.TestSpaceGuid, nil),
				mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil),
				mockKubernetesApi.EXPECT().GetAllPodsEnvsByServiceId(testCreds, tst.TestSpaceGuid, testInstanceId).
					Return([]k8s.PodEnvs{}, nil),
				mockKubernetesApi.EXPECT().GetServiceCredentials(testCreds, tst.TestSpaceGuid, testInstanceId).
					Return([]k8s.ServiceCredential{}, errors.New("No Port")),
			)

			putRequestBody := ServiceBindingsPutRequest{ServiceId: &tmpTestServiceId, PlanId: &tmpTestPlanId}
			rr := sendRequest("PUT", requestPath, marshallToJson(t, putRequestBody), r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestServiceBindingsDelete(t *testing.T) {
	requestPath := URLserviceInstancePath + "testBinding" + "/service_bindings/" + "testBinding"
	r, _, _, _, _ := prepareMocksAndRouter(t)
	r.Delete(URLserviceBindingsPath, (*Context).ServiceBindingsDelete)

	Convey("Test ServiceBindingsDelete", t, func() {
		Convey("Should returns succeeded response", func() {
			rr := sendRequest("DELETE", requestPath, nil, r)
			assertResponse(rr, "", 410)
		})
	})
}

func TestGetService(t *testing.T) {
	r, _, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Get(URLservicePath, (*Context).GetService)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/" + tst.TestSpaceGuid + "/service/" +
		tst.TestServiceId

	Convey("Test GetService", t, func() {
		Convey("Should returns succeeded response", func() {
			response := []k8s.K8sServiceInfo{{Name: "testName"}}
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetServiceVisibility(testCreds, tst.TestOrgGuid, tst.TestSpaceGuid, tst.TestServiceId).
				Return(response, nil)

			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 202)

			serviceResponse := []k8s.K8sServiceInfo{}
			err := readJson(rr, &serviceResponse)

			So(err, ShouldBeNil)
			So(len(serviceResponse), ShouldEqual, 1)
			So(serviceResponse, ShouldResemble, response)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetServiceVisibility(testCreds, tst.TestOrgGuid,
				tst.TestSpaceGuid, tst.TestServiceId).Return([]k8s.K8sServiceInfo{}, testError)
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())
			rr := sendRequest("GET", requestPath, nil, r)

			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when GetCluster cannot parse unexpected json response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, testError)
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())
			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestGetServices(t *testing.T) {
	r, _, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Get(URLservicesPath, (*Context).GetServices)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/" + tst.TestSpaceGuid + "/services"

	Convey("Test GetServices", t, func() {
		Convey("Should returns succeeded response", func() {
			response := []k8s.K8sServiceInfo{{Name: "testName"}}
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetServicesVisibility(testCreds, tst.TestOrgGuid, tst.TestSpaceGuid).
				Return(response, nil)

			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 202)

			serviceRespone := []k8s.K8sServiceInfo{}
			err := readJson(rr, &serviceRespone)

			So(err, ShouldBeNil)
			So(len(serviceRespone), ShouldEqual, 1)
			So(serviceRespone, ShouldResemble, response)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetServicesVisibility(testCreds, tst.TestOrgGuid, tst.TestSpaceGuid).
				Return([]k8s.K8sServiceInfo{}, testError)
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())
			rr := sendRequest("GET", requestPath, nil, r)

			assertResponse(rr, "", 202)
		})

		Convey("Should returns error when incorete request body", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, testError)
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())

			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 202)
		})
	})
}

func TestSetServiceVisibility(t *testing.T) {
	requestPath := "/v2/services/"

	r, _, mockKubernetesApi, mockStateService, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Post(requestPath, (*Context).SetServiceVisibility)

	request := ServiceInstancesPutRequest{ServiceId: tst.TestServiceId,
		OrganizationGuid: tst.TestOrgGuid, SpaceGuid: tst.TestSpaceGuid, Visibility: true}

	Convey("Test SetServiceVisibility", t, func() {
		Convey("Should returns succeeded response", func() {
			response := []k8s.K8sServiceInfo{{Name: "testName"}}
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().SetServicePublicVisibilityByServiceId(testCreds, request.OrganizationGuid,
				request.SpaceGuid, request.ServiceId, request.Visibility).Return(response, nil)

			rr := sendRequest("POST", requestPath, marshallToJson(t, request), r)
			assertResponse(rr, "", 202)

			serviceRespone := []k8s.K8sServiceInfo{}
			err := readJson(rr, &serviceRespone)

			So(err, ShouldBeNil)
			So(len(serviceRespone), ShouldEqual, 1)
			So(serviceRespone, ShouldResemble, response)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().SetServicePublicVisibilityByServiceId(testCreds, request.OrganizationGuid, request.SpaceGuid,
				request.ServiceId, request.Visibility).Return([]k8s.K8sServiceInfo{}, testError)
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())
			rr := sendRequest("POST", requestPath, marshallToJson(t, request), r)

			assertResponse(rr, "", 500)
		})

		Convey("Should returns error when incorete request body", func() {
			mockStateService.EXPECT().ReportProgress(gomock.Any(), "FAILED", gomock.Any())

			rr := sendRequest("POST", requestPath, []byte("{WrongJson]"), r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestGetSecret(t *testing.T) {
	r, _, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Get(URLsecretPath, (*Context).GetSecret)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/secret/" + tst.TestSecretName

	Convey("Test GetSecret", t, func() {
		Convey("Should returns succeeded response", func() {
			response := tst.GetTestSecret()
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetSecret(testCreds, tst.TestSecretName).Return(&response, nil)

			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 200)

			apiResponse := api.Secret{}
			err := readJson(rr, &apiResponse)

			So(err, ShouldBeNil)
			So(apiResponse, ShouldResemble, response)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().GetSecret(testCreds, tst.TestSecretName).Return(
				&api.Secret{}, testError)

			rr := sendRequest("GET", requestPath, nil, r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestCreateSecret(t *testing.T) {
	r, _, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Post(URLsecretPath, (*Context).CreateSecret)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/secret/" + tst.TestSecretName

	request := tst.GetTestSecret()

	Convey("Test CreateSecret", t, func() {
		Convey("Should returns succeeded response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().CreateSecret(testCreds, request).Return(nil)

			rr := sendRequest("POST", requestPath, marshallToJson(t, request), r)
			assertResponse(rr, "", 200)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().CreateSecret(testCreds, request).Return(testError)

			rr := sendRequest("POST", requestPath, marshallToJson(t, request), r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestUpdateSecret(t *testing.T) {
	r, _, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Put(URLsecretPath, (*Context).UpdateSecret)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/secret/" + tst.TestSecretName

	request := tst.GetTestSecret()

	Convey("Test UpdateSecret", t, func() {
		Convey("Should returns succeeded response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().UpdateSecret(testCreds, request).Return(nil)

			rr := sendRequest("PUT", requestPath, marshallToJson(t, request), r)
			assertResponse(rr, "", 200)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().UpdateSecret(testCreds, request).Return(testError)

			rr := sendRequest("PUT", requestPath, marshallToJson(t, request), r)
			assertResponse(rr, "", 500)
		})
	})
}

func TestDeleteSecret(t *testing.T) {
	r, _, mockKubernetesApi, _, mockCreatorConnector := prepareMocksAndRouter(t)
	r.Delete(URLsecretPath, (*Context).DeleteSecret)

	requestPath := "/rest/kubernetes/" + tst.TestOrgGuid + "/secret/" + tst.TestSecretName

	Convey("Test DeleteSecret", t, func() {
		Convey("Should returns succeeded response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().DeleteSecret(testCreds, tst.TestSecretName).Return(nil)

			rr := sendRequest("DELETE", requestPath, nil, r)
			assertResponse(rr, "", 200)
		})

		Convey("Should returns failed response", func() {
			mockCreatorConnector.EXPECT().GetCluster(tst.TestOrgGuid).Return(200, testCreds, nil)
			mockKubernetesApi.EXPECT().DeleteSecret(testCreds, tst.TestSecretName).Return(testError)

			rr := sendRequest("DELETE", requestPath, nil, r)
			assertResponse(rr, "", 500)
		})
	})
}

func sendRequest(rType, path string, body []byte, r *web.Router) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(rType, path, bytes.NewReader(body))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func assertResponse(rr *httptest.ResponseRecorder, body string, code int) {
	if body != "" {
		So(strings.TrimSpace(string(rr.Body.Bytes())), ShouldContainSubstring, body)
	}
	So(rr.Code, ShouldEqual, code)
}

func marshallToJson(t *testing.T, serviceInstance interface{}) []byte {
	if body, err := json.Marshal(serviceInstance); err != nil {
		t.Errorf(err.Error())
		t.FailNow()
		return nil
	} else {
		return body
	}
}

func readJson(rr *httptest.ResponseRecorder, retstruct interface{}) error {
	err := json.Unmarshal(rr.Body.Bytes(), &retstruct)
	if err != nil {
		return err
	}
	return nil
}
