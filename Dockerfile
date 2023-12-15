# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM golang:1.17-bullseye

RUN apt update -y && \
    apt upgrade -y && \
    apt autoremove -y && \
    apt install apt-transport-https ca-certificates curl gnupg \
    imagemagick \
    git iproute2 procps jq lsb-release vim \
    protobuf-compiler \
    python3-pip \
    cpio -y && \
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

RUN pip3 install reuse
