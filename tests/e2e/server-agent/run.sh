#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SERVER_AGENT_DIR="${ROOT_DIR}/tests/e2e/server-agent"

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%s)-$$}"

export HEIMDALLR_URL HEIMDALLR_USER HEIMDALLR_PASSWORD E2E_RUN_ID

"${ROOT_DIR}/scripts/wait-for-health.sh"
"${SERVER_AGENT_DIR}/flow.sh"
