# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM nginxinc/nginx-unprivileged:1.23-alpine-slim
COPY --chown=1000:1000 testdata/test-uc-addon-weidmueller-open-source/@0.1.0/default.conf /etc/nginx/conf.d/default.conf
COPY testdata/test-uc-addon-weidmueller-open-source/@0.1.0/LICENSE /licenses/
CMD ["nginx", "-g", "daemon off;"]
