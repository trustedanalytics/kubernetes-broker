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
export K8S_API_ADDRESS=http://localhost:8080
export VCAP_APPLICATION='{"port":8081, "host":"0.0.0.0"}'

export AUTH_USER="admin"
export AUTH_PASS="password"

export INSECURE_SKIP_VERIFY=true
export ACCEPT_INCOMPLETE=false

export MAX_ORG_QUOTA=10
export BROKER_LOG_LEVEL=DEBUG

export WAIT_BEFORE_NEXT_PV_CHECK_SEC=120
export WAIT_BEFORE_REMOVE_CLUSTER_SEC=600

export VCAP_SERVICES='{
"user-provided": [{
    "credentials": {
     "apiEndpoint": "http://random_endpoint_.eu",
     "clientId": "clientRandomId",
     "clientSecret": "clientRandomSecret",
     "tokenUri": "http://random.eu/oauth/token",
     "tokenKey": "http://uaa.gotapaas.eu/token_key"
    },
    "label": "user-provided",
    "name": "sso"
},
{
    "credentials": {
     "password": "admin",
     "url": "kube-tap.dev-krb.gotapaas.eu",
     "username": "admin"
    },
    "label": "user-provided",
    "name": "kubernetes-creator-credentials"
   }]
}'

export KUBE_SSL_ACTIVE=false

$GOPATH/bin/tap
exit $?
