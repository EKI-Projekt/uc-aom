# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM nginx:1.20.1-alpine

COPY scripts/nginx /etc/nginx
