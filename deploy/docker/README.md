# Docker 部署

## 方式一：Docker Compose（推荐）

```bash
docker compose -f deploy/docker/docker-compose.yml up -d --build
```

默认会挂载：
- `deploy/docker/config.yaml` -> `/app/configs/config.yaml`
- `deploy/docker/data` -> `/app/data`

访问：`http://localhost:8080`

## 方式二：生产配置

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

生产配置示例在 `deploy/docker/config.prod.yaml`。

## 常见命令

```bash
# 查看日志
docker logs -f lazy-auto-ops

# 停止
docker compose -f deploy/docker/docker-compose.yml down
```
