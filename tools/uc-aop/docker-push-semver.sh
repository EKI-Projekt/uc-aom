#!/bin/sh

# Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

# Set values of shell options
set -e # Exit immediately if a command exits with a non-zero status
set -x # Print commands and their arguments as they are executed.

TAG=$(echo "${INPUT_TAG_REF}" | sed -e 's|refs/tags||' | sed -e 's/^v//' | sed -e 's/+.*$//')

DOCKER_IMAGE_TAG_MAJOR=$(echo "$TAG" | cut -d"." -f1)
DOCKER_IMAGE_TAG_MINOR=$(echo "$TAG" | cut -d"." -f2)
DOCKER_IMAGE_TAG_PATCH=$(echo "$TAG" | cut -d"." -f3 | sed -e 's/-.*$//')
DOCKER_IMAGE_TAG_PATCH_WITH_PRE_RELEASE=$(echo "$TAG" | cut -d"." -f3 | sed -e 's/+.*$//')

echo "Tagging ${DOCKER_IMAGE_TAG_MAJOR}, ${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}, ${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH} and ${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH_WITH_PRE_RELEASE} for ${INPUT_TARGET_IMAGE_NAME} ..."

docker pull "${INPUT_SOURCE_IMAGE_NAME}:${TAG}"

docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}"
docker push "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}"
docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}"
docker push "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}"
docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH}"
docker push "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH}"
docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH_WITH_PRE_RELEASE}"
docker push "${INPUT_TARGET_IMAGE_NAME}:${DOCKER_IMAGE_TAG_MAJOR}.${DOCKER_IMAGE_TAG_MINOR}.${DOCKER_IMAGE_TAG_PATCH_WITH_PRE_RELEASE}"

if [ "${INPUT_TAG_LATEST}" = "yes" ] || [ "${INPUT_TAG_LATEST}" = "true" ]; then
  echo 'Creating and publishing "latest" tag ...'
  docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:latest"
  docker push "${INPUT_TARGET_IMAGE_NAME}:latest"
fi

if [ "${INPUT_TAG_CUSTOM}" != "" ]; then
  echo "Creating and publishing custom tag '${INPUT_TAG_CUSTOM}' ..."
  docker tag "${INPUT_SOURCE_IMAGE_NAME}:${TAG}" "${INPUT_TARGET_IMAGE_NAME}:${INPUT_TAG_CUSTOM}"
  docker push "${INPUT_TARGET_IMAGE_NAME}:${INPUT_TAG_CUSTOM}"
fi

echo "::set-output name=image_tag::${TAG}"
echo "::set-output name=image_tag_major::${DOCKER_IMAGE_TAG_MAJOR}"
echo "::set-output name=image_tag_minor::${DOCKER_IMAGE_TAG_MINOR}"
echo "::set-output name=image_tag_patch::${DOCKER_IMAGE_TAG_PATCH}"
echo "::set-output name=image_tag_patch_with_pre_release::${DOCKER_IMAGE_TAG_PATCH_WITH_PRE_RELEASE}"
