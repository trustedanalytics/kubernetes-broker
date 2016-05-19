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
FROM postgres:9.4
MAINTAINER Adrian Weber <adrian.weber@intel.com>

ENV PGPOOL_VERSION 3.4.6
ENV PGPOOL_CONF_HOME /usr/local/etc

# pgpool port
EXPOSE 5432

ADD pgpool.conf ${PGPOOL_CONF_HOME}/pgpool.conf
ADD pool_hba.conf ${PGPOOL_CONF_HOME}/pool_hba.conf

WORKDIR /
COPY start.sh /start.sh
RUN chmod +x /start.sh

RUN mkdir -p /var/run/pgpool

CMD ["/start.sh"]
