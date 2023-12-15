#!/bin/sh

# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

manifestsDir=$PWD/testdata
REGISTRY="${1-localhost:5000}"

echo "PWD: '$PWD'"
echo "docker buildx create --use --driver-opt network=host"
buildxBilder=$(docker buildx create --use --driver-opt network=host)

# build a multi architectur image
buildMultiArchImage() {
  local addOnVersionDir=$1
  local serviceKey=$2
  local image
  image=$(jq -r ".services[\"$serviceKey\"].config.image" "$addOnVersionDir"/manifest.json)
  if [ -e "$addOnVersionDir"/"$serviceKey".Dockerfile ]
    then
        dockerfile=$serviceKey".Dockerfile"
    else
        dockerfile="Dockerfile"
  fi
  echo "docker buildx build . -f $addOnVersionDir/$dockerfile --platform linux/amd64,linux/arm,linux/arm64 --sbom=false --provenance=false -t $REGISTRY/$image --push"
  docker buildx build . -f "$addOnVersionDir/$dockerfile" --platform linux/amd64,linux/arm,linux/arm64 --sbom=false --provenance=false -t "$REGISTRY/$image" --push
}

# get all services from the addon and build the image
buildMultiArchImagesFromManifest() {
  for addOnVersionDir in $(find "$manifestsDir" -type f -name "manifest.json" -exec dirname {} \;)
  do
    manifestPath=$addOnVersionDir/manifest.json
    for serviceKey in $(jq -r '.services | keys | .[]' "$manifestPath")
        do
        buildMultiArchImage "$(dirname "$manifestPath")" "$serviceKey"
        done
  done
}

# build a single arch image
buildSingleArchImage() {
  for addOnVersionDir in $(find "$manifestsDir" -type f -name "Dockerfile.amd64" -exec dirname {} \;)
  do
    for image in $(jq -r '.services | keys as $i | .[].config.image' "$addOnVersionDir"/manifest.json)
        do
        echo "docker buildx build . -f $addOnVersionDir/Dockerfile.amd64 --platform linux/amd64 --sbom=false --provenance=false -t $REGISTRY/$image --push"
        docker buildx build . -f "$addOnVersionDir/Dockerfile.amd64" --platform linux/amd64 --sbom=false --provenance=false -t "$REGISTRY/$image" --push
        done
  done
}

buildMultiArchImagesFromManifest
buildSingleArchImage

echo "docker buildx stop/rm $buildxBilder"
docker buildx stop "$buildxBilder"
docker buildx rm "$buildxBilder"
