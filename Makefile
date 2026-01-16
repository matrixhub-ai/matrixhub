# Copyright 2022 The Kubernetes Authors.
# Copyright 2026 The MatrixHub Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif

GO_CMD ?= go
GO_FMT ?= gofmt
# Use go.mod go version as a single source of truth of GO version.
GO_VERSION := $(shell awk '/^go /{print $$2}' go.mod|head -n1)

GIT_TAG ?= $(shell git describe --tags --dirty --always)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
# Image URL to use all building/pushing image targets
HOST_IMAGE_PLATFORM ?= linux/$(shell go env GOARCH)
PLATFORMS ?= linux/amd64,linux/arm64
DOCKER_BUILDX_CMD ?= docker buildx
IMAGE_BUILD_CMD ?= $(DOCKER_BUILDX_CMD) build

STAGING_IMAGE_REGISTRY := ghcr.io/matrixhub-ai
IMAGE_REGISTRY ?= $(STAGING_IMAGE_REGISTRY)

IMAGE_REPO := $(IMAGE_REGISTRY)/matrixhub

IMAGE_TAG := $(IMAGE_REPO):$(GIT_TAG)

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_DIR ?= $(PROJECT_DIR)/bin
ARTIFACTS ?= $(BIN_DIR)
TOOLS_DIR := $(PROJECT_DIR)/hack/internal/tools

# Use mirror registry if needed: export BASE_IMAGE_PREFIX=m.daocloud.io/docker.io
BASE_IMAGE_PREFIX ?= docker.io
ALPINE_VERSION ?=3.23
BASE_IMAGE ?= ${BASE_IMAGE_PREFIX}/library/alpine:${ALPINE_VERSION}
HTTP_PROXY ?=
HTTPS_PROXY ?=
CGO_ENABLED ?= 0

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Setting SED allows macos users to install GNU sed and use the latter
# instead of the default BSD sed.
ifeq ($(shell command -v gsed 2>/dev/null),)
    SED ?= $(shell command -v sed)
else
    SED ?= $(shell command -v gsed)
endif
ifeq ($(shell ${SED} --version 2>&1 | grep -q GNU; echo $$?),1)
    $(error !!! GNU sed is required. If on OS X, use 'brew install gnu-sed'.)
endif

version_pkg = github.com/matrixhub-ai/matrixhub/pkg/version
LD_FLAGS += -X '$(version_pkg).GitVersion=$(GIT_TAG)'
LD_FLAGS += -X '$(version_pkg).GitCommit=$(GIT_COMMIT)'
LD_FLAGS += -X '$(version_pkg).BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'

# Update these variables when preparing a new release or a release branch.
# Then run `make prepare-release-branch`
RELEASE_VERSION=v0.0.1
RELEASE_BRANCH=main
# Application version for Helm and npm (strips leading 'v' from RELEASE_VERSION)
APP_VERSION := $(shell echo $(RELEASE_VERSION) | cut -c2-)

.PHONY: all
all: generate fmt vet build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

include Makefile-deps.mk

##@ Development



.PHONY: generate
generate: gomod-download ## Run all code generation tasks

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO_CMD) fmt ./...

.PHONY: fmt-verify
fmt-verify:
	@out=`$(GO_FMT) -w -l -d $$(find . -name '*.go' | grep -v /vendor/)`; \
	if [ -n "$$out" ]; then \
	    echo "$$out"; \
	    exit 1; \
	fi

.PHONY: gomod-verify
gomod-verify:
	$(GO_CMD) mod tidy
	git --no-pager diff --exit-code go.mod go.sum

.PHONY: gomod-download
gomod-download:
	$(GO_CMD) mod download


.PHONY: vet
vet: ## Run go vet against code.
	$(GO_CMD) vet ./...

.PHONY: lint-fix
lint-fix:
	$(GO_CMD) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run --fix

##@ Build

.PHONY: build-web
build-web: ## Build web UI
	cd web && npm install && npm run build

.PHONY: build
build: build-web ## Build matrixhub binary with embedded web UI
	$(GO_BUILD_ENV) $(GO_CMD) build -ldflags="$(LD_FLAGS)" -tags embedweb -o bin/matrixhub cmd/matrixhub/main.go

.PHONY: run
run: generate fmt vet ## Run matrixhub server locally
	$(GO_CMD) run cmd/matrixhub/main.go

# Build the multiplatform container image locally.
.PHONY: image-local-build
image-local-build:
	BUILDER=$(shell $(DOCKER_BUILDX_CMD) create --use)
	$(MAKE) image-build PUSH="$(PUSH)" IMAGE_BUILD_EXTRA_OPTS="$(IMAGE_BUILD_EXTRA_OPTS)"
	$(DOCKER_BUILDX_CMD) rm $$BUILDER

# Build the multiplatform container image locally and push to repo.
.PHONY: image-local-push
image-local-push: PUSH=--push
image-local-push: image-local-build

.PHONY: image-build
image-build:
	$(IMAGE_BUILD_CMD) \
		-t $(IMAGE_TAG) \
		-t $(IMAGE_REPO):$(RELEASE_BRANCH) \
		--platform=$(PLATFORMS) \
		--build-arg BASE_IMAGE_PREFIX=$(BASE_IMAGE_PREFIX) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
		--build-arg CGO_ENABLED=$(CGO_ENABLED) \
		--build-arg GIT_TAG=$(GIT_TAG) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		$(PUSH) \
		$(IMAGE_BUILD_EXTRA_OPTS) \
		./

.PHONY: image-push
image-push: PUSH=--push
image-push: image-build



# Build an image just for the host architecture that can be used for Kind E2E tests.
.PHONY: kind-image-build
kind-image-build: PLATFORMS=$(HOST_IMAGE_PLATFORM)
kind-image-build: PUSH=--load
kind-image-build: kind image-build

##@ Deployment

.PHONY: compose-deploy
compose-deploy: kind-image-build ## Deploy using docker-compose
	docker compose -f deploy/docker-compose.yml up -d

.PHONY: compose-undeploy
compose-undeploy: ## Stop docker-compose deployment
	docker compose -f deploy/docker-compose.yml down

.PHONY: website-server
website-server: ## Run documentation website locally
	cd website && npm install && npm run build && npm run serve

.PHONY: website-perview
website-perview: ## Run website in dev/watch mode (faster - incremental builds)
	cd website && npm install && npm run start


##@ Release

.PHONY: clean-artifacts
clean-artifacts: ## Clean build artifacts
	rm -rf artifacts bin/matrixhub



