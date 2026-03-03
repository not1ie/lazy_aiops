# Docker Swarm 部署

## 1. 构建镜像并推送

```bash
docker build -t <REGISTRY>/lazy-aiops:latest -f Dockerfile .
docker push <REGISTRY>/lazy-aiops:latest
```

## 2. 通过环境变量注入镜像

`deploy/swarm/stack.yml` 支持环境变量：

```bash
LAZY_AIOPS_IMAGE=<REGISTRY>/lazy-aiops:latest docker stack deploy -c deploy/swarm/stack.yml lazy-aiops
```

## 3. 初始化 Swarm（如未初始化）

```bash
docker swarm init
```

## 4. 部署

```bash
docker stack deploy -c deploy/swarm/stack.yml lazy-aiops
```

## 5. 验证

```bash
docker service ls
docker service ps lazy-aiops_lazy-auto-ops
```

## 可选：一键发布脚本

```bash
IMAGE_REPO=<REGISTRY>/lazy-aiops bash scripts/release_swarm.sh
```

脚本会自动按 `commit` 打标签并优先使用 `@sha256` digest 发布，避免 `latest` 漂移。
