#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Healthcheck checks a ping to container name and compose service name
# because both network aliases are defined if the add-on is created.
CONTAINER_NAME=uc-addon-communication-receiver
COMPOSE_SERVICE_NAME=ucaomtest-addon-communication-receiver
ping -c 1 ${CONTAINER_NAME} && ping -c 1 ${COMPOSE_SERVICE_NAME}; if [ $? -eq 1 ]; then exit 1; else exit 0; fi
