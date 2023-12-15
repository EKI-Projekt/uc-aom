# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

BUILD_TIME := $(or $(BUILD_TIME), $(shell date -u '+%Y-%m-%dT%H:%M:%SZ'))
COPYRIGHT_YEAR := $(or $(COPYRIGHT_YEAR), $(shell date -u '+%Y'))
DOCKER_TAG := $(or $(DOCKER_TAG), "latest")

unit-test: ## Run the unit tests.
	go test -v -tags dev $(shell go list u-control/uc-aom/internal/aop/... | grep -vE '/aop/cmd|/aop/registry')
.PHONY: unit-test

test: unit-test ## Run the integration tests, requires a docker registry.
	go test -v -tags dev u-control/uc-aom/internal/aop/cmd/...
	go test -v -tags dev u-control/uc-aom/internal/aop/registry/...
.PHONY: test

build: unit-test ## Build the distributables.
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build \
	-ldflags "-X 'u-control/uc-aom/internal/aop/company.version=$(VERSION)' \
			  -X 'u-control/uc-aom/internal/aop/company.buildtime=$(BUILD_TIME)' \
			  -X 'u-control/uc-aom/internal/aop/company.copyrightyear=$(COPYRIGHT_YEAR)' \
			  -s \
			  -w" \
	-installsuffix 'static' \
	-tags prod \
	-o ../../build/uc-aom-packager \
	u-control/uc-aom/tools/uc-aop
.PHONY: build

buildx: unit-test ## Package uc-aop as a docker image uc-aop-latest
	docker buildx build                                         \
	    --no-cache                                              \
		--load                                                  \
		--tag uc-aop-latest                                     \
		--build-arg VERSION=v0.0.0-`git rev-parse --short HEAD` \
		--file tools/uc-aop/deployments/Dockerfile .
.PHONY: buildx

buildx_push: ## Push uc-aom-packger docker image to wmucdev registry
	docker buildx build                                                \
		--push                                                         \
		--no-cache                                                     \
		--platform linux/amd64                                         \
		--tag $(REGISTRY)/u-control/uc-aom-packager:$(DOCKER_TAG)      \
		--build-arg VERSION=$(DOCKER_TAG)-`git rev-parse --short HEAD` \
		--file tools/uc-aop/deployments/Dockerfile .
.PHONY: buildx_push

semver_push: ## Push uc-aom-packger docker image to wmucdev registry using semver tagging like uc-aom-packager:1, uc-aom-packager:1.2 and uc-aom-packager:1.2.3
	INPUT_TAG_REF=$(DOCKER_TAG) \
	INPUT_SOURCE_IMAGE_NAME=$(REGISTRY)/u-control/uc-aom-packager \
	INPUT_TARGET_IMAGE_NAME=$(REGISTRY)/u-control/uc-aom-packager \
	tools/uc-aop/docker-push-semver.sh
.PHONY: semver_push

clean: ## Remove all caches and any built distributables.
	@go clean -r -i -modcache -testcache
	@rm -f ../../build/uc-aom-packager
.PHONY: clean

help: ## Display this help message and exit.
	@echo ""
	@echo "     __    __    ______              ___        ______   .______"
	@echo "    |  |  |  |  /      |            /   \      /  __  \  |   _  \\"
	@echo "    |  |  |  | |  ,----' ______    /  ^  \    |  |  |  | |  |_)  |"
	@echo "    |  |  |  | |  |     |______|  /  /_\  \   |  |  |  | |   ___/"
	@echo "    |  \`--'  | |  \`----.         /  _____  \  |  \`--'  | |  |"
	@echo "     \______/   \______|        /__/     \__\  \______/  | _|"
	@echo ""
	@echo ""
	@echo "Select from one of the following targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

.DEFAULT_GOAL := help
