#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
STATE_DIR="${STATE_DIR:?STATE_DIR is required}"
RELEASE_VERSION="${RELEASE_VERSION:-v1.0.0-e2e}"
REPORT_ID="${REPORT_ID:-sast-e2e-1}"

APPLICATION_ID="$(cat "${STATE_DIR}/application_id")"
TOKEN="$(cat "${STATE_DIR}/token")"

auth=(-H "Authorization: Bearer ${TOKEN}")

release_json="$(curl -sf "${auth[@]}" -X POST \
  -H "Content-Type: application/json" \
  "${HEIMDALLR_URL}/api/v1/application/${APPLICATION_ID}/release?upsert=true" \
  -d "{\"version\":\"${RELEASE_VERSION}\",\"commit_sha\":\"abc123\",\"branch\":\"main\",\"pipeline_url\":\"https://example.com/pipeline/e2e\"}")"
echo "${release_json}" > "${STATE_DIR}/release.json"

RELEASE_ID="$(jq -r '.data.id' "${STATE_DIR}/release.json")"
echo "${RELEASE_ID}" > "${STATE_DIR}/release_id"
echo "${RELEASE_VERSION}" > "${STATE_DIR}/release_version"
echo "${REPORT_ID}" > "${STATE_DIR}/report_id"

curl -sf "${auth[@]}" -X POST \
  -H "Content-Type: application/json" \
  "${HEIMDALLR_URL}/api/v1/application/${APPLICATION_ID}/release/${RELEASE_ID}/report" \
  -d "{\"id\":\"${REPORT_ID}\",\"type\":\"sast\",\"status\":\"started\",\"location\":\"ci\",\"url\":\"https://example.com/run/e2e\",\"metadata\":{\"tool\":\"e2e\"}}"

echo "<h1>E2E SAST report</h1>" > "${STATE_DIR}/report.html"
OUTPUT_B64="$(base64 < "${STATE_DIR}/report.html" | tr -d '\n')"

curl -sf "${auth[@]}" -X PATCH \
  -H "Content-Type: application/json" \
  "${HEIMDALLR_URL}/api/v1/application/${APPLICATION_ID}/release/${RELEASE_ID}/report/${REPORT_ID}" \
  -d "{\"status\":\"success\",\"metadata\":{\"findings\":0,\"tool\":\"e2e\"},\"output\":\"${OUTPUT_B64}\"}"

echo "Pushed report ${REPORT_ID} for release ${RELEASE_VERSION}"
