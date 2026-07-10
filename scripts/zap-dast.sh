#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
OPENAPI_SPEC="${OPENAPI_SPEC:-api/docs/openapi.yaml}"
ZAP_WORKDIR="${ZAP_WORKDIR:-/tmp/heimdallr-zap}"
ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"
ZAP_FAIL_ON_WARN="${ZAP_FAIL_ON_WARN:-false}"

mkdir -p "${ZAP_WORKDIR}"
chmod a+rwx "${ZAP_WORKDIR}"
rm -f "${ZAP_WORKDIR}/zap-report.html" "${ZAP_WORKDIR}/zap-report.json"

echo "Waiting for Heimdallr at ${HEIMDALLR_URL}..."
"${BASH_SOURCE%/*}/wait-for-health.sh"

echo "Authenticating as ${HEIMDALLR_USER}..."
TOKEN="$(
  curl -sf -X POST "${HEIMDALLR_URL}/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"${HEIMDALLR_USER}\",\"password\":\"${HEIMDALLR_PASSWORD}\"}" \
    | jq -r '.data.token'
)"

if [ -z "${TOKEN}" ] || [ "${TOKEN}" = "null" ]; then
  echo "Failed to obtain auth token" >&2
  exit 1
fi

OPENAPI_ZAP="${ZAP_WORKDIR}/openapi-zap.yaml"
sed "s|url: /api|url: ${HEIMDALLR_URL}/api|" "${OPENAPI_SPEC}" > "${OPENAPI_ZAP}"

ZAP_OPTS=(
  -t "/zap/wrk/openapi-zap.yaml"
  -f openapi
  -r "/zap/wrk/zap-report.html"
  -J "/zap/wrk/zap-report.json"
  -z "config replacer.full_list(0).description=auth"
  -z "config replacer.full_list(0).enabled=true"
  -z "config replacer.full_list(0).matchtype=REQ_HEADER"
  -z "config replacer.full_list(0).matchstr=Authorization"
  -z "config replacer.full_list(0).regex=false"
  -z "config replacer.full_list(0).replacement=Bearer ${TOKEN}"
)

if [ "${ZAP_FAIL_ON_WARN}" != "true" ]; then
  ZAP_OPTS+=(-I)
fi

echo "Running OWASP ZAP API scan..."
docker run --rm --network host \
  -v "${ZAP_WORKDIR}:/zap/wrk:rw" \
  "${ZAP_IMAGE}" \
  zap-api-scan.py "${ZAP_OPTS[@]}"

echo "ZAP reports written to ${ZAP_WORKDIR}/zap-report.html"
