#!/bin/bash
# Copyright (c) 2016 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
export TEMPLATE_REPOSITORY_PORT="8082"
export TEMPLATE_REPOSITORY_USER="admin"
export TEMPLATE_REPOSITORY_PASS="password"
export TEMPLATE_REPOSITORY_SSL_ACTIVE="false"
export TEMPLATE_REPOSITORY_SSL_CERT_FILE_LOCATION="cert.pem"
export TEMPLATE_REPOSITORY_SSL_KEY_FILE_LOCATION="key.pem"
export TEMPLATE_REPOSITORY_SSL_CA_FILE_LOCATION="ca.pem"

export CONTAINER_BROKER_PORT="8081"
export CONTAINER_BROKER_USER="admin"
export CONTAINER_BROKER_PASS="password"
export CONTAINER_BROKER_SSL_ACTIVE="false"
export CONTAINER_BROKER_SSL_CERT_FILE_LOCATION="cert.pem"
export CONTAINER_BROKER_SSL_KEY_FILE_LOCATION="key.pem"
export CHECK_JOB_INTERVAL_SEC="10"

export K8S_API_ADDRESS="http://localhost:8080"
export K8S_API_USERNAME=""
export K8S_API_PASSWORD=""
export KUBE_SSL_ACTIVE=false

export INSECURE_SKIP_VERIFY=true
export BROKER_LOG_LEVEL=DEBUG
