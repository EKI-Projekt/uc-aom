# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM bash:alpine3.16

COPY testdata/test-uc-addon-port-check/@0.1.0/healthcheck.sh /healthcheck.sh
HEALTHCHECK --interval=1ms --timeout=2s --retries=1 \
    CMD bash /healthcheck.sh; if [ $? -eq 0 ]; then exit 0; else exit 1; fi
