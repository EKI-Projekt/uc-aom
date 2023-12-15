#!/bin/sh

# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# directories are mapped to docker volumes (localcatalog, remotecatalog and cache) which can contain an old ac-aom state
rm -rf /var/lib/uc-aom/*
rm -rf /var/run/uc-aom/*
rm -rf /var/cache/uc-aom/*

# create directories to create add-on proxy routes
rm -rf /var/lib/nginx/*

mkdir /var/lib/nginx/user-sites-available
mkdir /var/lib/nginx/user-sites-enabled
mkdir /var/lib/nginx/user-routes-available
mkdir /var/lib/nginx/user-routes-enabled
