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

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gocraft/web"

	"github.com/trustedanalytics/kubernetes-broker/logger"
)

var logger = logger_wrapper.InitLogger("api")

func ReadJson(req *web.Request, retstruct interface{}) error {
	var err error
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("Error reading request body:", err)
		return err
	}
	b := []byte(body)
	err = json.Unmarshal(b, &retstruct)
	if err != nil {
		logger.Error("Error parsing request body json:", err)
		return err
	}
	logger.Debug("Request JSON parsed as: ", retstruct)
	return nil
}

func WriteJson(rw web.ResponseWriter, response interface{}, status_code int) error {
	b, err := json.Marshal(&response)
	if err != nil {
		logger.Error("Error marshalling response:", err)
		return err
	}
	rw.Header().Set("Content-Type", "application/json")
	logger.Debug("Responding with status", status_code, " and JSON:", string(b))
	rw.WriteHeader(status_code)
	fmt.Fprintf(rw, "%s", string(b))
	return nil
}

func Respond500(rw web.ResponseWriter, err error) {
	logger.Error("Respond500: reason: error ", err)
	rw.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(rw, "%s", err.Error())
}

func Respond404(rw web.ResponseWriter, err error) {
	logger.Error("Respond404: reason: error ", err)
	rw.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(rw, "%s", err.Error())
}
