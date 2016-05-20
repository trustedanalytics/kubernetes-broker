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

USER=${POSTGRES_USERNAME:-pgadmin}
PASS=${POSTGRES_PASSWORD:-$(pwgen -s -1 16)}
DB=${POSTGRES_DBNAME:-}
EXTENSIONS=${POSTGRES_EXTENSIONS:-}

cd /var/lib/postgresql
DATA_DIR=/var/lib/postgresql/data
# Start PostgreSQL service
sudo -u postgres /usr/lib/postgresql/9.3/bin/postgres -D $DATA_DIR &

while ! sudo -u postgres psql -q -c "select true;"; do sleep 1; done

# Create user
echo "Creating user: \"$USER\"..."
sudo -u postgres psql -q -c "DROP ROLE IF EXISTS \"$USER\";"
sudo -u postgres psql -q <<-EOF
    CREATE ROLE "$USER" WITH ENCRYPTED PASSWORD '$PASS';
    ALTER ROLE "$USER" WITH ENCRYPTED PASSWORD '$PASS';
    ALTER ROLE "$USER" WITH SUPERUSER;
    ALTER ROLE "$USER" WITH LOGIN;
EOF

# Create dabatase
if [ ! -z "$DB" ]; then
    echo "Creating database: \"$DB\"..."
    sudo -u postgres psql -q <<-EOF
    CREATE DATABASE "$DB" WITH OWNER="$USER" ENCODING='UTF8';
    GRANT ALL ON DATABASE "$DB" TO "$USER"
EOF

    if [[ ! -z "$EXTENSIONS" ]]; then
        for extension in $EXTENSIONS; do
            echo "Installing extension \"$extension\" for database \"$DB\"..."
            sudo -u postgres psql -q "$DB" -c "CREATE EXTENSION \"$extension\";"
        done
    fi
fi

# Stop PostgreSQL service
sudo -u postgres /usr/lib/postgresql/9.3/bin/pg_ctl stop -m fast -w -D $DATA_DIR

echo "========================================================================"
echo "PostgreSQL User: \"$USER\""
echo "PostgreSQL Password: \"$PASS\""
if [ ! -z $DB ]; then
    echo "PostgreSQL Database: \"$DB\""
    if [[ ! -z "$EXTENSIONS" ]]; then
        echo "PostgreSQL Extensions: \"$EXTENSIONS\""
    fi
fi
echo "========================================================================"

#store file in persistent store
touch /var/lib/postgresql/.firstrun
