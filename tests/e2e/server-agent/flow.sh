#!/usr/bin/env bash
set -euo pipefail

HEIMDALLR_URL="${HEIMDALLR_URL:-http://localhost:8080}"
HEIMDALLR_USER="${HEIMDALLR_USER:-root}"
HEIMDALLR_PASSWORD="${HEIMDALLR_PASSWORD:-e2e-test-password}"
E2E_RUN_ID="${E2E_RUN_ID:-$(date +%s)-$$}"
HOSTNAME="e2e-server-${E2E_RUN_ID}.example.com"

auth_headers=(
  -H "X-Auth-Username: ${HEIMDALLR_USER}"
  -H "X-Auth-Password: ${HEIMDALLR_PASSWORD}"
  -H "Content-Type: application/json"
)

echo "Step 1: create orphan agent (datadog)"
orphan_json="$(curl -sf -w "\n%{http_code}" "${auth_headers[@]}" -X POST \
  "${HEIMDALLR_URL}/api/v1/agent" \
  -d '{"name":"datadog","type":"monitoring","version":"7.0.0"}')"
orphan_body="$(echo "${orphan_json}" | sed '$d')"
orphan_status="$(echo "${orphan_json}" | tail -n1)"
[[ "${orphan_status}" == "201" ]] || { echo "expected 201 creating orphan agent, got ${orphan_status}" >&2; exit 1; }
echo "${orphan_body}" | jq -e 'if (.data | has("server_id")) then .data.server_id == null else true end' >/dev/null
orphan_id="$(echo "${orphan_body}" | jq -r '.data.id')"
echo "  orphan agent id: ${orphan_id}"

echo "Step 2: create server with orphan + inline agent"
server_json="$(curl -sf -w "\n%{http_code}" "${auth_headers[@]}" -X POST \
  "${HEIMDALLR_URL}/api/v1/server" \
  -d "$(jq -nc \
    --arg hostname "${HOSTNAME}" \
    --arg orphan_id "${orphan_id}" \
    '{
      hostname: $hostname,
      operating_system: "linux",
      hypervisor: "kvm",
      location: "dc1",
      agent_ids: [$orphan_id],
      agents: [{name: "crowdstrike", type: "security", version: "1.0.0"}]
    }')")"
server_body="$(echo "${server_json}" | sed '$d')"
server_status="$(echo "${server_json}" | tail -n1)"
[[ "${server_status}" == "201" ]] || { echo "expected 201 creating server, got ${server_status}" >&2; exit 1; }
server_id="$(echo "${server_body}" | jq -r '.data.id')"
echo "  server id: ${server_id}"

echo "Step 3: verify server agent_count == 2"
curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/server/${server_id}" \
  | jq -e '.data.relations.agent_count == 2' >/dev/null

echo "Step 4: verify no unassigned agents"
curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/agent?unassigned=true" \
  | jq -e '(.data | length) == 0' >/dev/null

echo "Step 5: create second orphan agent (sentinel)"
second_orphan_json="$(curl -sf -w "\n%{http_code}" "${auth_headers[@]}" -X POST \
  "${HEIMDALLR_URL}/api/v1/agent" \
  -d '{"name":"sentinel","type":"security","version":"2.0.0"}')"
second_orphan_body="$(echo "${second_orphan_json}" | sed '$d')"
second_orphan_status="$(echo "${second_orphan_json}" | tail -n1)"
[[ "${second_orphan_status}" == "201" ]] || { echo "expected 201 creating second orphan, got ${second_orphan_status}" >&2; exit 1; }
second_orphan_id="$(echo "${second_orphan_body}" | jq -r '.data.id')"
echo "  second orphan id: ${second_orphan_id}"

echo "Step 6: attach second orphan via PUT server"
update_json="$(curl -sf -w "\n%{http_code}" "${auth_headers[@]}" -X PUT \
  "${HEIMDALLR_URL}/api/v1/server/${server_id}" \
  -d "$(jq -nc --arg id "${second_orphan_id}" '{agent_ids: [$id]}')")"
update_body="$(echo "${update_json}" | sed '$d')"
update_status="$(echo "${update_json}" | tail -n1)"
[[ "${update_status}" == "200" ]] || { echo "expected 200 updating server, got ${update_status}" >&2; exit 1; }
echo "${update_body}" | jq -e '.data.relations.agent_count == 3' >/dev/null

echo "Step 7: verify nested agent shows server hostname"
curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/server/${server_id}/agent/${orphan_id}" \
  | jq -e --arg hostname "${HOSTNAME}" '.data.server == $hostname' >/dev/null

echo "Step 8: verify global agent endpoint"
curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/agent/${orphan_id}" \
  | jq -e '.data.name == "datadog"' >/dev/null

echo "Step 9: delete nested agent"
delete_status="$(curl -sf -o /dev/null -w "%{http_code}" "${auth_headers[@]}" -X DELETE \
  "${HEIMDALLR_URL}/api/v1/server/${server_id}/agent/${orphan_id}")"
[[ "${delete_status}" == "204" ]] || { echo "expected 204 deleting agent, got ${delete_status}" >&2; exit 1; }

echo "Step 10: verify server agent_count == 2 after delete"
curl -sf "${auth_headers[@]}" "${HEIMDALLR_URL}/api/v1/server/${server_id}" \
  | jq -e '.data.relations.agent_count == 2' >/dev/null

echo "Server-agent E2E verification passed"
