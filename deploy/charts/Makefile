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

CHART_NAME := matrixhub
CHART_DIR := ./$(CHART_NAME)
VERSION_REGEX := '[vV]*[0-9]\+\.[0-9]\+\.[0-9]\+.*'
CHART_FILE := "$(CHART_DIR)/Chart.yaml"
VALUES_FILE := "$(CHART_DIR)/values.yaml"

# VERSION can be overridden, defaults to git tag
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo "v0.0.1")

.PHONY: all update-versions lint package clean

all: update-versions lint package

## Update version in Chart.yaml and values.yaml
update-versions:
	@echo "Updating Chart version to $(VERSION)"
	@chart_version=$$(echo $(VERSION) | tr -d 'v') ; \
	sed -i 's/version: "*$(VERSION_REGEX)"*/version: '$$chart_version'/g' $(CHART_FILE); \
	sed -i 's/appVersion: "*$(VERSION_REGEX)"*/appVersion: "'$$chart_version'"/g' $(CHART_FILE); \
	sed -i 's/tag: ""/tag: $(VERSION)/g' $(VALUES_FILE)

## Lint the chart
lint:
	@helm lint --with-subcharts --values $(CHART_DIR)/values.yaml $(CHART_DIR)

## Package the chart
package: lint
	@helm package $(CHART_DIR) -d .

## Clean generated files
clean:
	@rm -f *.tgz
