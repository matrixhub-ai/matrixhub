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
E2E_MATRIXHUB_IMAGE=${E2E_MATRIXHUB_IMAGE:-"ghcr.io/matrixhub-ai/matrixhub:latest"}

echo "================================================"
echo "MatrixHub Deployment"
echo "================================================"
echo "Cluster Name:     ${E2E_CLUSTER_NAME}"
echo "MatrixHub Image:  ${E2E_MATRIXHUB_IMAGE}"
echo "================================================"

# Get the image registry and tag from the E2E_MATRIXHUB_IMAGE
# Format: ghcr.io/matrixhub-ai/matrixhub-ci:tag
IMAGE_REGISTRY=""
IMAGE_REPOSITORY=""
IMAGE_TAG=""

# Check if image contains a registry (has at least 2 slashes)
SLASH_COUNT=$(echo "${E2E_MATRIXHUB_IMAGE}" | tr -cd '/' | wc -c)
if [ "${SLASH_COUNT}" -ge 2 ]; then
    # Full format: registry/org/repo:tag
    IMAGE_REGISTRY=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d'/' -f1)
    IMAGE_REPOSITORY=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d'/' -f2- | cut -d':' -f1)
    IMAGE_TAG=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d':' -f2)
else
    # Short format: org/repo:tag or repo:tag
    PARTS=$(echo "${E2E_MATRIXHUB_IMAGE}" | tr '/' '\n' | wc -l)
    if [ "${PARTS}" -eq 2 ]; then
        # org/repo:tag
        IMAGE_REPOSITORY=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d':' -f1)
        IMAGE_TAG=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d':' -f2)
    else
        # repo:tag
        IMAGE_REPOSITORY=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d':' -f1)
        IMAGE_TAG=$(echo "${E2E_MATRIXHUB_IMAGE}" | cut -d':' -f2)
    fi
fi

echo ""
echo "Parsed image configuration:"
echo "  Full Image:  ${E2E_MATRIXHUB_IMAGE}"
echo "  Registry:    ${IMAGE_REGISTRY:-<default>}"
echo "  Repository:  ${IMAGE_REPOSITORY}"
echo "  Tag:         ${IMAGE_TAG}"

# Load images into KIND
echo ""
echo "Loading images into KIND..."

# Define images to load
# Note: MatrixHub image should already be loaded from artifact in CI
# MySQL image is public and can be pulled if not present
IMAGES_TO_LOAD=(
    "${E2E_MATRIXHUB_IMAGE}"
    "mysql:8.4"
    "busybox:latest"
)

for IMAGE in "${IMAGES_TO_LOAD[@]}"; do
    echo ""
    echo "Processing: ${IMAGE}"

    # Check if image exists locally first
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${IMAGE}$"; then
        echo "  ✓ Image exists locally, skipping pull"
    else
        echo "  → Image not found locally, pulling..."
        if docker pull "${IMAGE}"; then
            echo "  ✓ Pulled successfully"
        else
            echo "  ✗ Failed to pull ${IMAGE}"
            docker images | grep -i matrixhub || echo "  No matrixhub images found"
            exit 1
        fi
    fi

    # Load into KIND
    echo "  → Loading into KIND..."
    if kind load docker-image "${IMAGE}" --name="${E2E_CLUSTER_NAME}"; then
        echo "  ✓ Loaded into KIND"
    else
        echo "  ✗ Failed to load into KIND"
        exit 1
    fi
done

echo ""
echo "✓ All images loaded successfully!"

# Install/upgrade MatrixHub
echo ""
echo "Deploying MatrixHub via Helm..."

# Deploy with E2E-specific overrides
helm upgrade --install matrixhub ./deploy/charts/matrixhub \
    --namespace matrixhub \
    --create-namespace \
    --set apiserver.image.registry="${IMAGE_REGISTRY}" \
    --set apiserver.image.repository="${IMAGE_REPOSITORY}" \
    --set apiserver.image.tag="${IMAGE_TAG}" \
    --set mysql.registry="docker.io" \
    --set mysql.repository="library/mysql" \
    --set mysql.persistence.size="5Gi" \
    --set apiserver.service.type="NodePort" \
    --set apiserver.service.nodePort=30001 \
    --set global.storage.apiserver.builtIn=true

echo "✓ Helm command completed"

# Check deployment status
echo ""
echo "Checking deployment status..."
kubectl get pods -n matrixhub
kubectl get svc -n matrixhub

# Wait for pods to be ready with timeout
echo ""
echo "Waiting for pods to be ready (timeout: 300s)..."

# Wait in a loop with progress updates
TIMEOUT=300
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $TIMEOUT ]; do
    echo ""
    echo "[$(($ELAPSED))s] Checking pod status..."
    kubectl get pods -n matrixhub

    # Check MySQL first
    MYSQL_READY=$(kubectl get pods -n matrixhub -l app=matrixhub-mysql -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "false")

    # Check MatrixHub
    MATRIXHUB_READY=$(kubectl get pods -n matrixhub -l app=matrixhub -o jsonpath='{.items[0].status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "false")

    if [ "$MYSQL_READY" = "True" ] && [ "$MATRIXHUB_READY" = "True" ]; then
        echo ""
        echo "✓ All pods are ready!"
        break
    fi

    if [ "$MYSQL_READY" != "True" ]; then
        echo "  → Waiting for MySQL..."
    fi

    if [ "$MATRIXHUB_READY" != "True" ]; then
        echo "  → Waiting for MatrixHub..."
    fi

    sleep $INTERVAL
    ELAPSED=$(($ELAPSED + $INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo ""
    echo "ERROR: Timeout waiting for pods to become ready"
    echo ""
    echo "Final pod status:"
    kubectl get pods -n matrixhub -o wide
    echo ""
    echo "MatrixHub pod logs (init containers):"
    kubectl logs -n matrixhub -l app=matrixhub -c check-db-ready --tail=30 || true
    echo ""
    echo "MatrixHub pod logs (main container):"
    kubectl logs -n matrixhub -l app=matrixhub -c matrixhub --tail=50 || true
    echo ""
    echo "MySQL pod logs:"
    kubectl logs -n matrixhub -l app=matrixhub-mysql --tail=50 || true
    echo ""
    echo "MySQL secret:"
    kubectl get secret -n matrixhub -l app.kubernetes.io/name=mysql -o yaml || true
    echo ""
    echo "Testing MySQL connection manually:"
    kubectl exec -n matrixhub -l app=matrixhub-mysql -- mysqladmin ping -uroot -h localhost || true
    echo ""
    echo "Pod events:"
    kubectl get events -n matrixhub --sort-by='.lastTimestamp' | tail -20 || true
    exit 1
fi

echo "✓ All pods are ready!"

echo ""
echo "================================================"
echo "MatrixHub Deployment Complete!"
echo "================================================"
echo "To check MatrixHub status:"
echo "  kubectl get pods -n matrixhub"
echo "  kubectl get svc -n matrixhub"