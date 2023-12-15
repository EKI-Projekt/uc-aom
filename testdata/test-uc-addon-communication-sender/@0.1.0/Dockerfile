# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM alpine
COPY testdata/test-uc-addon-communication-sender/@0.1.0/healthcheck.sh /healthcheck.sh
HEALTHCHECK --interval=1ms --timeout=2s --retries=1 \
    CMD /healthcheck.sh
