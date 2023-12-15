# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM alpine:3.17
COPY testdata/test-uc-addon-multi-service/@0.1.0/healthcheck_a.sh /healthcheck.sh
HEALTHCHECK --interval=1ms --timeout=2s --retries=1 \
    CMD /healthcheck.sh