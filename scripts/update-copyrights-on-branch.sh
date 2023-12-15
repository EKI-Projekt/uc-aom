#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Updates the copyright hint for all files that has been changed between the HEAD and the master commit

weidmuellerCopyright="Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>"

files=$(git diff --diff-filter=d --name-only --text origin/master..HEAD)
reuse annotate --copyright="$weidmuellerCopyright" --license=MIT --copyright-style=string --merge-copyright --skip-unrecognised $files


unrecognisedFilesWithPythonStyle=$(git diff --diff-filter=d --name-only --text origin/master..HEAD -- '*.Dockerfile' '*.env')

if [ -n "$unrecognisedFilesWithPythonStyle" ]; then
    reuse annotate --copyright="$weidmuellerCopyright" --license=MIT --copyright-style=string --style=python --merge-copyright $unrecognisedFilesWithPythonStyle
fi
