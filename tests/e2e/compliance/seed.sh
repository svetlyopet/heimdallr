#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
STATE_DIR="${STATE_DIR:-$(mktemp -d)}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%s)-$$}"
APPLICATION_NAME="e2e-compliance-app-${E2E_RUN_ID}"
TOKEN_NAME="e2e-compliance-token-${E2E_RUN_ID}"

auth_headers=(
  -H "X-Auth-Username: ${HEIMDALLR_USER}"
  -H "X-Auth-Password: ${HEIMDALLR_PASSWORD}"
)

app_json="$(curl -sf "${auth_headers[@]}" -X POST \
  -H "Content-Type: application/json" \
  "${HEIMDALLR_URL}/api/v1/application" \
  -d "{\"name\":\"${APPLICATION_NAME}\",\"description\":\"e2e\",\"repository_url\":\"https://example.com/e2e\"}")"
echo "${app_json}" > "${STATE_DIR}/application.json"

APPLICATION_ID="$(jq -r '.data.id' "${STATE_DIR}/application.json")"

token_json="$(curl -sf "${auth_headers[@]}" -X POST \
  -H "Content-Type: application/json" \
  "${HEIMDALLR_URL}/api/v1/auth/tokens" \
  -d "{\"name\":\"${TOKEN_NAME}\",\"scopes\":[\"application:write\"]}")"
echo "${token_json}" > "${STATE_DIR}/token.json"

TOKEN="$(jq -r '.data.token' "${STATE_DIR}/token.json")"
echo "${APPLICATION_ID}" > "${STATE_DIR}/application_id"
echo "${TOKEN}" > "${STATE_DIR}/token"

echo "Seeded application ${APPLICATION_ID} and API token"
