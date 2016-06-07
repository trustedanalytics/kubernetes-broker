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
package catalog

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	tst "github.com/trustedanalytics/kubernetes-broker/test"
)

var testCatalogPath = tst.GetTestCatalogPath("/catalog")

func TestGetOrgIdAndSpaceIdFromCfByServiceInstanceIdJson(t *testing.T) {
	Convey("Test GetOrgIdAndSpaceIdFromCfByServiceInstanceIdJson", t, func() {
		Convey("Should returns proper response", func() {
			CatalogPath = testCatalogPath
			var result = GetAvailableServicesMetadata()

			So(len(result.Services), ShouldEqual, 1)
			So(result.Services[0].Id, ShouldEqual, tst.TestServiceId)
			So(len(result.Services[0].Tags), ShouldEqual, 3)
			So(len(result.Services[0].Plans), ShouldEqual, 1)
			So(result.Services[0].Plans[0].Id, ShouldEqual, tst.TestPlanId)
		})

		Convey("Should returns error when parsing catalog directory", func() {
			CatalogPath = "/CATALOG_totalyWrong_Path"
			So(func() {
				GetAvailableServicesMetadata()
			}, ShouldPanic)
		})

		Convey("Should load data only once", func() {
			services_metadata := &ServicesMetadata{}
			GLOBAL_SERVICES_METADATA = services_metadata

			GetAvailableServicesMetadata()
			// here we exepecting that GLOBAL_SERVICES_METADATA was not overwrited
			So(GLOBAL_SERVICES_METADATA, ShouldPointTo, services_metadata)
		})

		Reset(func() {
			GLOBAL_SERVICES_METADATA = nil
		})
	})
}

func TestWhatToCreateByServiceAndPlanId(t *testing.T) {
	Convey("Test WhatToCreateByServiceAndPlanId", t, func() {
		CatalogPath = testCatalogPath

		Convey("Should returns proper response", func() {
			service, plan, err := WhatToCreateByServiceAndPlanId(tst.TestServiceId, tst.TestPlanId)
			So(err, ShouldBeNil)
			So(service.Id, ShouldEqual, tst.TestServiceId)
			So(plan.Id, ShouldEqual, tst.TestPlanId)
		})

		Convey("Should returns error when service not found", func() {
			_, _, err := WhatToCreateByServiceAndPlanId("fakeServiceName", tst.TestPlanId)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No such service by ID")
		})

		Convey("Should returns error when plan not found", func() {
			service, _, err := WhatToCreateByServiceAndPlanId(tst.TestServiceId, "fakePlanId")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "No such plan by ID")
			So(service.Id, ShouldEqual, tst.TestServiceId)
		})
	})
}

func TestGetKubernetesBlueprintForServiceAndPlan(t *testing.T) {
	CatalogPath = testCatalogPath

	Convey("Test GetKubernetesBlueprintForServiceAndPlan", t, func() {
		Convey("Should returns proper response", func() {
			result, err := GetKubernetesBlueprint(testCatalogPath, tst.TestInternalServiceId, tst.TestInternalPlanId, "")
			So(err, ShouldBeNil)
			So(len(result.ServiceJson), ShouldEqual, 1)
			So(result.Id, ShouldEqual, 0)

		})

		Convey("Should returns error when service not exist", func() {
			_, err := GetKubernetesBlueprint(testCatalogPath, "FakeService", "fakePlan", "")

			So(err, ShouldNotBeNil)
		})
	})
}
