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
package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type BasicAuth struct {
	User     string
	Password string
}

func RestGET(url string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	return makeRequest("GET", url, "", "application/json", basicAuth, client)
}

func RestPUT(url, body string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	return makeRequest("PUT", url, body, "application/json", basicAuth, client)
}

func RestPOST(url, body string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	return makeRequest("POST", url, body, "application/json", basicAuth, client)
}

func RestDELETE(url, body string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	return makeRequest("DELETE", url, body, "", basicAuth, client)
}

func RestPATCH(url, body string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	return makeRequest("PATCH", url, body, "application/json-patch+json", basicAuth, client)
}

func makeRequest(reqType, url, body, contentType string, basicAuth *BasicAuth, client *http.Client) (int, []byte, error) {
	logger.Info("Doing:  ", reqType, url)

	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(reqType, url, bytes.NewBuffer([]byte(body)))
	} else {
		req, _ = http.NewRequest(reqType, url, nil)
	}

	if basicAuth != nil {
		req.SetBasicAuth(basicAuth.User, basicAuth.Password)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("ERROR: Make http request "+reqType, err)
		return -1, nil, err
	}
	ret_code := resp.StatusCode
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("ERROR: Make http request "+reqType, err)
		return -1, nil, err
	}

	logger.Info("CODE:", ret_code, "BODY:", string(data))
	return ret_code, data, nil
}
