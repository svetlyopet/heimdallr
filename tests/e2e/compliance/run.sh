#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
COMPLIANCE_DIR="${ROOT_DIR}/tests/e2e/compliance"

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%s)-$$}"
STATE_DIR="$(mktemp -d)"

export HEIMDALLR_URL HEIMDALLR_USER HEIMDALLR_PASSWORD STATE_DIR E2E_RUN_ID

cleanup() {
  rm -rf "${STATE_DIR}"
}
trap cleanup EXIT

"${ROOT_DIR}/scripts/wait-for-health.sh"
"${COMPLIANCE_DIR}/seed.sh"
"${COMPLIANCE_DIR}/push-report.sh"
"${COMPLIANCE_DIR}/verify.sh"
