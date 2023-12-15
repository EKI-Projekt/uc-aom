#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

manifestsDir=$PWD/testdata

# Assume the dev environment as default.
buildConfig="${1-dev}"
username="${2}"
passwd="${3}"

dockerHostPortInterface="${4-0.0.0.0}"
dockerHostUrl="${5-host.docker.internal}"

sourceCredentials=$(mktemp /tmp/initialize-source-registry.XXXXXX)
targetCredentials=$(mktemp /tmp/initialize-target-registry.XXXXXX)

for addOnVersionDir in $(find "$manifestsDir" -type f -name "manifest.json" -exec dirname {} \;)
do
  addOnDir=$(dirname "$addOnVersionDir")
  realpath=$(realpath "$addOnDir")
  reponame=${realpath#$manifestsDir/}
  version=$(basename "$addOnVersionDir")
  version="${version#@}"

  addOnTempWorkDir=$(/bin/sh scripts/test-add-on-manifest-create.sh "$addOnVersionDir" "$dockerHostPortInterface" "$dockerHostUrl")

  echo "Pushing AddOn data '$addOnDir' in version '$version' to registry"
  cat << EOF > "$sourceCredentials"
{ "repositoryname": "$reponame-addon-pkg",
 "serveraddress": "registry:5000" }
EOF

  cat << EOF > "$targetCredentials"
{ "repositoryname": "$reponame-addon-pkg",
 "username": "$username",
 "password": "$passwd" }
EOF

  go run -tags $buildConfig tools/uc-aop/main.go push -m "$addOnTempWorkDir" -s $sourceCredentials -t $targetCredentials -v

  rm -rf $addOnTempWorkDir
done

rm "$sourceCredentials"
rm "$targetCredentials"
