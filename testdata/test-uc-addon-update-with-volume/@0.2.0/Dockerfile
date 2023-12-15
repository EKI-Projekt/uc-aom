# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

FROM alpine

HEALTHCHECK --interval=1ms --timeout=1s --retries=1 \
    CMD test -f /stayvolume/testFile.txt || exit 1
