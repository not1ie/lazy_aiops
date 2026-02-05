# Docker Swarm 部署

## 1. 构建镜像并推送

```bash
docker build -t <REGISTRY>/lazy-aiops:latest -f Dockerfile .
docker push <REGISTRY>/lazy-aiops:latest
```

## 2. 修改 stack 镜像

编辑 `deploy/swarm/stack.yml`：

```yaml
image: <REGISTRY>/lazy-aiops:latest
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
