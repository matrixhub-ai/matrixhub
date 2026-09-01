#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 1 ] || [ -z "$1" ]; then
  echo "usage: $0 IMAGE_TAG [DEPLOY_DIR]" >&2
  exit 2
fi

image_tag=$1
deploy_dir=${2:-.}
cd "${deploy_dir}"

env_source=/dev/null
if [ -f .env ]; then
  env_source=.env
fi

if grep -q '^MYSQL_ROOT_PASSWORD=' "${env_source}" &&
   grep -q '^MYSQL_PASSWORD=' "${env_source}"; then
  root_password_candidate=
  app_password_candidate=
elif [ -d data/mysql ] &&
     [ -n "$(find data/mysql -maxdepth 1 -mindepth 1 -print -quit)" ]; then
  # Before the demo workflow generated credentials, Compose used these defaults.
  root_password_candidate=password
  app_password_candidate=changeme
else
  root_password_candidate=$(openssl rand -hex 16)
  app_password_candidate=$(openssl rand -hex 16)
fi

env_tmp=$(mktemp .env.XXXXXX)
trap 'rm -f "${env_tmp}"' EXIT
chmod 600 "${env_tmp}"

awk \
  -v image_tag="${image_tag}" \
  -v root_password="${root_password_candidate}" \
  -v app_password="${app_password_candidate}" '
  /^MYSQL_ROOT_PASSWORD=/ {
    if (!root_seen) {
      print
      root_seen = 1
    }
    next
  }
  /^MYSQL_PASSWORD=/ {
    if (!app_seen) {
      print
      app_seen = 1
    }
    next
  }
  /^MATRIXHUB_IMAGE_TAG=/ {
    if (!tag_seen) {
      print "MATRIXHUB_IMAGE_TAG=" image_tag
      tag_seen = 1
    }
    next
  }
  { print }
  END {
    if (!root_seen) {
      print "MYSQL_ROOT_PASSWORD=" root_password
    }
    if (!app_seen) {
      print "MYSQL_PASSWORD=" app_password
    }
    if (!tag_seen) {
      print "MATRIXHUB_IMAGE_TAG=" image_tag
    }
  }
' "${env_source}" > "${env_tmp}"

mv "${env_tmp}" .env
trap - EXIT
