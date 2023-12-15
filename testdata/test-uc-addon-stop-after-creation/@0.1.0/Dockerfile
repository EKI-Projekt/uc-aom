# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM alpine
COPY testdata/test-uc-addon-stop-after-creation/@0.1.0/exit-after-first-boot.sh /exit-after-first-boot.sh
RUN chmod +x /exit-after-first-boot.sh
HEALTHCHECK --interval=1s --timeout=1s --retries=1 CMD test -f /initFile
