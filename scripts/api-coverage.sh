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

# ---------------------------------------------------------------------------
# E2E API coverage harness (ported from leopard's hack/utils.sh).
#
# Runs a mitmproxy-based recorder container that captures every API call the
# e2e suite makes, then diffs the recorded traffic against the OpenAPI specs
# with swagger-coverage-commandline and prints a per-module coverage report.
#
# The e2e HTTP client already honors http_proxy/https_proxy
# (test/tools/http_client.go uses http.ProxyFromEnvironment), so pointing those
# at the recorder is enough to capture traffic -- with one caveat handled by
# the caller: Go's ProxyFromEnvironment bypasses "localhost" and loopback IPs,
# so MATRIXHUB_BASE_URL must use a non-loopback hostname (e.g. matrixhub.local)
# for anything to be recorded.
#
# Usage (source, then call the lifecycle functions around the test run):
#   source scripts/api-coverage.sh
#   api_coverage::start
#   <run ginkgo>
#   api_coverage::report
#   api_coverage::stop
# ---------------------------------------------------------------------------

# Public, network-reachable mirror of the internal api-test-coverage image
# (mitmproxy recorder + swagger-coverage-commandline).
API_COVERAGE_IMAGE="${API_COVERAGE_IMAGE:-release.daocloud.io/matrixhub/api-test-coverage:1.0.0}"

# Directory holding the generated OpenAPI v2 specs.
API_COVERAGE_OPENAPI_DIR="${API_COVERAGE_OPENAPI_DIR:-./api/openapiv2}"

# Merged swagger spec passed to the recorder for the "total" coverage figure.
API_COVERAGE_MERGED_SWAGGER="${API_COVERAGE_MERGED_SWAGGER:-./merged_swagger.json}"

# Randomized container name so concurrent runs on one host do not collide.
API_COVERAGE_CONTAINER="${API_COVERAGE_CONTAINER:-matrixhub_api_coverage_${RANDOM}${RANDOM}}"

# Merge every *.swagger.json under $1 into a single swagger doc at $2.
# Combines paths + definitions; keeps the first file's top-level fields.
function api_coverage::merge_swagger() {
    local source_dir="${1:-${API_COVERAGE_OPENAPI_DIR}}"
    local output_file="${2:-${API_COVERAGE_MERGED_SWAGGER}}"

    if [ ! -d "${source_dir}" ]; then
        echo "api-coverage: openapi dir not found: ${source_dir}" >&2
        return 1
    fi

    local files=()
    while IFS= read -r f; do
        files+=("${f}")
    done < <(find "${source_dir}" -type f -name "*.swagger.json" | sort)
    if [ "${#files[@]}" -eq 0 ]; then
        echo "api-coverage: no *.swagger.json found under ${source_dir}" >&2
        return 1
    fi

    local main="${files[0]}"

    local tmp
    tmp=$(mktemp -d)
    jq -s 'map(.paths) | add' "${files[@]}" > "${tmp}/paths.json"
    jq -s 'map(.definitions) | add' "${files[@]}" > "${tmp}/definitions.json"
    jq 'del(.paths, .definitions)' "${main}" > "${tmp}/base.json"
    jq -s '
      .[0] as $base |
      .[1] as $paths |
      .[2] as $defs |
      $base + {paths: $paths} + {definitions: $defs}
    ' "${tmp}/base.json" "${tmp}/paths.json" "${tmp}/definitions.json" > "${output_file}"
    rm -rf "${tmp}"

    echo "api-coverage: merged ${#files[@]} specs into ${output_file}"
}

# Start the recorder container and export proxy env for the test process.
# Picks a random high port; the container runs on the host network so it can
# both listen on that port and forward to the service (via matrixhub.local).
function api_coverage::start() {
    local port=$(( (RANDOM % 30000) + 30000 ))

    export http_proxy="http://127.0.0.1:${port}"
    export https_proxy="http://127.0.0.1:${port}"
    export HTTP_PROXY="${http_proxy}"
    export HTTPS_PROXY="${https_proxy}"

    # Only the test's API traffic should traverse the recorder. Everything the Go
    # toolchain needs (module proxy, checksum db) must bypass it, or `ginkgo`'s
    # compile step fails to download modules through mitmproxy. Bypass loopback
    # and the Go module infrastructure, including whatever hosts GOPROXY/GOSUMDB
    # point at.
    local go_bypass="localhost,127.0.0.1,::1,proxy.golang.org,sum.golang.org,goproxy.cn,goproxy.io,storage.googleapis.com"
    local gp
    gp="$(go env GOPROXY GOSUMDB 2>/dev/null | tr ',' '\n' \
        | sed -nE 's#^[a-z]+://([^/:]+).*#\1#p' | paste -sd, - 2>/dev/null)"
    export no_proxy="${no_proxy:+${no_proxy},}${go_bypass}${gp:+,${gp}}"
    export NO_PROXY="${no_proxy}"

    api_coverage::merge_swagger "${API_COVERAGE_OPENAPI_DIR}" "${API_COVERAGE_MERGED_SWAGGER}" || return 1

    docker create -e MITMPROXY_PORT="${port}" --network host \
        --name "${API_COVERAGE_CONTAINER}" "${API_COVERAGE_IMAGE}" >/dev/null
    docker cp "${API_COVERAGE_MERGED_SWAGGER}" \
        "${API_COVERAGE_CONTAINER}:/merged_swagger.json" >/dev/null 2>&1 \
        || echo "api-coverage: failed to copy merged swagger into container" >&2
    docker start "${API_COVERAGE_CONTAINER}" >/dev/null

    # Wait until mitmdump is actually accepting connections; otherwise the first
    # proxied requests race the container start and get "connection refused".
    local i
    for i in $(seq 1 30); do
        if (exec 3<>"/dev/tcp/127.0.0.1/${port}") 2>/dev/null; then
            exec 3>&- 3<&- 2>/dev/null || true
            break
        fi
        sleep 1
    done

    echo "api-coverage: recorder started on ${http_proxy} (container ${API_COVERAGE_CONTAINER})"
    echo "api-coverage: no_proxy=${no_proxy}"
}

# Emit a line either to $GITHUB_STEP_SUMMARY (rendered on the run page) or stdout.
function api_coverage::_emit() {
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
        echo "$1" >> "${GITHUB_STEP_SUMMARY}"
    else
        echo "$1"
    fi
}

# Compute coverage from recorded traffic and print a Markdown report.
function api_coverage::report() {
    local container="${API_COVERAGE_CONTAINER}"

    # swagger-coverage-commandline throws a NullPointerException on an empty
    # record set, so short-circuit when nothing was captured (e.g. the suite
    # failed to compile/run) rather than surfacing a Java stack trace.
    local record_count
    record_count=$(docker exec "${container}" sh -c 'ls -1 /data/records/ 2>/dev/null | wc -l' | tr -d ' ')
    if [ -z "${record_count}" ] || [ "${record_count}" -eq 0 ]; then
        api_coverage::_emit "## E2E API Coverage"
        api_coverage::_emit ""
        api_coverage::_emit "_No API traffic was recorded (the suite may have failed before making requests)._"
        return 0
    fi

    local merged_output
    merged_output=$(docker exec "${container}" swagger-coverage-commandline \
        -i /data/records/ -c /data/result/ -s /merged_swagger.json | grep INFO)
    local merged_conditions merged_partial
    merged_conditions=$(echo "${merged_output}" | grep "Conditions:" | awk '{print $NF}')
    merged_partial=$(echo "${merged_output}" | grep "Partial coverage" | awk '{print $7}' | tr -d '\n')

    api_coverage::_emit "## E2E API Coverage"
    api_coverage::_emit ""
    api_coverage::_emit "| Module | Conditions | Partial Coverage |"
    api_coverage::_emit "| --- | --- | --- |"
    api_coverage::_emit "| **total** | ${merged_conditions:-N/A} | ${merged_partial:-N/A}% |"

    # Per-module figures, one swagger spec at a time, sorted by coverage desc.
    local results=()
    local swagger_file
    while IFS= read -r swagger_file; do
        [ -f "${swagger_file}" ] || continue
        local filename
        filename=$(basename "${swagger_file}" .swagger.json)
        docker cp "${swagger_file}" "${container}:/${filename}" >/dev/null 2>&1 || continue
        local output conditions partial
        output=$(docker exec "${container}" swagger-coverage-commandline \
            -i /data/records/ -c /data/result/ -s "/${filename}" | grep INFO)
        conditions=$(echo "${output}" | grep "Conditions:" | awk '{print $NF}')
        partial=$(echo "${output}" | grep "Partial coverage" | awk '{print $7}' | tr -d '\n')
        if [ "${conditions}" != "0/0" ] && [ -n "${partial}" ]; then
            results+=("${partial}|${filename}|${conditions}")
        fi
    done < <(find "${API_COVERAGE_OPENAPI_DIR}" -type f -name "*.swagger.json")

    local sorted
    IFS=$'\n' sorted=($(printf '%s\n' "${results[@]}" | sort -t'|' -k1,1nr)); unset IFS
    local row
    for row in "${sorted[@]}"; do
        IFS='|' read -r partial filename conditions <<< "${row}"
        api_coverage::_emit "| ${filename} | ${conditions:-N/A} | ${partial:-N/A}% |"
    done

    # Tested endpoints / case points recorded during the run.
    api_coverage::_emit ""
    api_coverage::_emit "<details><summary>Tested API endpoints</summary>"
    api_coverage::_emit ""
    api_coverage::_emit '```'
    printf '%s\n' "${merged_output}" | grep -B 1 -E "(✅|❌)" \
        | awk '{$1=$2=$3=""; print substr($0, 4)}' \
        | while IFS= read -r line; do api_coverage::_emit "${line}"; done
    api_coverage::_emit '```'
    api_coverage::_emit "</details>"
}

# Stop and remove the recorder container. Safe to call unconditionally.
function api_coverage::stop() {
    docker stop "${API_COVERAGE_CONTAINER}" >/dev/null 2>&1 || true
    docker rm "${API_COVERAGE_CONTAINER}" >/dev/null 2>&1 || true
    rm -f "${API_COVERAGE_MERGED_SWAGGER}" 2>/dev/null || true
}
