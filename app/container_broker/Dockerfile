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

FROM debian:8.4
MAINTAINER Jakub Wierzbowski <jakub.a.wierzbowski@intel.com>

RUN mkdir -p /opt/app /opt/app/catalogData/
ADD container_broker /opt/app
ADD catalogData/ /opt/app/catalogData/

RUN chmod +x /opt/app/container_broker

WORKDIR /opt/app/

ENV CONTAINER_BROKER_PORT "8082"
ENV CONTAINER_BROKER_USER "admin"
ENV CONTAINER_BROKER_PASS "password"
ENV CONTAINER_BROKER_SSL_ACTIVE "false"

ENV INSECURE_SKIP_VERIFY true
ENV BROKER_LOG_LEVEL DEBUG

ENV K8S_API_ADDRESS "http://localhost:8080"
ENV K8S_API_USERNAME ""
ENV K8S_API_PASSWORD ""
ENV KUBE_SSL_ACTIVE false

EXPOSE 8082

ENTRYPOINT ["/opt/app/container_broker"]
CMD [""]