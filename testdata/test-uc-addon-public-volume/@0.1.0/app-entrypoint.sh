#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

echo Running child entrypoint initialization steps here
mkdir -p /data/testDir/
touch /data/testDir/testFile.txt

exec /docker-entrypoint.sh "$@"
