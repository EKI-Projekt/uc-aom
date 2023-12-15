# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM golang:1.17-alpine as builder
WORKDIR /app
ADD /testdata/test-uc-addon-websocket/0.1.0-1/ /app/
RUN go mod download
RUN go build -ldflags "-s -w" -o server ./...

FROM alpine
WORKDIR /app
COPY --from=builder /app/ ./
CMD ["./server"]
