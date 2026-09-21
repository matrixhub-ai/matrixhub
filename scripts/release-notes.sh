#!/bin/bash

# Copyright 2026 MatrixHub Authors
# SPDX-License-Identifier: Apache-2.0

# Generate GitHub Release notes for an official tag from CHANGELOG/ files.
#
# Usage:
#   ./scripts/release-notes.sh <output-dir> <dest-tag>
#
# If CHANGELOG is not present at the checked-out ref (for example when promoting
# v0.1.0 on the same commit as v0.1.0-rc.1), the script falls back to
# origin/main for the CHANGELOG file.

set -euo pipefail

OUTPUT_DIR=${1:-}
DEST_TAG=${2:-}

if [ -z "${OUTPUT_DIR}" ] || [ -z "${DEST_TAG}" ]; then
  echo "usage: $0 <output-dir> <dest-tag>" >&2
  exit 1
fi

if ! [[ "${DEST_TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: dest tag must be an official release (vX.Y.Z), got ${DEST_TAG}" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
OUTPUT_DIR=$(cd "${OUTPUT_DIR}" && pwd)

VERSION="${DEST_TAG#v}"
MAJOR_MINOR=$(echo "${VERSION}" | cut -d. -f1,2)
CHANGELOG_PATH="CHANGELOG/CHANGELOG-${MAJOR_MINOR}.md"
OUTPUT_FILE="${OUTPUT_DIR}/release_notes_${DEST_TAG}.md"

extract_section() {
  local file=$1
  awk -v tag="${DEST_TAG}" '
    BEGIN { found=0 }
    $0 ~ "^## " tag "$" { found=1; print; next }
    found && $0 ~ "^## v[0-9]+\\.[0-9]+\\.[0-9]+" { exit }
    found { print }
  ' "${file}"
}

resolve_changelog_file() {
  if [ -f "${CHANGELOG_PATH}" ]; then
    echo "${CHANGELOG_PATH}"
    return
  fi

  if git show "origin/main:${CHANGELOG_PATH}" >/dev/null 2>&1; then
    git show "origin/main:${CHANGELOG_PATH}" > "/tmp/${CHANGELOG_PATH//\//_}"
    echo "/tmp/${CHANGELOG_PATH//\//_}"
    return
  fi

  echo ""
}

main() {
  git fetch origin main --quiet 2>/dev/null || true

  local source
  source=$(resolve_changelog_file)
  if [ -z "${source}" ]; then
    echo "error: missing ${CHANGELOG_PATH} at current ref and on origin/main" >&2
    exit 1
  fi

  extract_section "${source}" > "${OUTPUT_FILE}"

  cat >> "${OUTPUT_FILE}" <<EOF

## Supply-chain metadata

The release container image is signed with Cosign and includes an SPDX SBOM.
See the [Security Policy](https://github.com/matrixhub-ai/matrixhub/blob/${DEST_TAG}/SECURITY.md)
for verification instructions and dependency vulnerability handling.
EOF

  if [ "$(wc -l < "${OUTPUT_FILE}")" -lt 5 ]; then
    echo "error: release notes section for ${DEST_TAG} is empty in ${CHANGELOG_PATH}" >&2
    exit 1
  fi

  echo "Wrote ${OUTPUT_FILE}"
  cat "${OUTPUT_FILE}"
}

main
