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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

fail() { echo "Error: $*" >&2; exit 1; }
is_true() { case "$1" in 1|true|TRUE|True|yes|YES|Yes) return 0 ;; *) return 1 ;; esac; }
require_command() { command -v "$1" >/dev/null 2>&1 || fail "required command is not installed: $1"; }

discover_host() {
    local address="" interface=""
    [ -z "${UI_AUTOMATION_HOST:-}" ] || { printf '%s\n' "${UI_AUTOMATION_HOST}"; return; }
    case "$(uname -s)" in
        Darwin)
            interface=$(route -n get default 2>/dev/null | awk '/interface:/{print $2; exit}' || true)
            [ -z "${interface}" ] || address=$(ipconfig getifaddr "${interface}" 2>/dev/null || true)
            ;;
        Linux)
            if command -v ip >/dev/null 2>&1; then
                address=$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++)if($i=="src"){print $(i+1);exit}}' || true)
            else
                address=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
            fi
            ;;
    esac
    printf '%s\n' "${address:-host.docker.internal}"
}

validate_port() {
    [[ "$2" =~ ^[0-9]+$ ]] && [ "$2" -ge 1 ] && [ "$2" -le 65535 ] || fail "$1 must be an integer between 1 and 65535"
}

cleanup() {
    local exit_code=$?
    trap - EXIT INT TERM
    if is_true "${UI_AUTOMATION_KEEP_CLUSTER}"; then
        echo "Keeping KIND cluster for debugging: ${UI_AUTOMATION_CLUSTER_NAME}"
    elif ! kind delete cluster --name="${UI_AUTOMATION_CLUSTER_NAME}"; then
        echo "Error: failed to delete KIND cluster ${UI_AUTOMATION_CLUSTER_NAME}" >&2
        [ "${exit_code}" -ne 0 ] || exit_code=1
    fi
    exit "${exit_code}"
}

main() {
    local command_name host_address
    cd "${PROJECT_ROOT}"

    export UI_AUTOMATION_MODE="${UI_AUTOMATION_MODE:-local}"
    export UI_AUTOMATION_CLUSTER_NAME="${UI_AUTOMATION_CLUSTER_NAME:-matrixhub-ui-automation}"
    export UI_AUTOMATION_KIND_IMAGE_TAG="${UI_AUTOMATION_KIND_IMAGE_TAG:-v1.32.3}"
    export UI_AUTOMATION_MATRIXHUB_IMAGE="${UI_AUTOMATION_MATRIXHUB_IMAGE:-${E2E_MATRIXHUB_IMAGE:-ghcr.io/matrixhub-ai/matrixhub:latest}}"
    export UI_AUTOMATION_HTTP_PORT="${UI_AUTOMATION_HTTP_PORT:-30101}"
    export UI_AUTOMATION_SSH_PORT="${UI_AUTOMATION_SSH_PORT:-30122}"
    export UI_AUTOMATION_LISTEN_ADDRESS="${UI_AUTOMATION_LISTEN_ADDRESS:-0.0.0.0}"
    export UI_AUTOMATION_BUILD_IMAGE="${UI_AUTOMATION_BUILD_IMAGE:-false}"
    export UI_AUTOMATION_KEEP_CLUSTER="${UI_AUTOMATION_KEEP_CLUSTER:-false}"
    export UI_AUTOMATION_VERIFY_RUNNER_ACCESS="${UI_AUTOMATION_VERIFY_RUNNER_ACCESS:-true}"

    host_address=$(discover_host)
    export UI_AUTOMATION_BASE_URL="${UI_AUTOMATION_BASE_URL:-http://${host_address}:${UI_AUTOMATION_HTTP_PORT}/}"
    export UI_AUTOMATION_HEALTH_URL="${UI_AUTOMATION_HEALTH_URL:-http://127.0.0.1:${UI_AUTOMATION_HTTP_PORT}/}"

    validate_port UI_AUTOMATION_HTTP_PORT "${UI_AUTOMATION_HTTP_PORT}"
    validate_port UI_AUTOMATION_SSH_PORT "${UI_AUTOMATION_SSH_PORT}"
    [[ "${UI_AUTOMATION_LISTEN_ADDRESS}" =~ ^[0-9A-Fa-f:.]+$ ]] || fail "invalid UI_AUTOMATION_LISTEN_ADDRESS"
    for command_name in docker helm kind kubectl; do require_command "${command_name}"; done
    is_true "${UI_AUTOMATION_BUILD_IMAGE}" && require_command make
    "${SCRIPT_DIR}/run-ui-automation.sh" --validate-only

    if is_true "${UI_AUTOMATION_BUILD_IMAGE}"; then
        export UI_AUTOMATION_MATRIXHUB_IMAGE="ghcr.io/matrixhub-ai/matrixhub:${UI_AUTOMATION_IMAGE_TAG:-ui-automation-local}"
        make image-build VERSION="${UI_AUTOMATION_IMAGE_TAG:-ui-automation-local}"
    fi

    echo "UI Automation KIND: cluster=${UI_AUTOMATION_CLUSTER_NAME}, image=${UI_AUTOMATION_MATRIXHUB_IMAGE}"
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM

    E2E_CLUSTER_NAME="${UI_AUTOMATION_CLUSTER_NAME}" \
    E2E_KIND_IMAGE_TAG="${UI_AUTOMATION_KIND_IMAGE_TAG}" \
    E2E_HTTP_HOST_PORT="${UI_AUTOMATION_HTTP_PORT}" \
    E2E_SSH_HOST_PORT="${UI_AUTOMATION_SSH_PORT}" \
    E2E_KIND_HTTP_LISTEN_ADDRESS="${UI_AUTOMATION_LISTEN_ADDRESS}" \
    E2E_KIND_SSH_LISTEN_ADDRESS=127.0.0.1 "${SCRIPT_DIR}/setup-kind-cluster.sh"

    E2E_CLUSTER_NAME="${UI_AUTOMATION_CLUSTER_NAME}" \
    E2E_MATRIXHUB_IMAGE="${UI_AUTOMATION_MATRIXHUB_IMAGE}" \
    E2E_MATRIXHUB_HOST_URL="${UI_AUTOMATION_BASE_URL%/}" "${SCRIPT_DIR}/deploy-matrixhub.sh"

    "${SCRIPT_DIR}/run-ui-automation.sh"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
