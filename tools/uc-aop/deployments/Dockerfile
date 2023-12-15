# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Builder stage
FROM golang:1.17 AS builder
WORKDIR /go/src/uc-aom
COPY ./ .
ARG BUILD_TIME
ENV BUILD_TIME=${BUILD_TIME}
ARG COPYRIGHT_YEAR
ENV COPYRIGHT_YEAR=${COPYRIGHT_YEAR}
ARG VERSION
ENV VERSION=${VERSION:-v0.0.0}
RUN make -f tools/uc-aop/Makefile build

# Test stage
FROM golang:1.17 AS test
WORKDIR /go/src/uc-aom
COPY ./ .
RUN make -f tools/uc-aop/Makefile unit-test

# Production stage
FROM alpine:3.15

COPY --from=builder /go/build/uc-aom-packager /usr/local/bin/uc-aom-packager

CMD ["uc-aom-packager", "-h"]
