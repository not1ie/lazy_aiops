#!/usr/bin/env bash
set -euo pipefail

MODE=${1:-}
REGISTRY_IMAGE=${REGISTRY_IMAGE:-lazy-aiops:latest}
NAMESPACE=${NAMESPACE:-lazy-aiops}
K8S_CONTEXT=${K8S_CONTEXT:-}

usage() {
  cat <<USAGE
Usage:
  deploy/scripts/deploy.sh <k8s|swarm>

Env:
  REGISTRY_IMAGE   Image to deploy (default: lazy-aiops:latest)
  NAMESPACE        K8s namespace (default: lazy-aiops)
  K8S_CONTEXT      Optional kubectl context

Examples:
  REGISTRY_IMAGE=registry.example.com/lazy-aiops:latest deploy/scripts/deploy.sh k8s
  REGISTRY_IMAGE=registry.example.com/lazy-aiops:latest deploy/scripts/deploy.sh swarm
USAGE
}

if [ -z "$MODE" ]; then
  usage
  exit 1
fi

if [ "$MODE" = "k8s" ]; then
  if [ -n "$K8S_CONTEXT" ]; then
    kubectl config use-context "$K8S_CONTEXT"
  fi
  # update image
  sed -i.bak "s|image: .*lazy-aiops:latest|image: ${REGISTRY_IMAGE}|" deploy/k8s/deployment.yaml
  rm -f deploy/k8s/deployment.yaml.bak

  # apply manifests
  kubectl apply -k deploy/k8s
  echo "K8s deployed. Namespace: ${NAMESPACE}"
  exit 0
fi

if [ "$MODE" = "swarm" ]; then
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker not found. Installing..."
    bash <(curl -sSL https://linuxmirrors.cn/docker.sh)
  fi
  # update image
  sed -i.bak "s|image: .*lazy-aiops:latest|image: ${REGISTRY_IMAGE}|" deploy/swarm/stack.yml
  rm -f deploy/swarm/stack.yml.bak

  if ! docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null | grep -qiE 'active|pending'; then
    docker swarm init >/dev/null 2>&1 || true
  fi
  docker stack deploy -c deploy/swarm/stack.yml lazy-aiops
  echo "Swarm deployed: stack lazy-aiops"
  exit 0
fi

usage
exit 1
