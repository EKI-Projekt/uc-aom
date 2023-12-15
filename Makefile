# Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

GRPC_SERVER_PASS := $(or $(GRPC_SERVER_PASS), 127.0.0.1:49288)

# list of the tools can be found here https://github.com/golang/vscode-go/blob/master/docs/tools.md#tools
vs-code-dev-deps: ## Install dev dependencies for developing in VS Code with the go extension.
	go install -v golang.org/x/tools/gopls@v0.11.0
	go install -v github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/go-delve/delve/cmd/dlv@master
	go install github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest
	go install -v github.com/ramya-rao-a/go-outline@latest
	go install honnef.co/go/tools/cmd/staticcheck@v0.3.2
	go install github.com/cweill/gotests/gotests@latest
	go install github.com/fatih/gomodifytags@latest
	go install github.com/josharian/impl@latest
	go install github.com/haya14busa/goplay/cmd/goplay@latest
.PHONY: vs-code-dev-deps

# Setup environment variables for the generation process
# The GOHOSTPATH, GOHOSTOS and GOHOSTARCH variables are set by the bitbake recipe.
# These host environment variables are used by the compile task to distinguish between host and target device
# The binaries are installed based on the host system and only needed for generating go files.
# The generated go files are then build with the target environment variables.
generate: GOPATH := $(or ${GOHOSTPATH},$(GOPATH))
generate: PATH := $(if ${GOHOSTPATH},${GOHOSTPATH}/bin:${PATH},${PATH})
generate: GOOS := $(or ${GOHOSTOS},${GOOS})
generate: GOARCH := $(or ${GOHOSTARCH},${GOARCH})
generate: ## Setup environment variables and generate code.
	go generate u-control/uc-aom/internal/aom/grpc
.PHONY: generate

dev: vs-code-dev-deps generate ## Default command. Depends on vs-code-dev-deps and generate.
	echo dev
.PHONY: dev

build: generate ## Does nothing but depends on generate.
.PHONY: build

format: ## Run go fmt -x on all source files.
	go fmt -x u-control/uc-aom/...
.PHONY: format

unit-test: ## Run the unit tests.
	go test -v -tags dev $(shell go list ./... | grep -vE '/test|/tools/|/aop/cmd|/aop/registry')
.PHONY: unit-test

integration-test: ## Run the integration tests, requires a docker registry.
	go test -v -tags dev -timeout 5m u-control/uc-aom/test
.PHONY: integration-test

migration-test: ## Run the migration test
	go test -v -tags dev -timeout 20m u-control/uc-aom/test/migration
.PHONY: migration-test

test: unit-test integration-test ## Run the unit then the intergration tests
.PHONY: test

build-device: generate ## Produce distributable for the arm-based U-Control like UC2000-AC or IoT-GW30.
	env GOARCH=arm GOARM=7 go build \
		-ldflags "-s -w" \
		-installsuffix 'static' \
		-tags prod \
		-o build/uc-aomd u-control/uc-aom/cmd/uc-aomd
.PHONY: build-device

build-device-arm64: generate ## Produce distributable for the arm64-based U-Control like UC3000 or UC4000.
	env GOARCH=arm64 GOOS=linux go build \
		-ldflags "-s -w" \
		-installsuffix 'static' \
		-tags prod \
		-o build/uc-aomd u-control/uc-aom/cmd/uc-aomd
.PHONY: build-device64

build-doc: ## Generate documentation from the manifest JSON schema.
	pip3 install json-schema-for-humans -q
	generate-schema-doc --config-file configs/json-schema-doc-config.json api/uc-manifest.schema.json api/uc-manifest.schema-doc.md
.PHONY: build-doc

run-docker: ## Start the add-on manager in the docker based dev env.
	go run -tags dev u-control/uc-aom/cmd/uc-aomd -vvv
.PHONY: run-docker

build-cli: ## Build the app manager CLI
	go build -tags dev -o build/uc-aom u-control/uc-aom/cmd/uc-aom
.PHONY: build-cli

build-device-cli: generate ## Build the app manager CLI for the arm-based U-Control
	env GOARCH=arm GOARM=7 go build \
		-ldflags "-X 'u-control/uc-aom/internal/cli/config.grpcAddress=${GRPC_SERVER_PASS}' \
		-s \
		-w" \
		-installsuffix 'static' \
		-tags prod \
		-o build/uc-aom u-control/uc-aom/cmd/uc-aom
.PHONY: build-device-cli

build-device-cli-arm64: generate ## Build the app manager CLI for the arm64-based U-Control
	env GOARCH=arm64 GOOS=linux go build \
		-ldflags "-X 'u-control/uc-aom/internal/cli/config.grpcAddress=${GRPC_SERVER_PASS}' \
		-s \
		-w" \
		-installsuffix 'static' \
		-tags prod \
		-o build/uc-aom u-control/uc-aom/cmd/uc-aom
.PHONY: build-device-cli-arm64

build-example: ## Build the example
	cd examples/cli-swu && bash build-swu.sh
.PHONY: build-example

install: REGISTRYFILE := $(or ${REGISTRYFILE}, registrycredentials_prod.json)
install: ## Install production registry credentials file.
	install -d ${DESTDIR}/usr/share/uc-aom
	install -d ${DESTDIR}/var/lib/uc-aom
	install -m 0644 credentials/${REGISTRYFILE} ${DESTDIR}/usr/share/uc-aom/registrycredentials.json
.PHONY: install

clean: ## Remove all caches and any built distributables.
	@go clean -r -i -modcache -testcache
	@rm -f build/uc-aom
.PHONY: clean

reuse-lint: ## Run reuse lint to check Copyright an License information.
	reuse lint
.PHONY: reuse-lint

update-copyright: ## Updates the copyright hint for all files that has been changed between the HEAD and the master commit.
	./scripts/update-copyrights-on-branch.sh
.PHONY: update-copyright

help: ## Display this help message and exit.
	@echo ""
	@echo "     __    __    ______              ___        ______   .___  ___."
	@echo "    |  |  |  |  /      |            /   \      /  __  \  |   \/   |"
	@echo "    |  |  |  | |  ,----' ______    /  ^  \    |  |  |  | |  \  /  |"
	@echo "    |  |  |  | |  |     |______|  /  /_\  \   |  |  |  | |  |\/|  |"
	@echo "    |  \`--'  | |  \`----.         /  _____  \  |  \`--'  | |  |  |  |"
	@echo "     \______/   \______|        /__/     \__\  \______/  |__|  |__|"
	@echo ""
	@echo ""
	@echo "Select from one of the following targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

.DEFAULT_GOAL := help
