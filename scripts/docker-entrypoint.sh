#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

echo "RUN_MODE: ${RUN_MODE}"
echo $PWD

# clean all data in docker volumes before start the service
scripts/clean-docker-volumes.sh

# build and push test docker images
scripts/build-and-push-docker-images.sh

# push AddOns to registry
scripts/initialize-registry-with-addon.sh

# install files
make install REGISTRYFILE="registrycredentials_dev.json"

# create admin user in dev environment
groupadd -g 1000 admin
useradd -u 1000 -g 1000 admin

if [ -z ${RUN_MODE} ] || [ ${RUN_MODE} = "start" ]; then
    make build
    echo "service is starting..."
    make run-docker
elif [ ${RUN_MODE} = "shell" ]; then
    /bin/bash
elif [ ${RUN_MODE} = "dev" ]; then
    make dev
    echo "service is ready for development"
    # container is going to sleep infinity because otherwise it will stop immediatly
    sleep infinity
else
  echo "error: Unknown command '${RUN_MODE}'"
  exit 1
fi
