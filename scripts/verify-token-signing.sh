#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

export MATRIXHUB_TOKEN_SIGNING_SECRET=deployment-contract-secret-32-bytes
chart=deploy/charts/matrixhub

for mode in generated explicit existing; do
  args=()
  case "${mode}" in
    explicit) args+=(--set-string "apiserver.tokenSigningSecret=${MATRIXHUB_TOKEN_SIGNING_SECRET}") ;;
    existing) args+=(--set-string apiserver.tokenSigningExistingSecret=external-signing-key) ;;
  esac
  helm template signing-contract "${chart}" "${args[@]}" | ruby -ryaml -rbase64 -e '
    documents = YAML.load_stream(STDIN.read).compact
    secret = documents.find { |doc| doc["kind"] == "Secret" && doc.dig("data", "matrixhub.dsn") }
    deployment = documents.find { |doc| doc["kind"] == "Deployment" && doc.dig("spec", "template", "spec", "containers").any? { |container| container["env"]&.any? { |env| env["name"] == "MATRIXHUB_TOKEN_SIGNING_SECRET" } } }
    abort "missing signing secret injection" unless deployment
    container = deployment.dig("spec", "template", "spec", "containers").find { |entry| entry["env"]&.any? { |env| env["name"] == "MATRIXHUB_TOKEN_SIGNING_SECRET" } }
    ref = container["env"].find { |env| env["name"] == "MATRIXHUB_TOKEN_SIGNING_SECRET" }.dig("valueFrom", "secretKeyRef")
    abort "wrong secret key" unless ref["key"] == "token-signing-secret"
    encoded = secret.dig("data", "token-signing-secret")
    if ARGV[0] == "existing"
      abort "existing secret ignored" unless ref["name"] == "external-signing-key" && encoded.nil?
    else
      value = Base64.strict_decode64(encoded || "")
      abort "wrong secret reference" unless ref["name"] == secret["metadata"]["name"]
      abort "insecure generated secret" unless value.bytesize >= 32 && value != "insecure-dev-token-signing-secret-0000"
      abort "explicit secret ignored" if ARGV[0] == "explicit" && value != ENV.fetch("MATRIXHUB_TOKEN_SIGNING_SECRET")
    end
  ' "${mode}"
done

for compose in deploy/docker-compose.yml deploy/docker-compose.mysql.yml deploy/docker-compose.sqlite.yml; do
  docker compose -f "${compose}" config --format json | ruby -rjson -e '
    config = JSON.parse(STDIN.read)
    abort "compose signing secret missing" unless config.dig("services", "matrixhub", "environment", "MATRIXHUB_TOKEN_SIGNING_SECRET") == ENV.fetch("MATRIXHUB_TOKEN_SIGNING_SECRET")
  '
  env -u MATRIXHUB_TOKEN_SIGNING_SECRET docker compose -f "${compose}" config --quiet
done

printf 'Token signing deployment checks passed\n'