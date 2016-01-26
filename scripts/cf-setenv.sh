#!/bin/bash
source scripts/cf-config.sh

cf  set-env $BROKER AUTH_PASS "$AUTH_PASS"
cf  set-env $BROKER AUTH_USER "$AUTH_USER"
cf  set-env $BROKER ACCEPT_INCOMPLETE true
cf  set-env $BROKER INSECURE_SKIP_VERIFY "$INSECURE_SKIP_VERIFY"

