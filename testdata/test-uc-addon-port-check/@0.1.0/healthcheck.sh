#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

gatewayAddress=$(ip route | awk '/^default/{print $3}') > /dev/null 2>&1

# The PORT variable is a setting of the add-on and can be set by the user
nc -z -w 1 "${gatewayAddress}" "${PORT}"
if [ "$?" -eq 0 ]; then
    exit 0
fi


exit 1
