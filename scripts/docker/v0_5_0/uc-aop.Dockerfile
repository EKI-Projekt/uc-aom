# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Builder stage
FROM golang:1.17 AS builder

WORKDIR /go/src/uc-aom
COPY ./ .

RUN git checkout -f tags/0.5.0 -b release/0.5.0

WORKDIR /go/src/uc-aom

RUN env VERSION=v0.5.0 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -ldflags "-X 'u-control/uc-aom/internal/aop/company.version=$(VERSION)' \
    -s \
    -w" \
    -installsuffix 'static' \
    -tags dev \
    -o build/uc-aom-packager \
    u-control/uc-aom/tools/uc-aop

# Production stage
FROM alpine:3.15

RUN apk add --no-cache jq imagemagick ghostscript-fonts

COPY --from=builder /go/src/uc-aom/build/uc-aom-packager /usr/local/bin/uc-aom-packager
COPY --from=builder /go/src/uc-aom/testdata /testdata

COPY scripts/docker/v0_5_0/source-credentials.json /tmp/source-credentials.json
COPY scripts/docker/v0_5_0/target-credentials-template.json /tmp/target-credentials-template.json
COPY scripts/test-add-on-manifest-create.sh /tmp/test-add-on-manifest-create.sh
