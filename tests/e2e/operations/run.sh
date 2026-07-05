#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
PLAYBOOK_DIR="${ROOT_DIR}/tests/e2e/operations"

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
JOB_ID_SUCCESS="${JOB_ID_SUCCESS:-1000}"
JOB_ID_FAILURE="${JOB_ID_FAILURE:-1001}"

export HEIMDALLR_URL HEIMDALLR_USER HEIMDALLR_PASSWORD

"${ROOT_DIR}/scripts/wait-for-health.sh"

ansible-playbook "${PLAYBOOK_DIR}/seed.yaml"
ansible-playbook "${PLAYBOOK_DIR}/job-success.yaml" -e "job_id=${JOB_ID_SUCCESS}"
ansible-playbook "${PLAYBOOK_DIR}/job-failure.yaml" -e "job_id=${JOB_ID_FAILURE}"

AUTOMATION_ID="$(cat "${PLAYBOOK_DIR}/.automation_id")"
export AUTOMATION_ID JOB_ID_SUCCESS JOB_ID_FAILURE
"${PLAYBOOK_DIR}/verify.sh"
