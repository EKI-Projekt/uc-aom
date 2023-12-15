#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

ACCESS_ALLOWED=0
ACCESS_BLOCKED=1

usage()
{
  cat << EOF
Options:
  is_blocked - checks status of root access, returns
		1 - if access is currently blocked
		0 - if access is currently allowed
EOF
}

status()
{
  # ROOT_ACCESS_CONFIG_FILE variable is defined in the docker compose file
  ROOT_ACCESS_ENABLED=$(jq .rootAccessEnabled "$ROOT_ACCESS_CONFIG_FILE")
  if [ "$ROOT_ACCESS_ENABLED" -eq 1 ];then
    exit "$ACCESS_ALLOWED"
  fi

  if [ "$ROOT_ACCESS_ENABLED" -eq 0 ];then
    exit "$ACCESS_BLOCKED"
  fi

  exit "$ACCESS_BLOCKED"
}

case $1 in
  "is_blocked")
    echo "checking if root access is currently blocked"
    status
    ;;
  *)
    usage
    exit 2
    ;;
esac
