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

# Portions Copyright (c) 2016 Intel Corporation
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
function configure_backend_hosts {
    sed -i 's/REPLICATION_USER/'"${REPLICATION_USER}"'/g' ${PGPOOL_CONF_HOME}/pgpool.conf &&
    sed -i 's/REPLICATION_PASS/'"${REPLICATION_PASS}"'/g' ${PGPOOL_CONF_HOME}/pgpool.conf &&

    MASTER_HOST_NAME=${MASTER_HOST_NAME^^} &&
    MASTER_SERVICE_HOST=${!MASTER_HOST_NAME} &&

    SLAVE_1_HOST_NAME=${SLAVE_1_HOST_NAME^^} &&
    SLAVE_1_SERVICE_HOST=${!SLAVE_1_HOST_NAME} &&

    SLAVE_2_HOST_NAME=${SLAVE_2_HOST_NAME^^} &&
    SLAVE_2_SERVICE_HOST=${!SLAVE_2_HOST_NAME} &&

    sed -i 's/MASTER_SERVICE_HOST/'"${MASTER_SERVICE_HOST}"'/g' ${PGPOOL_CONF_HOME}/pgpool.conf &&
    sed -i 's/SLAVE_1_SERVICE_HOST/'"${SLAVE_1_SERVICE_HOST}"'/g' ${PGPOOL_CONF_HOME}/pgpool.conf &&
    sed -i 's/SLAVE_2_SERVICE_HOST/'"${SLAVE_2_SERVICE_HOST}"'/g' ${PGPOOL_CONF_HOME}/pgpool.conf
}

function start_pgpool {
    pgpool -n -d
}

function prepare_pgpool_conf {
    cd ${PGPOOL_CONF_HOME} &&
    #generate pool_passwd
    pg_md5 -u ${REPLICATION_USER} ${REPLICATION_PASS} -m -f ${PGPOOL_CONF_HOME}/pgpool.conf &&
    #generate pcp.conf
    echo ${REPLICATION_USER}:`pg_md5 ${REPLICATION_PASS}` > ${PGPOOL_CONF_HOME}/pcp.conf &&
    #enable access to all db from remote host for REPLICATION_USER, with password
    echo "host all ${REPLICATION_USER} ::1/128 md5" >> ${PGPOOL_CONF_HOME}/pool_hba.conf
}

function download_dependencies {
    # We'll need postgresql-server-dev-9.4 to build pgpool extensions,
    echo "Downloading required system dependencies..." &&
    apt-get update &&
    apt-get install -y postgresql-server-dev-9.4 curl build-essential &&
    curl -L -o pgpool-II-${PGPOOL_VERSION}.tar.gz http://www.pgpool.net/download.php?f=pgpool-II-${PGPOOL_VERSION}.tar.gz &&
    tar zxvf pgpool-II-${PGPOOL_VERSION}.tar.gz
}

function compile_pgpool {
    echo "Compiling pgpool package..." &&
    cd /pgpool-II-${PGPOOL_VERSION} &&
    ./configure --with-openssl &&
    make &&
    make install &&
    # Build pgpool2 extensions for postgres
    cd /pgpool-II-${PGPOOL_VERSION}/src/sql &&
    make &&
    make install &&
    ldconfig
}

download_dependencies &&
compile_pgpool &&
prepare_pgpool_conf &&
configure_backend_hosts &&
start_pgpool
