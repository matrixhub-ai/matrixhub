#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"
ROOT_DIR="$(realpath "${DIR}/..")"
FETCH_FILE="${ROOT_DIR}/api/ts/fetch.pb.ts"
CLEANUP_FILE="${ROOT_DIR}/api/ts/v1alpha1/cleanup.pb.ts"

if [[ ! -f "${FETCH_FILE}" ]]; then
  echo "missing ${FETCH_FILE}" >&2
  exit 1
fi

if [[ -f "${CLEANUP_FILE}" ]]; then
  perl -0pi -e 's/^import \* as GoogleProtobufWrappers from "\.\.\/google\/protobuf\/wrappers\.pb"\n//m; s/sweepDone\?: GoogleProtobufWrappers\.BoolValue/sweepDone?: boolean | null/g' "${CLEANUP_FILE}"
fi

if grep -q "export type FetchFn = typeof fetch" "${FETCH_FILE}"; then
  exit 0
fi

grep -q "export interface InitReq extends RequestInit" "${FETCH_FILE}"
grep -q "return fetch(url, req)" "${FETCH_FILE}"
grep -q "const result = await fetch(url, req)" "${FETCH_FILE}"

perl -0pi -e 's/export interface InitReq extends RequestInit \{\n  pathPrefix\?: string\n\}\n/export interface InitReq extends RequestInit {\n  pathPrefix?: string\n}\n\nexport type FetchFn = typeof fetch\n\nlet fetchFn: FetchFn = (input, init) => fetch(input, init)\n\nexport function setFetchFn(fetcher: FetchFn) {\n  fetchFn = fetcher\n}\n\nexport function resetFetchFn() {\n  fetchFn = (input, init) => fetch(input, init)\n}\n/s' "${FETCH_FILE}"

perl -0pi -e 's/return fetch\(url, req\)/return fetchFn(url, req)/g' "${FETCH_FILE}"
perl -0pi -e 's/const result = await fetch\(url, req\)/const result = await fetchFn(url, req)/g' "${FETCH_FILE}"
