#!/bin/sh

# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

NAME=uc-aom
RUN_MODE=$1
SERVICE=$2

cd $(dirname $(readlink -f "$0"))

if [ -z "${RUN_MODE}" ]; then
  RUN_MODE="start"
fi

case "${RUN_MODE}" in
  dev|shell|start)
    ARGS="-f ./docker/docker-compose-dev-env.yml"
    ;;
  tc-add-ons)
    ARGS="-f ./docker/docker-compose-tc-add-ons.yml"
    ;;
  *)
    echo "error: Unknown command '$1'"
    echo ""
    echo "Usage:"
    echo "  $0 [dev|shell|start|tc-add-ons] <service>"
    echo ""
    echo "         dev - Launch full remote environment for development with vs code."
    echo "       shell - Launch the docker container."
    echo "       start - Build, install then run uc-aom (default)."
    echo "  tc-add-ons - Package and push the current set of test add-ons to wmucdev."
    echo ""
    echo "Optional arguments:"
    echo "  service - exclusively build and run this service. Default is to start *all* services."
    exit 1
esac

export RUN_MODE="${RUN_MODE}"

if [ -z "${SERVICE}" ]; then
  docker-compose -f ./docker/docker-compose.yml ${ARGS} -p ${NAME} up --build
else
  docker-compose -f ./docker/docker-compose.yml ${ARGS} -p ${NAME} build ${SERVICE} && \
    docker-compose -f ./docker/docker-compose.yml ${ARGS} -p ${NAME} run --rm --service-ports ${SERVICE}
fi
