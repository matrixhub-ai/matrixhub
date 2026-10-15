#!/usr/bin/env bash
# Build this checkout, not the upstream published image. No credentials/data are
# copied into the Docker context. Run from WSL/Linux with Docker available.
set -euo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"
mkdir -p .mh-local/gomod .mh-local/gocache .mh-local/runtime-bundle
docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp \
  -e GOTOOLCHAIN=go1.26.9 -e GOMODCACHE=/workspace/.mh-local/gomod \
  -e GOCACHE=/workspace/.mh-local/gocache -v "$repo_root:/workspace" \
  -w /workspace golang:1.23-bookworm \
  go build -o .mh-local/matrixhub ./cmd/matrixhub
docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -e CI=true \
  -v "$repo_root:/workspace" -w /workspace/ui node:24-bookworm-slim \
  sh -c 'npx --yes pnpm@10.17.0 install --frozen-lockfile --ignore-scripts && npx --yes pnpm@10.17.0 build'
cp .mh-local/matrixhub .mh-local/runtime-bundle/matrixhub
cp deploy/security-prototype/Dockerfile.runtime .mh-local/runtime-bundle/Dockerfile.runtime
cp -a db/migrations .mh-local/runtime-bundle/
cp -a ui/dist/. .mh-local/runtime-bundle/ui/
printf '%s\n' 'Runtime bundle prepared. Start with: docker compose -f deploy/security-prototype/compose.yaml up -d --build'
