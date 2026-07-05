#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
AUTOMATION_ID="${AUTOMATION_ID:?AUTOMATION_ID is required}"
JOB_ID_SUCCESS="${JOB_ID_SUCCESS:-1000}"
JOB_ID_FAILURE="${JOB_ID_FAILURE:-1001}"

auth_headers=(
  -H "X-Auth-Username: ${HEIMDALLR_USER}"
  -H "X-Auth-Password: ${HEIMDALLR_PASSWORD}"
)

success_json="$(curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/automation/${AUTOMATION_ID}/job/${JOB_ID_SUCCESS}")"
failure_json="$(curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/automation/${AUTOMATION_ID}/job/${JOB_ID_FAILURE}")"

echo "${success_json}" | jq -e '.data.status == "success"' >/dev/null
echo "${success_json}" | jq -e '.data.output != ""' >/dev/null
echo "${failure_json}" | jq -e '.data.status == "failed"' >/dev/null

echo "Operations E2E verification passed"
