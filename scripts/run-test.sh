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
# Project root is one level up from the script directory
PROJECT_ROOT="${SCRIPT_DIR}/.."

# Change to project root directory
cd "${PROJECT_ROOT}"

# Test level: smoke, all
TEST_LEVEL=${1:-"smoke"}

# Base URL from environment or default to localhost:9527 (for local dev)
MATRIXHUB_BASE_URL="${MATRIXHUB_BASE_URL:-http://localhost:9527}"

echo "================================================"
echo "MatrixHub E2E Test Runner"
echo "================================================"
echo "Test Level: ${TEST_LEVEL}"
echo "Base URL:   ${MATRIXHUB_BASE_URL}"
echo "================================================"

# Step 1: Wait for service to be ready
echo ""
echo "Waiting for MatrixHub service to be ready..."
TIMEOUT=60
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $TIMEOUT ]; do
    echo "[$(($ELAPSED))s] Checking service health..."

    if curl -s -o /dev/null "${MATRIXHUB_BASE_URL}" 2>&1; then
        echo ""
        echo "✓ MatrixHub service is ready!"
        break
    fi

    echo "  → Waiting for service to be ready..."
    sleep $INTERVAL
    ELAPSED=$(($ELAPSED + $INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo ""
    echo "ERROR: Timeout waiting for MatrixHub service at ${MATRIXHUB_BASE_URL}"
    echo ""
    echo "Please ensure MatrixHub is running first."
    exit 1
fi

# Step 2: Run E2E tests with ginkgo
echo ""
echo "Running E2E tests..."
echo ""

export MATRIXHUB_BASE_URL="${MATRIXHUB_BASE_URL}"

case ${TEST_LEVEL} in
    "smoke")
        echo "Running smoke tests..."
        ginkgo -v --timeout=10m ./test/e2e_apiserver/...
        ;;
    "all")
        echo "Running all E2E tests..."
        ginkgo -v --timeout=20m ./test/e2e_apiserver/...
        ;;
    *)
        echo "Unknown test level: ${TEST_LEVEL}"
        echo "Supported levels: smoke, all"
        exit 1
        ;;
esac

echo ""
echo "================================================"
echo "E2E Test Complete!"
echo "================================================"
