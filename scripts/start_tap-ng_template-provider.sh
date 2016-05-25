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

export HOST="0.0.0.0"
export PORT="8081"
export AUTH_USER="admin"
export AUTH_PASS="password"

export INSECURE_SKIP_VERIFY=true
export BROKER_LOG_LEVEL=DEBUG
export KUBE_SSL_ACTIVE=false

$GOPATH/bin/tap-ng_template-provider
exit $?
