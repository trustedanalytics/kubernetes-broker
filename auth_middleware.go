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
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/dvsekhvalnov/jose2go"
	jwtRsa "github.com/dvsekhvalnov/jose2go/keys/rsa"
	"github.com/gocraft/web"

	brokerHttp "github.com/trustedanalytics/kubernetes-broker/http"
)

var publicKey *rsa.PublicKey
var TokenKeyURL string

func (c *Context) JWTAuthorizeMiddleware(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	logger.Info("Trying to access url ", req.URL.Path, " by JWTAuthorize")
	if ok := isUserAuthorized(req); ok {
		next(rw, req)
		return
	} else {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
}

func (c *Context) BasicAuthorizeMiddleware(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	logger.Info("Trying to access url ", req.URL.Path, " by BasicAuthorize")
	username, password, is_ok := req.BasicAuth()
	if !is_ok || username != cfenv.CurrentEnv()["AUTH_USER"] || password != cfenv.CurrentEnv()["AUTH_PASS"] {
		logger.Info("EnforceAuthMiddleware - BasicAuth: Invalid Basic Auth credentials")
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(rw, "%s", `{"error":"invalid basic auth credentials"}`)
		return
	}
	logger.Info("EnforceAuthMiddleware - BasicAuth: User authenticated as ", username)
	next(rw, req)
}

func isUserAuthorized(req *web.Request) bool {
	validScope := "console.admin"

	var err error
	if publicKey == nil {
		publicKey, err = getJWTPublicKey()
		if err != nil {
			logger.Warning("EnforceAuthMiddleware - JWT: Can't get Public key!", err)
			return false
		}
	}

	tapToken, err := parseJWTToken(req.Header.Get("Authorization"))
	if err != nil {
		logger.Warning("EnforceAuthMiddleware - JWT: Invalid token!", err)
		return false
	}

	for _, scope := range tapToken.Scope {
		if scope == validScope {
			logger.Info("EnforceAuthMiddleware - JWT: User authenticated as", tapToken.Username)
			return true
		}
	}
	logger.Info("EnforceAuthMiddleware - JWT: User has not access! Username:", tapToken.Username)
	return false
}

func getJWTPublicKey() (*rsa.PublicKey, error) {
	_, publicKeyRaw, err := brokerHttp.RestGET(TokenKeyURL, nil, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	type PublicKeyServiceResponse struct {
		Algorithm string `json:"alg"`
		Key       string `json:"value"`
	}
	response := PublicKeyServiceResponse{}

	err = json.Unmarshal(publicKeyRaw, &response)
	if err != nil {
		return nil, err
	}
	return jwtRsa.ReadPublic([]byte(response.Key))
}

func parseJWTToken(authHeader string) (*TapJWTToken, error) {
	if authHeader != "" && len(strings.Split(authHeader, " ")) > 1 {
		token := strings.Split(authHeader, " ")[1]
		payload, _, err := jose.Decode(token, publicKey)
		if err != nil {
			return nil, err
		}
		tapToken := &TapJWTToken{}
		err = json.Unmarshal([]byte(payload), tapToken)
		return tapToken, err
	} else {
		return nil, errors.New("Authorisation header incorrect!")
	}
}
