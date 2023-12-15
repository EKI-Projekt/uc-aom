# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Builder stage
FROM golang:1.17 AS builder

WORKDIR /go/src/uc-aom
COPY ./ .

RUN git checkout -f tags/0.3.2 -b release/0.3.2

WORKDIR /go/src/uc-aom/tools
RUN env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build \
    -tags dev \
    -o uc-aom-packager \
    main.go


# Production stage
FROM alpine:3.15

RUN apk add --no-cache jq imagemagick ghostscript-fonts

COPY --from=builder /go/src/uc-aom/tools/uc-aom-packager /usr/local/bin/uc-aom-packager
COPY scripts/docker/v0_3_2/source-credentials.json /tmp/source-credentials.json
COPY scripts/docker/v0_3_2/target-credentials-template.json /tmp/target-credentials-template.json
COPY scripts/test-add-on-manifest-create.sh /tmp/test-add-on-manifest-create.sh

# Copy current test add-ons into container
# We can't use the test add-on from v0.3.2 because they don't contain vendor informations
# FIXME: add vendor information to all test add-on in v0.3.2
COPY testdata /testdata
