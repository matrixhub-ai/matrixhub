#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

function check() {
  echo "Verify proto generation"
  "${ROOT_DIR}"/scripts/update-proto-gen.sh
  git --no-pager diff --exit-code
}

cd "${ROOT_DIR}" && check
