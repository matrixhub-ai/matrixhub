#!/bin/bash

set -e
set -o pipefail
set -u
set -x

bash_source_path="$(dirname "${BASH_SOURCE[0]}")"
pro_path=$(cd "$bash_source_path/.."; pwd)

function gen_openapi_sdk() {
    swagger_path=${1:-"$pro_path/api/openapiv2"}
    client_path="$pro_path/test/client"

    mkdir -p "$client_path"

    local versions
    versions=$(ls "$swagger_path")

    for version in $versions; do
        local files
        files=$(ls "$swagger_path/$version")
        for file in $files; do
            if [[ $file == *.swagger.json ]]; then
                local name="${file%.swagger.json}"
                local target="$client_path/$version/$name"
                rm -rf "$target" && mkdir -p "$target"
                docker run --rm -u "$(id -u):$(id -g)" \
                    -v "$swagger_path/$version":/swagger \
                    -v "$target":/gen \
                    docker.m.daocloud.io/swaggerapi/swagger-codegen-cli generate \
                    -i /swagger/"$file" \
                    -l go \
                    -o /gen \
                    --additional-properties "packageName=$version,isGoSubmodule=true"

                find "$target" -type f ! -name "*.go" -delete

                rm -f "$target/go.mod" "$target/go.sum" 2>/dev/null || true

                rm -rf "$target/.git" "$target/docs" "$target/api" 2>/dev/null || true
            fi
        done
    done
}

gen_openapi_sdk
