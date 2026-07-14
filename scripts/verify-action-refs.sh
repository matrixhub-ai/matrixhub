#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"
ROOT_DIR="$(realpath "${DIR}/..")"

cd "${ROOT_DIR}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required to validate GitHub action commit and release refs"
  exit 1
fi

status=0

while IFS= read -r line; do
  file="${line%%:*}"
  rest="${line#*:}"
  lineno="${rest%%:*}"
  value="${line#*uses:}"
  value="${value%%#*}"
  value="$(echo "${value}" | xargs)"
  value="${value#\"}"
  value="${value%\"}"
  value="${value#\'}"
  value="${value%\'}"

  if [[ -z "${value}" || "${value}" == ./* || "${value}" == docker://* ]]; then
    continue
  fi
  if [[ "${value}" == *'$'* ]]; then
    echo "warning: ${file}:${lineno}: skipping dynamic action ref: ${value}"
    continue
  fi
  if [[ "${value}" != *@* ]]; then
    echo "error: ${file}:${lineno}: action ref is missing @ref: ${value}"
    status=1
    continue
  fi

  action_path="${value%@*}"
  ref="${value##*@}"
  owner_repo="$(echo "${action_path}" | cut -d/ -f1-2)"

  if [[ "${owner_repo}" != */* ]]; then
    echo "error: ${file}:${lineno}: invalid action reference: ${value}"
    status=1
    continue
  fi

  if git ls-remote --exit-code "https://github.com/${owner_repo}.git" "refs/tags/${ref}" >/dev/null 2>&1 ||
     git ls-remote --exit-code "https://github.com/${owner_repo}.git" "refs/heads/${ref}" >/dev/null 2>&1 ||
     gh api "repos/${owner_repo}/commits/${ref}" >/dev/null 2>&1 ||
     gh api "repos/${owner_repo}/releases/tags/${ref}" >/dev/null 2>&1; then
    echo "OK ${value}"
  else
    echo "error: ${file}:${lineno}: cannot resolve action ref: ${value}"
    status=1
  fi
done < <(grep -RInE '^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*[^[:space:]]+' .github/workflows .github/actions || true)

exit "${status}"
