# Kubernetes 部署

## 1. 构建并推送镜像

```bash
# 构建镜像
docker build -t <REGISTRY>/lazy-aiops:latest -f Dockerfile .

# 推送镜像
docker push <REGISTRY>/lazy-aiops:latest
```

## 2. 修改镜像地址

更新 `deploy/k8s/deployment.yaml` 中的镜像地址：

```yaml
image: <REGISTRY>/lazy-aiops:latest
```

## 3. 应用部署

```bash
kubectl apply -k deploy/k8s
```

## 4. 可选：启用 Ingress

如果集群已安装 Ingress Controller：

```bash
kubectl apply -f deploy/k8s/ingress.yaml
```

## 5. 验证

```bash
kubectl -n lazy-aiops get pods
kubectl -n lazy-aiops get svc

# 本地端口转发
kubectl -n lazy-aiops port-forward svc/lazy-auto-ops 8080:80
```

访问：`http://localhost:8080`
