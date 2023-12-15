# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Builder stage
FROM golang:1.17 AS builder

RUN apt update -y && \
    apt upgrade -y && \
    apt autoremove -y && \
    apt install protobuf-compiler -y && \
    apt clean

WORKDIR /go/src/uc-aom
COPY ./ .

RUN git checkout -f tags/0.5.2 -b release/0.5.2
RUN make generate && \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -tags dev \
    -o build/uc-aomd u-control/uc-aom/cmd/uc-aomd

# Production stage
FROM alpine:3.15

COPY --from=builder /go/src/uc-aom/build/uc-aomd /usr/local/bin/uc-aomd
COPY scripts/docker/v0_5_2/registrycredentials.json /usr/share/uc-aom/registrycredentials.json
COPY scripts/docker/v0_5_2/portainer.env /var/lib/portainer-ce/portainer.env

# add docker because of nginx proxy reload
RUN apk update && apk add --no-cache docker-cli

CMD ["uc-aomd", "-vvv"]
