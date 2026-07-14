#!/usr/bin/env bash

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
PROJECT_ROOT="${SCRIPT_DIR}/.."

cd "${PROJECT_ROOT}"

UNIT_TEST_PKGS="${UNIT_TEST_PKGS:-./cmd/... ./internal/...}"
UNIT_TEST_EXCLUDE_PKGS="${UNIT_TEST_EXCLUDE_PKGS-./internal/apiserver/handler/hf}"
COUNT="${COUNT:-1}"
VERBOSE="${VERBOSE:-false}"
RACE="${RACE:-false}"
COVERAGE="${COVERAGE:-false}"
COVERAGE_PROFILE="${COVERAGE_PROFILE:-coverage.out}"
COVERAGE_HTML="${COVERAGE_HTML:-false}"
COVERAGE_HTML_FILE="${COVERAGE_HTML_FILE:-${COVERAGE_PROFILE%.*}.html}"

args=(-short "-count=${COUNT}")

if [[ "${VERBOSE}" == "true" ]]; then
	args+=(-v)
fi

if [[ "${RACE}" == "true" ]]; then
	args+=(-race)
fi

if [[ "${COVERAGE}" == "true" ]]; then
	args+=("-coverprofile=${COVERAGE_PROFILE}" -covermode=atomic)
fi

# Intentional word splitting: package variables are space-separated lists of Go
# package patterns, e.g. "./cmd/... ./internal/...".
read -r -a package_patterns <<< "${UNIT_TEST_PKGS}"

if [[ ${#package_patterns[@]} -eq 0 ]]; then
	echo "UNIT_TEST_PKGS is empty" >&2
	exit 1
fi

packages=()
while IFS= read -r pkg; do
	packages+=("${pkg}")
done < <(go list "${package_patterns[@]}")

if [[ -n "${UNIT_TEST_EXCLUDE_PKGS}" ]]; then
	read -r -a exclude_patterns <<< "${UNIT_TEST_EXCLUDE_PKGS}"
	excluded_packages=()
	while IFS= read -r pkg; do
		excluded_packages+=("${pkg}")
	done < <(go list "${exclude_patterns[@]}")

	filtered_packages=()
	for pkg in "${packages[@]}"; do
		skip=false
		for excluded_pkg in "${excluded_packages[@]}"; do
			if [[ "${pkg}" == "${excluded_pkg}" ]]; then
				skip=true
				break
			fi
		done
		if [[ "${skip}" == "false" ]]; then
			filtered_packages+=("${pkg}")
		fi
	done
	packages=("${filtered_packages[@]}")
fi

if [[ ${#packages[@]} -eq 0 ]]; then
	echo "No unit-test packages selected" >&2
	exit 1
fi

echo "Running unit tests:"
echo "  packages: ${UNIT_TEST_PKGS}"
if [[ -n "${UNIT_TEST_EXCLUDE_PKGS}" ]]; then
	echo "  excludes: ${UNIT_TEST_EXCLUDE_PKGS}"
fi
echo "  coverage: ${COVERAGE}"
echo "  race: ${RACE}"
echo ""

go test "${args[@]}" "${packages[@]}"

if [[ "${COVERAGE}" == "true" ]]; then
	echo ""
	go tool cover -func="${COVERAGE_PROFILE}"

	if [[ "${COVERAGE_HTML}" == "true" ]]; then
		go tool cover -html="${COVERAGE_PROFILE}" -o "${COVERAGE_HTML_FILE}"
		echo ""
		echo "Coverage HTML written to ${COVERAGE_HTML_FILE}"
	fi
fi
