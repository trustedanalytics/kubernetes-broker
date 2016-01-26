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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cloudfoundry-community/go-cfenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
)

type CloudApi interface {
	GetOrgIdAndSpaceIdFromCfByServiceInstanceId(service_instance_id string) (string, string, error)
	GetSpaceDetailsFromCfBySpaceId(space_id string) (CfSpaceDetails, error)
	GetInstanceDetailsFromCfById(instance_id string) (CfInstanceDetails, error)
	GetServiceBrokerByName(brokerName string) (FindServiceBrokerResponse, error)
	UpdateServiceBroker() (ServiceBroker, error)
}

type CfApi struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Endpoint     string
	baseUrl      string
	client       *http.Client
}

func NewCFApiClient(ClientID, ClientSecret, TokenURL, Endpoint string) *CfApi {
	client, baseurl, err := cfApiClientAndUrl(ClientID, ClientSecret, TokenURL, Endpoint)
	if err != nil {
		logger.Fatal("Can't create http client for CloundFoundry API!", err)
	}
	return &CfApi{ClientID, ClientSecret, TokenKeyURL, Endpoint, baseurl, client}
}

func cfApiClientAndUrl(ClientID, ClientSecret, TokenURL, Endpoint string) (*http.Client, string, error) {
	tokenConfig := &clientcredentials.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		Scopes:       []string{},
		TokenURL:     TokenURL,
	}
	return tokenConfig.Client(oauth2.NoContext), Endpoint, nil
}

func (c *CfApi) GetOrgIdAndSpaceIdFromCfByServiceInstanceId(service_instance_id string) (string, string, error) {
	inst_details, err := c.GetInstanceDetailsFromCfById(service_instance_id)
	if err != nil {
		logger.Error("GetInstanceDetailsFromCfById: ", err)
		return "", "", err
	}
	space_details, err := c.GetSpaceDetailsFromCfBySpaceId(inst_details.Entity.SpaceGuid)
	if err != nil {
		logger.Error("GetSpaceDetailsFromCfBySpaceId: ", err)
		return "", "", err
	}
	return space_details.Entity.OrgGuid, inst_details.Entity.SpaceGuid, nil
}

type CfSpaceEntityDetails struct {
	Name                string      `json:"name"`
	OrgGuid             string      `json:"organization_guid"`
	SpaceQuota          interface{} `json:"space_quota_definition_guid"`
	AllowSsh            bool        `json:"allow_ssh"`
	OrgUrl              string      `json:"organization_url"`
	DevUrl              string      `json:"developers_url"`
	MgmtUrl             string      `json:"managers_url"`
	AuditUrl            string      `json:"auditors_url"`
	AppsUrl             string      `json:"apps_url"`
	RoutesUrl           string      `json:"routes_url"`
	DomainsUrl          string      `json:"domains_url"`
	ServiceInstancesUrl string      `json:"service_instances_url"`
	AppEventsUrl        string      `json:"app_events_url"`
	EventsUrl           string      `json:"events_url"`
	SecurityGroupsUrl   string      `json:"security_groups_url"`
}

type CfSpaceDetails struct {
	Metadata interface{}          `json:"metadata"`
	Entity   CfSpaceEntityDetails `json:"entity"`
}

// https://apidocs.cloudfoundry.org/228/spaces/retrieve_a_particular_space.html
func (c *CfApi) GetSpaceDetailsFromCfBySpaceId(space_id string) (CfSpaceDetails, error) {
	// GET /v2/spaces/11675800-6624-438e-9c88-d34f0e957609
	space_details := CfSpaceDetails{}

	url := c.baseUrl + "/v2/spaces/" + space_id
	logger.Debug(fmt.Sprintf("GetSpaceDetailsFromCfBySpaceId Accesing CF, url: %s, space_id: %s", url, space_id))

	status, body_b, err := brokerHttp.RestGET(url, nil, c.client)
	if status != 200 {
		logger.Error("Status code is invalid: ", status)
		return space_details, errors.New("Status code is invalid")
	}

	err = json.Unmarshal(body_b, &space_details)
	if err != nil {
		logger.Error("Error: ", err)
		return space_details, err
	}
	return space_details, nil
}

type CfEntityDetails struct {
	Name               string                 `json:"name"`
	Credentials        map[string]interface{} `json:"credentials"`
	ServicePlanGuid    string                 `json:"service_plan_guid"`
	SpaceGuid          string                 `json:"space_guid"`
	GatewayData        interface{}            `json:"gateway_data"`
	DashboardUrl       string                 `json:"dashboard_url"`
	Type               string                 `json:"type"`
	LastOperation      interface{}            `json:"last_operation"`
	Tags               []string               `json:"tags"`
	SpaceUrl           string                 `json:"space_url"`
	ServicePlanUrl     string                 `json:"service_plan_url"`
	ServiceBindingsUrl string                 `json:"service_bindings_url"`
	ServiceKeysUrl     string                 `json:"service_keys_url"`
	RoutesUrl          string                 `json:"routes_url"`
}

type CfInstanceDetails struct {
	Metadata interface{}     `json:"metadata"`
	Entity   CfEntityDetails `json:"entity"`
}

// https://apidocs.cloudfoundry.org/228/service_instances/retrieve_a_particular_service_instance.html
func (c *CfApi) GetInstanceDetailsFromCfById(instance_id string) (CfInstanceDetails, error) {
	// GET /v2/service_instances/236db61a-f603-4a1f-b8eb-3846c277d441
	inst_details := CfInstanceDetails{}

	url := c.baseUrl + "/v2/service_instances/" + instance_id
	logger.Debug(fmt.Sprintf("GetInstanceDetailsFromCfById Accesing CF, url: %s, instance_id: %s", url, instance_id))

	status, body_b, err := brokerHttp.RestGET(url, nil, c.client)
	if status != 200 {
		logger.Error("Status code is invalid: ", status)
		return inst_details, errors.New("Status code is invalid")
	}

	err = json.Unmarshal(body_b, &inst_details)
	if err != nil {
		logger.Error("Error: ", err)
		return inst_details, err
	}
	return inst_details, nil
}

type FindServiceBrokerResponse struct {
	TotalResult int `json:"total_results"`
	TotalPages  int `json:"total_pages"`
	Resources   []ServiceBroker
}

type ServiceBroker struct {
	Metadata Metadata
	Entity   Entity
}

type Metadata struct {
	Guid string `json:"guid"`
	Url  string `json:"url,omitempty"`
}

type Entity struct {
	Name     string `json:"name"`
	Password string `json:"auth_password"`
	Username string `json:"auth_username"`
	Url      string `json:"broker_url"`
}

func (c *CfApi) GetServiceBrokerByName(brokerName string) (FindServiceBrokerResponse, error) {
	result := FindServiceBrokerResponse{}

	url := c.baseUrl + fmt.Sprintf("/v2/service_brokers?q=%s", url.QueryEscape("name:"+brokerName))
	logger.Debug(fmt.Sprintf("GetServiceBrokerByName Accesing CF, url: %s, broker name: %s", url, brokerName))

	status, body_b, err := brokerHttp.RestGET(url, nil, c.client)
	if status != 200 {
		logger.Error("Status code is invalid: ", status)
		return result, errors.New("Status code is invalid")
	}

	err = json.Unmarshal(body_b, &result)
	if err != nil {
		logger.Error("GetServiceBrokerByName Error: ", err)
		return result, err
	}

	logger.Debug("Parsed GetServiceBrokerByName response:", result)
	return result, nil
}

func (c *CfApi) UpdateServiceBroker() (ServiceBroker, error) {
	result := ServiceBroker{}

	cfApp, err := cfenv.Current()
	if err != nil {
		return result, err
	}

	currentBrokerName := cfApp.Name
	currentBrokerPassword := cfenv.CurrentEnv()["AUTH_PASS"]

	foundBrokers, err := c.GetServiceBrokerByName(currentBrokerName)
	if err != nil {
		logger.Error("Error: ", err)
		return result, err
	}

	if foundBrokers.TotalResult != 1 || len(foundBrokers.Resources) != 1 {
		return result, errors.New(fmt.Sprintf("Can't Update broker - number of found brokers is not equal to 1."+
			"Broker name: %s, No of found results: %n", currentBrokerName, foundBrokers.TotalResult))
	}

	body := fmt.Sprintf(
		`{"broker_url":"%s","auth_username":"%s","auth_password":"%s"}`,
		foundBrokers.Resources[0].Entity.Url, foundBrokers.Resources[0].Entity.Username, currentBrokerPassword,
	)

	url := c.baseUrl + fmt.Sprintf("/v2/service_brokers/%s", foundBrokers.Resources[0].Metadata.Guid)
	logger.Debug(fmt.Sprintf("UpdateServiceBrokerByName Accesing CF, url: %s, broker name: %s", url, currentBrokerName))

	status, body_b, err := brokerHttp.RestPUT(url, body, nil, c.client)
	if status != 200 {
		logger.Error("Status code is invalid: ", status)
		return result, errors.New("Status code is invalid")
	}
	err = json.Unmarshal(body_b, &result)
	if err != nil {
		logger.Error("Error: ", err)
		return result, err
	}
	logger.Debug("Parsed UpdateServiceBrokerByName response:", result)
	return result, nil
}
