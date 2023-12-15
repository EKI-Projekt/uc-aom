# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM debian:bullseye-slim AS build

RUN apt-get update -y
RUN apt-get install -y git build-essential libconfig-dev libz-dev libssl-dev libarchive-dev libjson-c-dev libcurl4-openssl-dev

WORKDIR /tmp
RUN git clone --depth 1 --branch 2022.05 https://github.com/sbabic/swupdate.git
WORKDIR /tmp/swupdate

# Run 'make oldconfig' to update configuration
COPY /.config /tmp/swupdate/.config
RUN make oldconfig

# Build swupdate
RUN make swupdate


FROM debian:bullseye-slim AS runtime
COPY --from=build /tmp/swupdate/swupdate /usr/bin/swupdate
COPY --from=build /tmp/swupdate/examples/www/v2 /usr/share/swupdate/www
RUN apt-get update -y
RUN apt-get install libconfig9 libarchive-dev libcurl4 libjson-c5 -y
