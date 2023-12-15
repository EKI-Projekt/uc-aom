# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM nginx:alpine-slim
COPY testdata/test-uc-addon-settings/@0.1.0/40-create-index-html-of-env.sh /docker-entrypoint.d/40-create-index-html-of-env.sh
RUN chmod +x /docker-entrypoint.d/40-create-index-html-of-env.sh
