#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PROTO_DIR=api/proto
PROTO_VERSION=("v1alpha1")
PROTOVENDOR_DIR=./../../api/protovendor
SWAGGER_DIR=./../../api/openapiv2
TS_DIR=./../../api/ts
cd ${PROTO_DIR}

protoc_version=$(protoc --version)
if [ "${protoc_version#"libprotoc "}" != "3.19.4" ]; then
  which protoc
  echo "please install protoc version 3.19.4"
  echo "🚀 Download address: https://github.com/protocolbuffers/protobuf/releases/tag/v3.19.4"
  exit 1
fi

go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.15.2
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.15.2
go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts@v1.1.2
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

SERVICES=(
  "project"
)

V1ALPHA=0
for version in ${PROTO_VERSION[@]}
  do
  PROTOGEN_DIR=./../../api/protogen
  for service in ${SERVICES[V1ALPHA]}
    do
    protoc \
    --proto_path=".:${PROTOVENDOR_DIR}" \
    --proto_path=".:./../../api/proto" "$version/$service.proto" \
    --go_out ${PROTOGEN_DIR}  \
    --go-grpc_opt require_unimplemented_servers=false \
    --go-grpc_out ${PROTOGEN_DIR}  \
    --grpc-gateway_out ${PROTOGEN_DIR} \
    --grpc-gateway_opt logtostderr=true \
    --openapiv2_out ${SWAGGER_DIR} \
    --openapiv2_opt logtostderr=true \
    --grpc-gateway-ts_out=${TS_DIR} \
    --validate_out="lang=go:${PROTOGEN_DIR}"
  done
  V1ALPHA=$V1ALPHA+1
done