#!/usr/bin/env bash
set -euo pipefail

# Unified verification entrypoint:
# 1) frontend-backend API contract check
# 2) backend compile
# 3) frontend compile
# 4) optional runtime smoke checks (when BASE_URL + TOKEN are provided)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "[1/4] Verify API usage contract"
python3 scripts/verify_frontend_backend_api.py

echo "[2/4] Build backend"
GO_CACHE_DIR="${TMPDIR:-/tmp}/lazy_aiops_go_build_cache"
mkdir -p "${GO_CACHE_DIR}"
GOCACHE="${GO_CACHE_DIR}" go build ./...

echo "[3/4] Build frontend"
(
  cd frontend
  npm run build
)

echo "[4/4] Runtime smoke (optional)"
if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  bash scripts/smoke_runtime_checks.sh
else
  echo "SKIP: BASE_URL or TOKEN not set, runtime smoke not executed."
fi

echo "All checks completed."
