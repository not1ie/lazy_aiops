#!/usr/bin/env bash
set -euo pipefail

# Runtime smoke checks for Lazy Auto Ops APIs.
# Usage:
#   BASE_URL=http://127.0.0.1:8080 TOKEN=<jwt> bash scripts/smoke_runtime_checks.sh
# Optional vars:
#   DOCKER_HOST_ID, CONTAINER_ID, SERVICE_ID
#   K8S_CLUSTER_ID, K8S_NAMESPACE, K8S_POD, K8S_CONTAINER

BASE_URL="${BASE_URL:-${1:-}}"
TOKEN="${TOKEN:-${2:-}}"

if [[ -z "${BASE_URL}" || -z "${TOKEN}" ]]; then
  cat <<'EOF'
Usage:
  BASE_URL=http://127.0.0.1:8080 TOKEN=<jwt> bash scripts/smoke_runtime_checks.sh

Optional:
  DOCKER_HOST_ID=<docker_host_uuid>
  CONTAINER_ID=<container_id_or_name>
  SERVICE_ID=<service_id_or_name>
  K8S_CLUSTER_ID=<cluster_id>
  K8S_NAMESPACE=<namespace>
  K8S_POD=<pod_name>
  K8S_CONTAINER=<container_name>
EOF
  exit 1
fi

pass=0
fail=0
skip=0

log_pass() { printf '[PASS] %s\n' "$1"; pass=$((pass + 1)); }
log_fail() { printf '[FAIL] %s\n' "$1"; fail=$((fail + 1)); }
log_skip() { printf '[SKIP] %s\n' "$1"; skip=$((skip + 1)); }

status_allowed() {
  local status="$1"
  local allowed_csv="$2"
  IFS=',' read -r -a codes <<<"$allowed_csv"
  for c in "${codes[@]}"; do
    if [[ "${status}" == "${c}" ]]; then
      return 0
    fi
  done
  return 1
}

urlenc() {
  python3 -c 'import sys,urllib.parse;print(urllib.parse.quote(sys.argv[1], safe=""))' "$1"
}

api_check() {
  local name="$1"
  local method="$2"
  local path="$3"
  local allowed="$4"
  local body="${5:-}"

  local url="${BASE_URL}${path}"
  local tmp
  tmp="$(mktemp)"

  local http_code
  if [[ -n "${body}" ]]; then
    http_code="$(curl -sS -m 25 -o "${tmp}" -w '%{http_code}' \
      -X "${method}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H 'Accept: application/json' \
      -H 'Content-Type: application/json' \
      --data "${body}" \
      "${url}" || true)"
  else
    http_code="$(curl -sS -m 25 -o "${tmp}" -w '%{http_code}' \
      -X "${method}" \
      -H "Authorization: Bearer ${TOKEN}" \
      -H 'Accept: application/json' \
      "${url}" || true)"
  fi

  if status_allowed "${http_code}" "${allowed}"; then
    log_pass "${name} -> ${method} ${path} [${http_code}]"
  else
    local preview
    preview="$(head -c 320 "${tmp}" | tr '\n' ' ' || true)"
    log_fail "${name} -> ${method} ${path} [${http_code}] body=${preview}"
  fi
  rm -f "${tmp}"
}

ws_upgrade_check() {
  local name="$1"
  local path="$2"
  local allowed="$3"
  local url="${BASE_URL}${path}"
  local code
  code="$(curl -sS -m 20 -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer ${TOKEN}" \
    -H 'Connection: Upgrade' \
    -H 'Upgrade: websocket' \
    -H 'Sec-WebSocket-Version: 13' \
    -H 'Sec-WebSocket-Key: SGVsbG9Db2RleA==' \
    "${url}" || true)"

  if status_allowed "${code}" "${allowed}"; then
    log_pass "${name} -> WS ${path} [${code}]"
  else
    log_fail "${name} -> WS ${path} [${code}]"
  fi
}

echo "== Core APIs =="
api_check "User Info" GET "/api/v1/user/info" "200"
api_check "CMDB Hosts" GET "/api/v1/cmdb/hosts" "200"
api_check "Docker Hosts" GET "/api/v1/docker/hosts" "200"
api_check "K8s Clusters" GET "/api/v1/k8s/clusters" "200"
api_check "Task List" GET "/api/v1/task/tasks" "200"
api_check "Monitor Metrics" GET "/api/v1/monitor/metrics" "200"
api_check "Monitor Metrics History" GET "/api/v1/monitor/metrics/history?hours=24" "200"
api_check "Monitor Alerts" GET "/api/v1/monitor/alerts" "200"
api_check "Monitor Servers" GET "/api/v1/monitor/servers" "200"
api_check "AI Sessions" GET "/api/v1/ai/sessions" "200"
api_check "AI Configs" GET "/api/v1/ai/configs" "200"
api_check "AI Analysis History" GET "/api/v1/ai/analyze/history" "200"
api_check "AI Create Session" POST "/api/v1/ai/sessions" "200" '{"title":"smoke-check"}'

if [[ -n "${DOCKER_HOST_ID:-}" ]]; then
  echo "== Docker Host APIs =="
  api_check "Docker Containers" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/containers" "200"
  api_check "Docker Container Stats" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/containers/stats" "200"
  api_check "Docker Services Detail" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/services?detail=1" "200"
  api_check "Docker Nodes" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/nodes" "200"
  api_check "Docker Registries" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/registries" "200"
  api_check "Docker Configs" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/configs" "200"
  api_check "Docker Secrets" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/secrets" "200"
  api_check "Docker Volumes" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/volumes" "200"
  api_check "Docker Stacks" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/stacks" "200"

  if [[ -n "${CONTAINER_ID:-}" ]]; then
    local_cid="$(urlenc "${CONTAINER_ID}")"
    api_check "Docker Container Logs" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/containers/${local_cid}/logs?tail=30" "200,400,500"
    api_check "Docker Container Exec" POST "/api/v1/docker/hosts/${DOCKER_HOST_ID}/containers/${local_cid}/exec" "200,400,500" '{"command":"echo smoke"}'
    ws_upgrade_check "Docker Container WebShell" "/api/v1/docker/hosts/${DOCKER_HOST_ID}/containers/${local_cid}/exec/ws?token=$(urlenc "${TOKEN}")&shell=%2Fbin%2Fsh" "101,400"
  else
    log_skip "Docker container-level checks (missing CONTAINER_ID)"
  fi

  if [[ -n "${SERVICE_ID:-}" ]]; then
    local_sid="$(urlenc "${SERVICE_ID}")"
    api_check "Docker Service Logs" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/services/${local_sid}/logs?tail=30" "200,400,500"
    api_check "Docker Service Tasks" GET "/api/v1/docker/hosts/${DOCKER_HOST_ID}/services/${local_sid}/tasks" "200,400,500"
  else
    log_skip "Docker service-level checks (missing SERVICE_ID)"
  fi
else
  log_skip "Docker host-level checks (missing DOCKER_HOST_ID)"
fi

if [[ -n "${K8S_CLUSTER_ID:-}" ]]; then
  echo "== K8s APIs =="
  api_check "K8s Namespaces" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces" "200"
  api_check "K8s Nodes" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/nodes" "200"
  api_check "K8s Workloads" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/workloads" "200"
  api_check "K8s Services" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/services" "200"
  api_check "K8s Ingresses" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/ingresses" "200"
  api_check "K8s Events" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/events" "200"

  if [[ -n "${K8S_NAMESPACE:-}" ]]; then
    ns_enc="$(urlenc "${K8S_NAMESPACE}")"
    api_check "K8s Namespace Pods" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces/${ns_enc}/pods" "200"
    api_check "K8s Deployments" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces/${ns_enc}/deployments" "200"

    if [[ -n "${K8S_POD:-}" ]]; then
      pod_enc="$(urlenc "${K8S_POD}")"
      api_check "K8s Pod Detail" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces/${ns_enc}/pods/${pod_enc}" "200"
      api_check "K8s Pod Logs" GET "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces/${ns_enc}/pods/${pod_enc}/logs?tail=30" "200,400,500"

      if [[ -n "${K8S_CONTAINER:-}" ]]; then
        container_enc="$(urlenc "${K8S_CONTAINER}")"
        ws_upgrade_check "K8s WebShell" "/api/v1/k8s/clusters/${K8S_CLUSTER_ID}/namespaces/${ns_enc}/pods/${pod_enc}/exec?container=${container_enc}&token=$(urlenc "${TOKEN}")&command=%2Fbin%2Fsh" "101,400"
      else
        log_skip "K8s WebShell check (missing K8S_CONTAINER)"
      fi
    else
      log_skip "K8s pod-level checks (missing K8S_POD)"
    fi
  else
    log_skip "K8s namespace-level checks (missing K8S_NAMESPACE)"
  fi
else
  log_skip "K8s checks (missing K8S_CLUSTER_ID)"
fi

echo
echo "Summary: pass=${pass} fail=${fail} skip=${skip}"
if [[ "${fail}" -gt 0 ]]; then
  exit 2
fi

