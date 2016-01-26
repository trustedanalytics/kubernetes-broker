#!/bin/bash
source scripts/cf-config.sh

cf  logs --recent $BROKER
