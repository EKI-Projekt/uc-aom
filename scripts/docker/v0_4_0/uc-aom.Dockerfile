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

RUN git checkout -f tags/0.4.0 -b release/0.4.0
RUN make generate && \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -tags dev \
    -o build/uc-aom u-control/uc-aom/cmd/uc-aom

# Production stage
FROM alpine:3.15

COPY --from=builder /go/src/uc-aom/build/uc-aom /usr/local/bin/uc-aom
COPY scripts/docker/v0_4_0/registrycredentials.json /usr/share/uc-aom/registrycredentials.json

# add docker because of nginx proxy reload
RUN apk update && apk add --no-cache docker-cli

CMD ["uc-aom", "-vvv"]
