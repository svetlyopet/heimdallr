#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-60}"
SLEEP_SECONDS="${SLEEP_SECONDS:-2}"

attempt=1
while [ "$attempt" -le "$MAX_ATTEMPTS" ]; do
  if curl -sf "${HEIMDALLR_URL}/api/health" >/dev/null; then
    echo "Heimdallr is healthy at ${HEIMDALLR_URL}"
    exit 0
  fi

  echo "Waiting for Heimdallr (${attempt}/${MAX_ATTEMPTS})..."
  attempt=$((attempt + 1))
  sleep "$SLEEP_SECONDS"
done

echo "Heimdallr did not become healthy in time" >&2
exit 1
