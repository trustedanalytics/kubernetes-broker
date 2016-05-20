#!/bin/bash

# Copyright (c) 2014 Ferran Rodenas
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

cd /var/lib/postgresql

# Initialize data directory
DATA_DIR=/var/lib/postgresql/data


if [ ! -f $DATA_DIR/postgresql.conf ]; then
    mkdir -p $DATA_DIR
    chown postgres:postgres $DATA_DIR

    sudo -u postgres /usr/lib/postgresql/9.4/bin/initdb -E utf8 --locale en_US.UTF-8 -D $DATA_DIR
    sed -i -e"s/^#listen_addresses =.*$/listen_addresses = '*'/" $DATA_DIR/postgresql.conf
    echo  "shared_preload_libraries='pg_stat_statements'">> $DATA_DIR/postgresql.conf
    echo "host    all    all    0.0.0.0/0    md5" >> $DATA_DIR/pg_hba.conf

    mkdir -p $DATA_DIR/pg_log
fi
chown -R postgres:postgres $DATA_DIR
chmod -R 700 $DATA_DIR

# Initialize first run
if [ ! -e /var/lib/postgresql/.firstrun ]
then
    echo "Initialize postgresql database"
    /scripts/first_run.sh
else
    echo "Omitting initialization of database"
fi

# Start PostgreSQL
echo "Starting PostgreSQL..."
sudo -u postgres /usr/lib/postgresql/9.4/bin/postgres -D $DATA_DIR
