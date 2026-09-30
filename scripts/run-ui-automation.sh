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

fail() { echo "Error: $*" >&2; return 1; }
is_true() { case "$1" in 1|true|TRUE|True|yes|YES|Yes) return 0 ;; *) return 1 ;; esac; }
require_command() { command -v "$1" >/dev/null 2>&1 || fail "required command is not installed: $1"; }
project_path() { case "$1" in /*) printf '%s\n' "$1" ;; *) printf '%s/%s\n' "${PROJECT_ROOT}" "$1" ;; esac; }

normalize_url() {
    case "$1" in http://*|https://*) ;; *) fail "UI_AUTOMATION_BASE_URL must start with http:// or https://"; return ;; esac
    if [[ "$1" =~ [[:space:]#] ]]; then
        fail "UI_AUTOMATION_BASE_URL must not contain whitespace or #"
        return
    fi
    case "$1" in */) printf '%s\n' "$1" ;; *) printf '%s/\n' "$1" ;; esac
}

discover_framework() {
    local path=""
    if [ -n "${UI_AUTOMATION_FRAMEWORK_DIR:-}" ]; then
        path="${UI_AUTOMATION_FRAMEWORK_DIR}"
        case "${path}" in /*) ;; *) path="${PROJECT_ROOT}/${path}" ;; esac
        [ -f "${path}/package.json" ] && [ -f "${path}/src/ai-run.ts" ] || {
            fail "UI_AUTOMATION_FRAMEWORK_DIR is not a ui-factory-ai checkout: ${path}"
            return
        }
        (cd "${path}" && pwd)
        return
    fi
    for path in "${PROJECT_ROOT}/../ui-factory-ai" \
        "${PROJECT_ROOT}/../../ai-factory-ai/ui-factory-ai" \
        "${PROJECT_ROOT}/../../ui-factory-ai"; do
        if [ -f "${path}/package.json" ] && [ -f "${path}/src/ai-run.ts" ]; then
            (cd "${path}" && pwd)
            return
        fi
    done
    fail "cannot find ui-factory-ai; set UI_AUTOMATION_FRAMEWORK_DIR"
}

read_run_id() {
    local run_id="${UI_AUTOMATION_RUN_ID:-}"
    if [ -z "${run_id}" ] && [ -f "${RUN_STATE_FILE}" ]; then
        run_id=$(awk '$1 == "runId:" { print $2; exit }' "${RUN_STATE_FILE}")
    fi
    [ -n "${run_id}" ] || { fail "set UI_AUTOMATION_RUN_ID or create an active ui-factory-ai task first"; return; }
    [[ "${run_id}" =~ ^[A-Za-z0-9._:-]+$ ]] || { fail "invalid UI automation run ID: ${run_id}"; return; }
    printf '%s\n' "${run_id}"
}

update_base_url() {
    local file=$1 temporary mode
    temporary=$(mktemp "${file}.tmp.XXXXXX")
    if ! awk -v url="${UI_AUTOMATION_BASE_URL}" '
        BEGIN { local_block = 0; updated = 0 }
        /^local:[[:space:]]*(#.*)?$/ { local_block = 1; print; next }
        local_block && /^[^[:space:]#]/ { local_block = 0 }
        local_block && /^[[:space:]]+baseUrl:[[:space:]]*/ {
            match($0, /^[[:space:]]*/); indent = substr($0, 1, RLENGTH); comment = ""
            if (match($0, /[[:space:]]+#/)) comment = substr($0, RSTART)
            print indent "baseUrl: " url comment; updated++; next
        }
        { print }
        END { if (updated != 1) exit 42 }
    ' "${file}" > "${temporary}"; then
        rm -f "${temporary}"
        fail "expected exactly one local.baseUrl entry in ${file}"
        return
    fi
    mode=$(stat -f '%Lp' "${file}" 2>/dev/null || stat -c '%a' "${file}")
    chmod "${mode}" "${temporary}"
    mv "${temporary}" "${file}"
}

wait_for_ui() {
    local elapsed=0 interval=5
    echo "Waiting for MatrixHub UI at ${UI_AUTOMATION_HEALTH_URL}..."
    while [ "${elapsed}" -lt "${UI_AUTOMATION_UI_TIMEOUT}" ]; do
        if curl --fail --silent --show-error --location --max-time 5 --output /dev/null "${UI_AUTOMATION_HEALTH_URL}"; then
            echo "MatrixHub UI is reachable."
            return
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done
    fail "timed out waiting for MatrixHub UI at ${UI_AUTOMATION_HEALTH_URL}"
}

configure() {
    UI_AUTOMATION_MODE="${UI_AUTOMATION_MODE:-local}"
    UI_AUTOMATION_HTTP_PORT="${UI_AUTOMATION_HTTP_PORT:-3000}"
    UI_AUTOMATION_BASE_URL=$(normalize_url "${UI_AUTOMATION_BASE_URL:-http://127.0.0.1:${UI_AUTOMATION_HTTP_PORT}/}")
    UI_AUTOMATION_HEALTH_URL="${UI_AUTOMATION_HEALTH_URL:-${UI_AUTOMATION_BASE_URL}}"
    UI_AUTOMATION_UI_TIMEOUT="${UI_AUTOMATION_UI_TIMEOUT:-60}"
    UI_AUTOMATION_VERIFY_RUNNER_ACCESS="${UI_AUTOMATION_VERIFY_RUNNER_ACCESS:-false}"
    UI_AUTOMATION_RUNNER_NETWORK="${UI_AUTOMATION_RUNNER_NETWORK:-host}"
    AUTOMATION_ROOT=$(project_path "${UI_AUTOMATION_ROOT:-test/ui-automation}")
    MANIFEST_FILE=$(project_path "${UI_AUTOMATION_MANIFEST_FILE:-test/ui-automation/config/manifest.yaml}")
    ENVIRONMENT_FILE=$(project_path "${UI_AUTOMATION_ENV_FILE:-test/ui-automation/ai/environment.yaml}")
    RUN_STATE_FILE=$(project_path "${UI_AUTOMATION_RUN_STATE_FILE:-test/ui-automation/.ui-factory/runs/default/active-run.yaml}")
    REPORT_DIR="${UI_AUTOMATION_REPORT_DIR:-}"
    [ -z "${REPORT_DIR}" ] || REPORT_DIR=$(project_path "${REPORT_DIR}")
    if [ "${UI_AUTOMATION_MODE}" = ci ] && [ -z "${REPORT_DIR}" ]; then
        REPORT_DIR="${AUTOMATION_ROOT}/reports"
    fi
}

validate() {
    local command_name image_tag
    case "${UI_AUTOMATION_MODE}" in local|ci) ;; *) fail "UI_AUTOMATION_MODE must be local or ci (got: ${UI_AUTOMATION_MODE})"; return ;; esac
    for command_name in awk curl mktemp stat; do require_command "${command_name}"; done
    [ -d "${AUTOMATION_ROOT}" ] || fail "UI automation root not found: ${AUTOMATION_ROOT}"
    [ -f "${MANIFEST_FILE}" ] || fail "UI automation manifest not found: ${MANIFEST_FILE}"

    if [ "${UI_AUTOMATION_MODE}" = local ]; then
        require_command npm
        [ -f "${ENVIRONMENT_FILE}" ] || fail "UI automation environment file not found: ${ENVIRONMENT_FILE}"
        FRAMEWORK_DIR=$(discover_framework)
        [ -d "${FRAMEWORK_DIR}/node_modules" ] || fail "run npm ci in ${FRAMEWORK_DIR} first"
        RUN_ID=$(read_run_id)
    else
        require_command docker
        require_command id
        [ -n "${UI_AUTOMATION_RUNNER_IMAGE:-}" ] || fail "UI_AUTOMATION_RUNNER_IMAGE is required in CI mode"
        if [[ "${UI_AUTOMATION_RUNNER_IMAGE}" != *@sha256:* ]]; then
            image_tag="${UI_AUTOMATION_RUNNER_IMAGE##*/}"
            [[ "${image_tag}" == *:* && "${image_tag}" != *:latest ]] || fail "UI_AUTOMATION_RUNNER_IMAGE must use a fixed tag or digest"
        fi
    fi
    is_true "${UI_AUTOMATION_VERIFY_RUNNER_ACCESS}" && require_command docker
    return 0
}

run_local() {
    local args=(run ai:run -- "${UI_AUTOMATION_SELECTION:-matrixhub}" --run-id "${RUN_ID}")
    [ -z "${REPORT_DIR}" ] || args+=(--report-dir "${REPORT_DIR}")
    (cd "${FRAMEWORK_DIR}" && UI_TARGET_PROJECT_DIR="${PROJECT_ROOT}" npm "${args[@]}")
}

run_ci() {
    local args=(
        run --rm --pull=never --network="${UI_AUTOMATION_RUNNER_NETWORK}" --shm-size=1g
        --read-only --init --cap-drop=ALL --security-opt=no-new-privileges
        --pids-limit=512 --tmpfs /tmp:rw,nosuid,nodev,size=1g
        --user "$(id -u):$(id -g)" --workdir "${AUTOMATION_ROOT}"
        --env HOME=/tmp/ui-factory-home --env CI --env BASE_URL --env USERNAME --env PASSWORD
        --env REPORT_DIR --env ARTIFACT_PROFILE
        --mount "type=bind,src=${AUTOMATION_ROOT},dst=${AUTOMATION_ROOT},readonly"
        --mount "type=bind,src=${REPORT_DIR},dst=${REPORT_DIR}"
    )
    [ -z "${UI_AUTOMATION_RUN_ID:-}" ] || args+=(--env UI_FACTORY_RUN_ID)
    args+=("${UI_AUTOMATION_RUNNER_IMAGE}" "${MANIFEST_FILE}")
    CI=true BASE_URL="${UI_AUTOMATION_BASE_URL}" \
        USERNAME="${UI_AUTOMATION_USERNAME:-admin}" PASSWORD="${UI_AUTOMATION_PASSWORD:-changeme}" \
        REPORT_DIR="${REPORT_DIR}" ARTIFACT_PROFILE="${UI_AUTOMATION_ARTIFACT_PROFILE:-failure}" \
        UI_FACTORY_RUN_ID="${UI_AUTOMATION_RUN_ID:-}" docker "${args[@]}"
}

main() {
    cd "${PROJECT_ROOT}"
    configure
    validate
    [ "${1:-}" = --validate-only ] && return

    if [ "${UI_AUTOMATION_MODE}" = ci ]; then
        mkdir -p "${REPORT_DIR}"
        docker pull "${UI_AUTOMATION_RUNNER_IMAGE}"
        docker image inspect "${UI_AUTOMATION_RUNNER_IMAGE}" >/dev/null
    fi

    echo "UI Automation: mode=${UI_AUTOMATION_MODE}, url=${UI_AUTOMATION_BASE_URL}"
    wait_for_ui
    if is_true "${UI_AUTOMATION_VERIFY_RUNNER_ACCESS}"; then
        docker run --rm --network="${UI_AUTOMATION_RUNNER_NETWORK}" busybox:latest \
            wget -q -T 10 -O /dev/null "${UI_AUTOMATION_BASE_URL%/}/healthz"
    fi
    [ "${UI_AUTOMATION_MODE}" != local ] || update_base_url "${ENVIRONMENT_FILE}"

    local exit_code=0
    if [ "${UI_AUTOMATION_MODE}" = local ]; then
        run_local || exit_code=$?
    else
        run_ci || exit_code=$?
    fi
    if [ "${exit_code}" -ne 0 ]; then
        echo "UI Automation failed with exit code ${exit_code}." >&2
        return "${exit_code}"
    fi
    return "${exit_code}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
