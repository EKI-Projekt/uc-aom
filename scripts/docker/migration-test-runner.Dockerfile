# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM golang:1.17-bullseye

RUN apt update -y && \
    apt upgrade -y && \
    apt autoremove -y && \
    apt install apt-transport-https ca-certificates curl gnupg lsb-release jq -y && \
    apt clean

RUN curl -fsSL https://download.docker.com/linux/debian/gpg \
    | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

RUN echo "deb [arch=amd64" \
    "signed-by=/usr/share/keyrings/docker-archive-keyring.gpg]" \
    "https://download.docker.com/linux/debian $(lsb_release -cs) stable" \
    | tee /etc/apt/sources.list.d/docker.list > /dev/null

RUN apt update && \
    apt install docker-ce-cli -y && \
    apt clean

# Create directories for nginx proxy routes
RUN mkdir -p /var/lib/nginx/user-routes-available \
    /var/lib/nginx/user-routes-enabled \
    /var/lib/nginx/user-sites-available \
    /var/lib/nginx/user-sites-enabled
