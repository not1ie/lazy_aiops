#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-}"

if [[ -z "${PASSWORD}" ]]; then
  echo "ERROR: PASSWORD is required. Example: PASSWORD='your-pass' bash scripts/verify_jump_p0.sh"
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required"
  exit 1
fi

RID="p0-$(date +%s)-$$"
created_asset="false"
asset_id=""
account_id=""
policy_id=""
rule_id=""
session_id=""
session2_id=""

cleanup() {
  if [[ -n "${policy_id}" ]]; then
    curl -sS -X DELETE "${BASE_URL}/api/v1/jump/policies/${policy_id}" -H "Authorization: Bearer ${TOKEN}" >/dev/null || true
  fi
  if [[ -n "${account_id}" ]]; then
    curl -sS -X DELETE "${BASE_URL}/api/v1/jump/accounts/${account_id}" -H "Authorization: Bearer ${TOKEN}" >/dev/null || true
  fi
  if [[ -n "${rule_id}" ]]; then
    curl -sS -X DELETE "${BASE_URL}/api/v1/jump/command-rules/${rule_id}" -H "Authorization: Bearer ${TOKEN}" >/dev/null || true
  fi
  if [[ "${created_asset}" == "true" && -n "${asset_id}" ]]; then
    curl -sS -X DELETE "${BASE_URL}/api/v1/jump/assets/${asset_id}" -H "Authorization: Bearer ${TOKEN}" >/dev/null || true
  fi
}
trap cleanup EXIT

echo "[1/10] login"
login_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}"
)"
TOKEN="$(echo "${login_rsp}" | jq -r '.data.token // empty')"
if [[ -z "${TOKEN}" ]]; then
  echo "ERROR: login failed: ${login_rsp}"
  exit 1
fi

echo "[2/10] check jump plugin loaded"
plugins_rsp="$(curl -sS "${BASE_URL}/api/v1/plugins" -H "Authorization: Bearer ${TOKEN}")"
if ! echo "${plugins_rsp}" | jq -e '.data.loaded[]?.name == "jump"' >/dev/null; then
  echo "ERROR: jump plugin not loaded"
  exit 1
fi

echo "[3/10] get base data"
user_info="$(curl -sS "${BASE_URL}/api/v1/user/info" -H "Authorization: Bearer ${TOKEN}")"
admin_user_id="$(echo "${user_info}" | jq -r '.data.id // empty')"
if [[ -z "${admin_user_id}" ]]; then
  echo "ERROR: cannot resolve current user id"
  exit 1
fi

assets_rsp="$(curl -sS "${BASE_URL}/api/v1/jump/assets" -H "Authorization: Bearer ${TOKEN}")"
asset_id="$(echo "${assets_rsp}" | jq -r '.data[]? | select(.enabled==true) | .id' | head -n1)"
if [[ -z "${asset_id}" ]]; then
  create_asset_rsp="$(
    curl -sS -X POST "${BASE_URL}/api/v1/jump/assets" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H 'Content-Type: application/json' \
      -d "{\"name\":\"verify-${RID}\",\"asset_type\":\"host\",\"protocol\":\"ssh\",\"address\":\"127.0.0.1\",\"port\":22,\"source\":\"manual\",\"enabled\":true}"
  )"
  asset_id="$(echo "${create_asset_rsp}" | jq -r '.data.id // empty')"
  if [[ -z "${asset_id}" ]]; then
    echo "ERROR: create asset failed: ${create_asset_rsp}"
    exit 1
  fi
  created_asset="true"
fi

echo "[4/10] create account & policy"
account_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/accounts" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"verify-account-${RID}\",\"username\":\"root\",\"auth_type\":\"password\",\"secret\":\"verify-pass-${RID}\",\"enabled\":true}"
)"
account_id="$(echo "${account_rsp}" | jq -r '.data.id // empty')"
if [[ -z "${account_id}" ]]; then
  echo "ERROR: create account failed: ${account_rsp}"
  exit 1
fi

policy_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/policies" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"verify-policy-${RID}\",\"user_id\":\"${admin_user_id}\",\"asset_id\":\"${asset_id}\",\"account_id\":\"${account_id}\",\"protocol\":\"ssh\",\"time_window_start\":\"00:00\",\"time_window_end\":\"23:59\",\"max_duration_sec\":1800,\"concurrent_limit\":2,\"status\":1}"
)"
policy_id="$(echo "${policy_rsp}" | jq -r '.data.id // empty')"
if [[ -z "${policy_id}" ]]; then
  echo "ERROR: create policy failed: ${policy_rsp}"
  exit 1
fi

echo "[5/10] start session"
start_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/sessions/start" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "{\"asset_id\":\"${asset_id}\",\"account_id\":\"${account_id}\",\"protocol\":\"ssh\"}"
)"
session_id="$(echo "${start_rsp}" | jq -r '.data.session.id // empty')"
if [[ -z "${session_id}" ]]; then
  echo "ERROR: start session failed: ${start_rsp}"
  exit 1
fi

echo "[6/10] record command"
record_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/sessions/${session_id}/commands" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d '{"command":"echo jump-p0-ok","result_code":0,"output_snippet":"jump-p0-ok"}'
)"
if ! echo "${record_rsp}" | jq -e '.code == 0' >/dev/null; then
  echo "ERROR: record command failed: ${record_rsp}"
  exit 1
fi

echo "[7/10] create blocking rule and trigger risk"
rule_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/command-rules" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"verify-block-${RID}\",\"pattern\":\"__JUMP_P0_BLOCK__\",\"match_type\":\"contains\",\"rule_kind\":\"risk\",\"severity\":\"critical\",\"action\":\"block\",\"enabled\":true,\"priority\":999}"
)"
rule_id="$(echo "${rule_rsp}" | jq -r '.data.id // empty')"
if [[ -z "${rule_id}" ]]; then
  echo "ERROR: create rule failed: ${rule_rsp}"
  exit 1
fi

block_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/sessions/${session_id}/commands" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d '{"command":"echo __JUMP_P0_BLOCK__","result_code":0}'
)"
if ! echo "${block_rsp}" | jq -e '.code == 0 and .data.blocked == true' >/dev/null; then
  echo "ERROR: block rule not effective: ${block_rsp}"
  exit 1
fi

echo "[8/10] verify risk events and command audit"
risk_rsp="$(curl -sS "${BASE_URL}/api/v1/jump/risk-events?session_id=${session_id}" -H "Authorization: Bearer ${TOKEN}")"
if ! echo "${risk_rsp}" | jq -e '.code == 0 and (.data | length) >= 1' >/dev/null; then
  echo "ERROR: risk events missing: ${risk_rsp}"
  exit 1
fi

cmd_rsp="$(curl -sS "${BASE_URL}/api/v1/jump/sessions/${session_id}/commands?limit=5" -H "Authorization: Bearer ${TOKEN}")"
if ! echo "${cmd_rsp}" | jq -e '.code == 0 and (.data | length) >= 1' >/dev/null; then
  echo "ERROR: command audit missing: ${cmd_rsp}"
  exit 1
fi

echo "[9/10] verify force disconnect"
start2_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/sessions/start" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d "{\"asset_id\":\"${asset_id}\",\"account_id\":\"${account_id}\",\"protocol\":\"ssh\"}"
)"
session2_id="$(echo "${start2_rsp}" | jq -r '.data.session.id // empty')"
if [[ -z "${session2_id}" ]]; then
  echo "ERROR: start second session failed: ${start2_rsp}"
  exit 1
fi

disconnect_rsp="$(
  curl -sS -X POST "${BASE_URL}/api/v1/jump/sessions/${session2_id}/disconnect" \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Content-Type: application/json' \
    -d '{"reason":"verify script force disconnect"}'
)"
if ! echo "${disconnect_rsp}" | jq -e '.code == 0 and (.data.status == "closed" or .data.status == "rejected")' >/dev/null; then
  echo "ERROR: force disconnect failed: ${disconnect_rsp}"
  exit 1
fi

echo "[10/10] done"
echo "PASS: jump P0 verification completed"
