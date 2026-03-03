#!/usr/bin/env bash
set -euo pipefail

# Build + deploy helper for Docker Swarm with safer update behavior.
#
# Required:
#   IMAGE_REPO  e.g. registry.example.com/lazy-aiops
#
# Optional:
#   STACK_NAME=lazy-aiops
#   SERVICE_NAME=lazy-aiops_lazy-auto-ops
#   BUILD_ARGS="--build-arg NODE_IMAGE=... --build-arg GO_IMAGE=... --build-arg RUNTIME_IMAGE=..."
#   PUSH_LATEST=1
#
# Usage:
#   IMAGE_REPO=registry.example.com/lazy-aiops bash scripts/release_swarm.sh

if [[ -z "${IMAGE_REPO:-}" ]]; then
  echo "ERROR: IMAGE_REPO is required, e.g. registry.example.com/lazy-aiops"
  exit 1
fi

STACK_NAME="${STACK_NAME:-lazy-aiops}"
SERVICE_NAME="${SERVICE_NAME:-${STACK_NAME}_lazy-auto-ops}"
PUSH_LATEST="${PUSH_LATEST:-1}"
BUILD_ARGS="${BUILD_ARGS:-}"

COMMIT="$(git rev-parse --short=12 HEAD)"
TAGGED_IMAGE="${IMAGE_REPO}:${COMMIT}"
LATEST_IMAGE="${IMAGE_REPO}:latest"

echo "[1/5] Building image ${TAGGED_IMAGE}"
docker build \
  --label "org.opencontainers.image.revision=${COMMIT}" \
  -t "${TAGGED_IMAGE}" \
  -t "${LATEST_IMAGE}" \
  ${BUILD_ARGS} \
  -f Dockerfile .

echo "[2/5] Pushing ${TAGGED_IMAGE}"
docker push "${TAGGED_IMAGE}"
if [[ "${PUSH_LATEST}" == "1" ]]; then
  echo "[2/5] Pushing ${LATEST_IMAGE}"
  docker push "${LATEST_IMAGE}"
fi

DIGEST_REF="$(docker image inspect "${TAGGED_IMAGE}" --format '{{index .RepoDigests 0}}' 2>/dev/null || true)"
DEPLOY_IMAGE="${DIGEST_REF:-${TAGGED_IMAGE}}"

echo "[3/5] Deploying stack=${STACK_NAME} image=${DEPLOY_IMAGE}"
LAZY_AIOPS_IMAGE="${DEPLOY_IMAGE}" docker stack deploy -c deploy/swarm/stack.yml "${STACK_NAME}"

echo "[4/5] Forcing service rollout with stop-first order"
docker service update \
  --update-order stop-first \
  --with-registry-auth \
  --image "${DEPLOY_IMAGE}" \
  "${SERVICE_NAME}"

echo "[5/5] Verifying"
docker service ps "${SERVICE_NAME}" --no-trunc
docker service logs --tail 120 "${SERVICE_NAME}" || true
curl -fsS "http://127.0.0.1:8080/health" || true

echo "Done: commit=${COMMIT} image=${DEPLOY_IMAGE}"
