#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

PROJECT_NAME=uc-aom-mit

if [ $# -ne 1 ]; then
  echo "Unknown command '$*'"
  exit 1
fi

case "$1" in
  down)
    docker stop uc-aom-mit-migration-test-runner
    docker rm uc-aom-mit-migration-test-runner
    docker compose \
      --project-name ${PROJECT_NAME}\
       -f ./scripts/docker/docker-compose-migration-test.yml \
      down --remove-orphans -v --rmi "all"
    ;;
  run)
    docker exec \
        uc-aom-mit-migration-test-runner \
        make migration-test

    ;;
  up)
    docker compose \
      --project-name ${PROJECT_NAME} \
      -f ./scripts/docker/docker-compose-migration-test.yml \
      up -d --build
    docker build --tag uc-aom-mit-migration-test-runner:latest -f ./scripts/docker/migration-test-runner.Dockerfile .

    docker run \
        --env DEBIAN_FRONTEND=noninteractive \
        --name uc-aom-mit-migration-test-runner \
        --network ${PROJECT_NAME}_default \
        --volume "$PWD":/go/src/uc-aom/ \
        --volume /var/run/docker.sock:/var/run/docker.sock \
        --volume ${PROJECT_NAME}_localcatalog:/var/lib/uc-aom \
        --volume ${PROJECT_NAME}_remotecatalog:/var/run/uc-aom \
        --volume ${PROJECT_NAME}_proxyroutes:/var/lib/nginx \
        --volume ${PROJECT_NAME}_cache:/var/cache/uc-aom \
        --workdir /go/src/uc-aom \
        --tty \
        --detach \
        uc-aom-mit-migration-test-runner:latest


    # Fill system integration test registry with docker images of test add-ons.
    docker exec \
        uc-aom-mit-migration-test-runner \
        scripts/build-and-push-docker-images.sh localhost:5100
    ;;
  *)
  echo "Unknown command '$1'"
  exit 1
    ;;
esac
