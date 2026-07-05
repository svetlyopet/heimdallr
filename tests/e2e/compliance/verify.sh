#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
STATE_DIR="${STATE_DIR:?STATE_DIR is required}"

APPLICATION_ID="$(cat "${STATE_DIR}/application_id")"
RELEASE_ID="$(cat "${STATE_DIR}/release_id")"
RELEASE_VERSION="$(cat "${STATE_DIR}/release_version")"
REPORT_ID="$(cat "${STATE_DIR}/report_id")"

auth=(-u "${HEIMDALLR_USER}:${HEIMDALLR_PASSWORD}")

release_json="$(curl -sf "${auth[@]}" "${HEIMDALLR_URL}/api/v1/application/${APPLICATION_ID}/release/${RELEASE_ID}")"
echo "${release_json}" | jq -e ".data.version == \"${RELEASE_VERSION}\"" >/dev/null
echo "${release_json}" | jq -e '.data.commit_sha == "abc123"' >/dev/null

report_json="$(curl -sf "${auth[@]}" "${HEIMDALLR_URL}/api/v1/application/${APPLICATION_ID}/release/${RELEASE_ID}/report/${REPORT_ID}")"
echo "${report_json}" | jq -e '.data.status == "success"' >/dev/null

list_json="$(curl -sf "${auth[@]}" "${HEIMDALLR_URL}/api/v1/report?status=success")"
echo "${list_json}" | jq -e --arg id "${REPORT_ID}" '.data[] | select(.id == $id)' >/dev/null

echo "Compliance E2E verification passed"
