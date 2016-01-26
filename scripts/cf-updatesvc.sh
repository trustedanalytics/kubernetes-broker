#!/bin/bash -x
source scripts/cf-config.sh

#cf purge-service-offering elasticsearch13-multinode
#cf purge-service-offering mongodb-clustered
#cf purge-service-offering mysql56-clustered
#cf purge-service-offering postgresql94-clustered
#cf purge-service-offering cassandra21-multinode
#cf delete-service-broker $BROKER

cf create-service-broker $BROKER $AUTH_USER $AUTH_PASS https://$BROKER.$DOMAIN
cf update-service-broker $BROKER $AUTH_USER $AUTH_PASS https://$BROKER.$DOMAIN

cf enable-service-access elk-multinode
cf enable-service-access mongodb-clustered
cf enable-service-access mysql56-clustered
cf enable-service-access postgresql94-clustered
cf enable-service-access cassandra21-multinode

