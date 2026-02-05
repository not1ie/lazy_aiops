# 统一部署入口

该目录提供 **一套入口**，可选 **K8s** 或 **Docker Swarm** 部署。

## 快速开始

```bash
# K8s
REGISTRY_IMAGE=registry.example.com/lazy-aiops:latest deploy/scripts/deploy.sh k8s

# Docker Swarm
REGISTRY_IMAGE=registry.example.com/lazy-aiops:latest deploy/scripts/deploy.sh swarm
```

## 目录结构

- `deploy/k8s/` K8s 部署清单
- `deploy/swarm/` Docker Swarm 部署清单
- `deploy/scripts/deploy.sh` 统一部署脚本

## 说明

- 脚本会自动替换 `image` 字段。
- 若需要更复杂的参数管理，可使用 Kustomize 或 Helm（后续可扩展）。
