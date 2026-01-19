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

GO_CMD ?= go

GIT_TAG ?= $(shell git describe --tags --dirty --always)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)

PLATFORMS ?= linux/amd64,linux/arm64
IMAGE_BUILD_CMD ?= docker buildx build

IMAGE_REPO := ghcr.io/matrixhub-ai/matrixhub

BASE_IMAGE_PREFIX ?= docker.io
NPM_CONFIG_REGISTRY ?= https://registry.npmjs.org
GOPROXY ?= https://proxy.golang.org,direct
HTTP_PROXY ?=
HTTPS_PROXY ?=

version_pkg = github.com/matrixhub-ai/matrixhub/pkg/version
LD_FLAGS += -X '$(version_pkg).GitVersion=$(GIT_TAG)'
LD_FLAGS += -X '$(version_pkg).GitCommit=$(GIT_COMMIT)'
LD_FLAGS += -X '$(version_pkg).BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'

.PHONY: all
all: help

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

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with --fix option
	$(GO_CMD) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0 run --fix

web/dist: web
	make -C web dist

.PHONY: serve-web
serve-web: ## Run web frontend locally
	make -C web serve

.PHONY: serve-api
serve-api: ## Serve the API only
	go run ./cmd/matrixhub

.PHONY: run
run: ## Run MatrixHub locally (web + API)
	make -j2 serve-web serve-api

.PHONY: image-build
image-build: ## Build the MatrixHub image
	$(IMAGE_BUILD_CMD) \
		-t $(IMAGE_REPO):$(GIT_TAG) \
		-t $(IMAGE_REPO):$(GIT_BRANCH) \
		--platform=$(PLATFORMS) \
		--build-arg BASE_IMAGE_PREFIX=$(BASE_IMAGE_PREFIX) \
		--build-arg NPM_CONFIG_REGISTRY=$(NPM_CONFIG_REGISTRY) \
		--build-arg GOPROXY=$(GOPROXY) \
		--build-arg HTTP_PROXY="$(HTTP_PROXY)" \
		--build-arg HTTPS_PROXY="$(HTTPS_PROXY)" \
		--build-arg LD_FLAGS="$(LD_FLAGS)" \
		$(PUSH) \
		$(IMAGE_BUILD_EXTRA_OPTS) \
		.

.PHONY: image-push
image-push: PUSH=--push
image-push: image-build

website/build: website
	make -C website build

.PHONY: serve-website
serve-website: ## Run documentation website locally
	make -C website serve

.PHONY: clean
clean: ## Clean all build artifacts
	rm -rf bin
	make -C web clean
	make -C website clean
