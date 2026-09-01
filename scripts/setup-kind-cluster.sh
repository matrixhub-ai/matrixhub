#!/bin/bash

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

set -e

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Project root is two levels up from the script directory
PROJECT_ROOT="${SCRIPT_DIR}/.."

# Change to project root directory
cd "${PROJECT_ROOT}"
echo "Working directory: $(pwd)"

# Environment variables with defaults
E2E_CLUSTER_NAME=${E2E_CLUSTER_NAME:-"matrixhub-e2e"}
E2E_KIND_IMAGE_TAG=${E2E_KIND_IMAGE_TAG:-"v1.32.3"}

echo "================================================"
echo "MatrixHub Kind Cluster Setup"
echo "================================================"
echo "Cluster Name: ${E2E_CLUSTER_NAME}"
echo "K8s Version:   ${E2E_KIND_IMAGE_TAG}"
echo "================================================"

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "Error: kind is not installed"
    exit 1
fi

# Check kind version (require >= 0.27)
KIND_VERSION=$(kind version 2>/dev/null | grep -oE 'kind v[0-9]+\.[0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
MIN_KIND_VERSION="0.27.0"

if [ -z "$KIND_VERSION" ]; then
    echo "Error: Unable to determine kind version"
    exit 1
fi

# Compare versions using sort
if [ "$(printf '%s\n' "$MIN_KIND_VERSION" "$KIND_VERSION" | sort -V | head -n1)" != "$MIN_KIND_VERSION" ]; then
    echo "Error: kind version $KIND_VERSION is too old (require >= $MIN_KIND_VERSION)"
    exit 1
fi

# Check if cluster already exists
if kind get clusters | grep -q "^${E2E_CLUSTER_NAME}$"; then
    echo "Cluster '${E2E_CLUSTER_NAME}' already exists, deleting it..."
    kind delete cluster --name="${E2E_CLUSTER_NAME}"
fi

# Create KIND cluster
echo ""
echo "Creating KIND cluster..."
cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30001
    hostPort: 30001
    listenAddress: "127.0.0.1"
    protocol: TCP
  - containerPort: 30022
    hostPort: 30022
    listenAddress: "127.0.0.1"
    protocol: TCP
EOF
kind create cluster --name="${E2E_CLUSTER_NAME}" --image=kindest/node:${E2E_KIND_IMAGE_TAG} --config=/tmp/kind-config.yaml --wait=120s

echo "KIND cluster created successfully"

# Verify cluster is ready
echo ""
echo "Verifying cluster is ready..."
kubectl wait --for=condition=ready nodes --all --timeout=300s
echo "All nodes are ready"

echo ""
echo "================================================"
echo "KIND Cluster Setup Complete!"
echo "================================================"
