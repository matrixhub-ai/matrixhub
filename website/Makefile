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

node_modules: package.json
	npm install

# Start the dev server with HMR (hot reload).
.PHONY: start
start: node_modules
	npm run start

# Build the static site (production).
build: node_modules src
	npm run build

# Serve the pre-built static site (production).
.PHONY: serve
serve: build
	npm run serve

# Clean build artifacts
.PHONY: clean
clean:
	npm run clear
