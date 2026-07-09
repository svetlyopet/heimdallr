#!/usr/bin/env bash

set -euo pipefail

heimdallr_login() {
  curl -sf -X POST "${HEIMDALLR_URL}/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${HEIMDALLR_USER}\",\"password\":\"${HEIMDALLR_PASSWORD}\"}" \
    | jq -r '.data.token'
}

setup_auth_headers() {
  HEIMDALLR_TOKEN="$(heimdallr_login)"
  auth_headers=(
    -H "Authorization: Bearer ${HEIMDALLR_TOKEN}"
    -H "Content-Type: application/json"
  )
}
