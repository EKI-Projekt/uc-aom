#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

echo $PWD

REGISTRY_USERNAME=$1
REGISTRY_PWD=$2

# clean all data in docker volumes before start the service
scripts/clean-docker-volumes.sh

# build and push test docker images
scripts/build-and-push-docker-images.sh

# push AddOns to wmucdev registry
scripts/initialize-registry-with-addon.sh \
  "prod" \
  "${REGISTRY_USERNAME}" \
  "${REGISTRY_PWD}" \
  "127.0.0.1" \
  "localhost"
