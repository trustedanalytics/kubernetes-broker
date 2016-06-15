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

package api

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dvsekhvalnov/jose2go"
	jwtRsa "github.com/dvsekhvalnov/jose2go/keys/rsa"
	"github.com/gocraft/web"
)

type TapJWTToken struct {
	Jti       string   `json:"jti"`
	Sub       string   `json:"sub"`
	Scope     []string `json:"scope"`
	ClientId  string   `json:"client_id"`
	Cid       string   `json:"cid"`
	Azp       string   `json:"azp"`
	GrantType string   `json:"grant_type"`
	UserId    string   `json:"user_id"`
	Username  string   `json:"user_name"`
	Email     string   `json:"email"`
	RevSig    string   `json:"rev_sig"`
	Iat       int64    `json:"iat"`
	Exp       int64    `json:"exp"`
	Iss       string   `json:"iss"`
	Zid       string   `json:"zid"`
	Aud       []string `json:"aud"`
}

var publicKey *rsa.PublicKey

func (c *Context) BasicAuthorizeMiddleware(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	logger.Info("Trying to access url ", req.URL.Path, " by BasicAuthorize")
	username, password, is_ok := req.BasicAuth()
	if !is_ok || username != os.Getenv("TEMPLATE_REPOSITORY_USER") || password != os.Getenv("TEMPLATE_REPOSITORY_PASS") {
		logger.Info("EnforceAuthMiddleware - BasicAuth: Invalid Basic Auth credentials")
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(rw, "%s", `{"error":"invalid basic auth credentials"}`)
		return
	}
	logger.Info("EnforceAuthMiddleware - BasicAuth: User authenticated as ", username)
	next(rw, req)
}

func (c *Context) JWTAuthorizeMiddleware(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	logger.Info("Trying to access url ", req.URL.Path, " by JWTAuthorize")
	if ok := isUserAuthorized(req); ok {
		next(rw, req)
		return
	} else {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
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
	publicKeyFile, err := ioutil.ReadFile(os.Getenv("JWT_PUBLIC_KEY_FILE_LOCATION"))
	if err != nil {
		return nil, err
	}
	return jwtRsa.ReadPublic(publicKeyFile)
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
