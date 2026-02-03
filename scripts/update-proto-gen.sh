#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

PROTOVENDOR_DIR="${ROOT_DIR}/api/proto/vendor"
PROTO_DIR="${ROOT_DIR}/api/proto"
SWAGGER_DIR="${ROOT_DIR}/api/openapiv2"
TS_DIR="${ROOT_DIR}/api/ts"
GO_DIR="${ROOT_DIR}/api/go"
GOBIN="${ROOT_DIR}/bin"
PATH="${GOBIN}:${PATH}"

# Install protoc
install_protoc() {
  local os_name=$(uname -s | tr '[:upper:]' '[:lower:]')
  local arch_name=$(uname -m)
  local protoc_version="33.5"
  if [[ -x "${ROOT_DIR}/bin/protoc" ]]; then
    echo "protoc ${protoc_version} is already installed."
    return
  fi

  if [[ "$arch_name" == "x86_64" || "$arch_name" == "amd64" ]]; then
    arch_name="x86_64"
  elif [[ "$arch_name" == "aarch64" || "$arch_name" == "arm64" ]]; then
    arch_name="aarch_64"
  else
    echo "Unsupported architecture: $arch_name"
    exit 1
  fi

  if [[ "$os_name" == "linux" ]]; then
    os_name="linux"
  elif [[ "$os_name" == "darwin" ]]; then
    os_name="osx"
  else
    echo "Unsupported OS: $os_name"
    exit 1
  fi

  local protoc_zip="protoc-${protoc_version}-${os_name}-${arch_name}.zip"
  local download_url="https://github.com/protocolbuffers/protobuf/releases/download/v${protoc_version}/${protoc_zip}"
  echo "${download_url}"

  wget -O "${protoc_zip}" "${download_url}"
  unzip -o "${protoc_zip}" -d "${ROOT_DIR}" -x "readme.txt"
  rm -f "${protoc_zip}"
}

install_protoc

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.27.7
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.7
go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts@v1.1.2
go install github.com/envoyproxy/protoc-gen-validate@v1.3.0

VERSIONS=(
  "v1alpha1"
)

rm -rf "${GO_DIR}" "${SWAGGER_DIR}" "${TS_DIR}"

mkdir -p "${GO_DIR}" "${SWAGGER_DIR}" "${TS_DIR}"

for version in ${VERSIONS[@]}; do
  for proto in "${PROTO_DIR}/${version}"/*.proto; do

    protoc \
      "${proto}" \
      --proto_path=".:${PROTOVENDOR_DIR}" \
      --proto_path=".:${PROTO_DIR}" \
      --go_out="${GO_DIR}" \
      --go_opt=paths=source_relative \
      --go-grpc_out="${GO_DIR}" \
      --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false \
      --grpc-gateway_out="${GO_DIR}" \
      --grpc-gateway_opt=paths=source_relative \
      --validate_out="lang=go:${GO_DIR}" \
      --validate_opt=paths=source_relative \
      --openapiv2_out="${SWAGGER_DIR}" \
      --grpc-gateway-ts_out="${TS_DIR}"
  done
done
