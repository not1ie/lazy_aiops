# Web 访问指南

## 🎉 好消息！Web UI 已经添加完成

我已经为 Lazy Auto Ops 添加了完整的 Web 用户界面，现在你可以通过浏览器方便地使用系统了！

## 📋 当前状态

### ✅ 已完成的工作

1. **Web UI 实现**
   - 登录页面（美观的渐变设计）
   - 仪表板（显示系统信息和已加载插件）
   - 插件管理页面
   - 响应式设计，支持移动端

2. **后端更新**
   - 添加了静态文件服务
   - 更新了 Dockerfile 包含 web 目录
   - 使用 Debian 基础镜像解决 SQLite3 编译问题

3. **文件结构**
   ```
   web/
   └── static/
       ├── index.html      # 主页面
       ├── css/
       │   └── style.css   # 样式文件
       └── js/
           └── app.js      # 前端逻辑
   ```

### ⚠️ 需要重新部署

由于代码已更新，需要重新部署才能看到 Web 界面。

## 🚀 重新部署步骤

### 方法 1: 使用自动部署脚本（推荐）

```bash
cd lazy-auto-ops
chmod +x quick-update.sh
./quick-update.sh
```

这个脚本会：
1. 打包最新代码
2. 上传到服务器
3. 重新构建 Docker 镜像
4. 启动服务

### 方法 2: 手动部署

```bash
# 1. 打包
tar czf lazy-auto-ops.tar.gz \
    --exclude='*.tar.gz' \
    --exclude='.git' \
    cmd/ configs/ internal/ plugins/ web/ go.mod go.sum Dockerfile deploy/

# 2. 上传到服务器
scp lazy-auto-ops.tar.gz root@192.168.10.100:/root/lazy-auto-ops/

# 3. SSH 到服务器
ssh root@192.168.10.100

# 4. 解压并部署
cd /root/lazy-auto-ops
tar xzf lazy-auto-ops.tar.gz
cd deploy/docker
docker compose down
docker compose build
docker compose up -d
```

## 🌐 访问 Web 界面

部署完成后，在浏览器中访问：

```
http://192.168.10.100:8080
```

### 默认登录信息

- **用户名**: `admin`
- **密码**: `admin123`

## 📱 Web UI 功能

### 1. 登录页面
- 美观的渐变背景设计
- 简洁的登录表单
- 错误提示

### 2. 仪表板
- 系统版本信息
- 已加载插件数量
- 系统运行状态
- 插件列表展示

### 3. 插件管理
- 查看已加载的插件
- 查看可用的插件
- 插件详细信息

### 4. 用户体验
- 响应式设计，支持手机和平板
- 现代化的 UI 设计
- 流畅的页面切换
- JWT Token 自动管理

## 🔧 技术实现

### 前端
- 纯 HTML/CSS/JavaScript（无框架依赖）
- 现代化的渐变设计
- LocalStorage 存储 Token
- Fetch API 调用后端

### 后端
- Gin 框架静态文件服务
- JWT 认证
- CORS 支持
- RESTful API

## 📊 API 端点

Web UI 使用以下 API：

- `GET /` - 首页
- `GET /static/*` - 静态资源
- `POST /api/v1/login` - 登录
- `GET /api/v1/user/info` - 用户信息
- `GET /api/v1/system/info` - 系统信息
- `GET /api/v1/plugins` - 插件列表

## 🐛 故障排查

### 问题：无法访问 Web 界面

1. **检查服务状态**
   ```bash
   ssh root@192.168.10.100
   cd /root/lazy-auto-ops/deploy/docker
   docker compose ps
   ```

2. **查看日志**
   ```bash
   docker compose logs -f lazy-auto-ops
   ```

3. **检查端口**
   ```bash
   curl http://192.168.10.100:8080/health
   ```

### 问题：登录失败

- 确认使用正确的用户名和密码（admin/admin123）
- 检查浏览器控制台是否有错误
- 查看服务器日志

### 问题：页面显示不正常

- 清除浏览器缓存
- 检查静态文件是否正确加载
- 查看浏览器控制台的网络请求

## 📝 下一步计划

可以考虑添加：

1. **更多功能页面**
   - CMDB 主机管理界面
   - 任务调度界面
   - 监控告警界面

2. **用户管理**
   - 用户列表
   - 权限管理
   - 密码修改

3. **系统设置**
   - 插件配置
   - 系统参数
   - 日志查看

## 💡 提示

- Web UI 使用 JWT Token 进行认证，Token 存储在浏览器的 LocalStorage 中
- 退出登录会清除 Token
- 如果 Token 过期，会自动跳转到登录页面
- 所有 API 请求都需要在 Header 中携带 Token（除了登录和系统信息接口）

---

**享受使用 Lazy Auto Ops 的 Web 界面吧！** 🎉
