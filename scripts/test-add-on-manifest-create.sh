#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

add_on_directory="${1}"

# In the docker dev environment we are using the default public host interface.
# This is needed because otherwise the nginx can not forward to the add-on webserver.
docker_host_port_interface="${2-0.0.0.0}"

# The special DNS name host.docker.internal resolves to the internal IP address of the host.
# We are using this url to forward the requests from ngnix proxy to the add-on webserver.
# See also: https://docs.docker.com/desktop/windows/networking/
# Linux settings: https://github.com/docker/for-linux/issues/264#issuecomment-965465879
docker_host_url="${3-host.docker.internal}"

manifest_create_temp_directory=$(mktemp -d)
printf >&2 "cp -r %s/* %s\n" "$add_on_directory" "$manifest_create_temp_directory"
cp -r "${add_on_directory}"/* "$manifest_create_temp_directory"

# Replace manifest placeholders with docker environment settings
sed -e "s,@HOST_PORT_INTERFACE@,${docker_host_port_interface},g" \
    -e "s,@HOST_URL@,${docker_host_url},g" \
    -i "$manifest_create_temp_directory"/manifest.json

title=$(jq -r '.title' "$manifest_create_temp_directory"/manifest.json)
logo=$(jq -r '.logo' "$manifest_create_temp_directory"/manifest.json)
version=$(jq -r '.version' "$manifest_create_temp_directory"/manifest.json)

printf >&2 "Generating logo: '%s/%s'\n" "$manifest_create_temp_directory" "$logo"
convert -size 128x128 -background none -layers merge \
        \( -trim +repage -gravity Center -fill white -pointsize 12 -weight Heavy \
        label:"$title\n$version" \
        \( +clone -extent 128x128 -background black -shadow 80x3+3+3 \) \
        +swap -background none -layers merge \) \
        -insert 0 -gravity Center -append -background \#eb8c00 \
        -extent 128x128 "$manifest_create_temp_directory/$logo"

# Return the temp directory with the create test add-on.
printf "%s" "${manifest_create_temp_directory}"
return 0
