#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Before you can use this script you need to define an ssh configuration in your local file ~/.ssh/config
#
# Example for a configuration:
# Host example-config
#     HostName <ip-address>
#     User root
#     StrictHostKeyChecking no
#
# As second argument you have to provide the name of the binary you want to deploy e.g. 'uc-aom' for the CLI or 'uc-aomd' for the daemon
#
# Now, you can run this script:
# ./scripts/deploy2device.sh example-config <BINARY_NAME>

SSH_CONFIG=$1
BINARY_NAME=$2

if [ -z "${BINARY_NAME}" ]
then
    echo "Please provide a binary name that you want to deploy e.g. 'uc-aom' or 'uc-aomd'"
    exit 1
fi

if ! [ -e ./build/"${BINARY_NAME}" ]
then
    echo "No '${BINARY_NAME}' to deploy, please call the build command via the makefile."
    exit 1
fi

scp ./build/"${BINARY_NAME}" "${SSH_CONFIG}":/tmp/

ssh "${SSH_CONFIG}" "systemctl stop uc-aom"
ssh "${SSH_CONFIG}" "mount -o remount,rw /dev/root / "
ssh "${SSH_CONFIG}" "mv /tmp/${BINARY_NAME} /usr/bin/"
ssh "${SSH_CONFIG}" "systemctl start uc-aom"
