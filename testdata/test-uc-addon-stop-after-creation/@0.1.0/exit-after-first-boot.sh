#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Script stops the docker container on first boot.
# The stopped container is required for testing the auto start of apps on boot up of the uc-aom.

INITFILE=/initFile
if test -f "$INITFILE"; then
    echo "$INITFILE exist. Staying alive."
    tail -f /dev/null
else
    echo "$INITFILE not found. Creating file and exit."
    touch $INITFILE
    exit 1
fi
