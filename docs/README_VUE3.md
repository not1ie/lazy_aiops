# 🚀 Lazy Auto Ops - Vue3版本

> AI驱动的轻量级运维平台 - 现代化Vue3前端

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Vue](https://img.shields.io/badge/vue-3.5.13-green.svg)](https://vuejs.org/)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)

## ✨ 特性

- 🤖 **AI日志分析** - 智能分析日志，自动判断告警并诊断故障
- 🎨 **Apple风格UI** - 现代化深色主题，毛玻璃效果，流畅动画
- ⚡ **极速体验** - Vue3 + Vite，秒级启动，热重载
- 🔌 **插件化架构** - 21+插件，灵活扩展
- 📦 **5分钟部署** - 单一二进制，开箱即用
- 🌐 **完全开源** - MIT许可证，自由使用

## 🎯 核心功能

### 已实现 ✅
- ✅ 用户认证系统
- ✅ 仪表板概览
- ✅ CMDB资产管理
- ✅ AI日志分析界面
- ✅ 响应式布局

### 开发中 🔄
- 🔄 监控中心
- 🔄 告警管理
- 🔄 任务调度
- 🔄 工作流编排
- 🔄 日志中心

## 🚀 快速开始

### 方式一：一键演示（推荐）

```bash
./demo-vue3.sh
```

选择运行模式：
1. 开发模式 - 前后端分离，支持热重载
2. 生产模式 - 构建后运行

### 方式二：开发模式

#### 1. 启动后端
```bash
go run cmd/server/main.go
```

#### 2. 启动前端
```bash
cd web-vue
npm install
npm run dev
```

访问: http://localhost:5173

### 方式三：生产部署

```bash
# 构建前端
./build-vue.sh

# 构建后端
go build -o lazy-auto-ops cmd/server/main.go

# 运行
./lazy-auto-ops
```

访问: http://localhost:8080

### 方式四：Docker部署

```bash
# 构建镜像
./docker-build-vue3.sh

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  --name lazy-auto-ops \
  lazy-auto-ops:vue3-latest
```

访问: http://localhost:8080

## 🔐 默认账号

- **用户名**: `admin`
- **密码**: `admin123`

## 📁 项目结构

```
lazy-auto-ops/
├── web-vue/              # Vue3前端项目 ⭐
│   ├── src/
│   │   ├── api/          # API接口
│   │   ├── components/   # Apple风格组件库
│   │   ├── layouts/      # 布局组件
│   │   ├── router/       # 路由配置
│   │   ├── stores/       # Pinia状态管理
│   │   ├── views/        # 页面组件
│   │   └── assets/       # 静态资源
│   ├── package.json
│   └── vite.config.js
├── cmd/server/           # Go后端入口
├── internal/             # 后端核心代码
├── plugins/              # 插件系统（21+插件）
├── configs/              # 配置文件
├── web/static/           # 前端构建产物
├── build-vue.sh          # 前端构建脚本
├── demo-vue3.sh          # 演示脚本
└── docker-build-vue3.sh  # Docker构建脚本
```

## 🎨 技术栈

### 前端
- **框架**: Vue 3.5.13
- **构建**: Vite 5.0+
- **路由**: Vue Router 4
- **状态**: Pinia
- **图表**: ECharts 5
- **图标**: Font Awesome 6
- **HTTP**: Axios

### 后端
- **语言**: Go 1.21+
- **框架**: Gin
- **数据库**: SQLite
- **AI**: OpenAI API / Ollama

## 🎨 设计系统

### Apple风格组件

#### 按钮
```vue
<AppleButton type="primary">主要按钮</AppleButton>
<AppleButton type="secondary">次要按钮</AppleButton>
<AppleButton type="ghost">幽灵按钮</AppleButton>
```

#### 卡片
```vue
<AppleCard title="标题" hoverable>
  <template #extra>
    <AppleButton>操作</AppleButton>
  </template>
  内容
</AppleCard>
```

#### 输入框
```vue
<AppleInput
  v-model="value"
  label="标签"
  placeholder="请输入"
  icon="fas fa-user"
/>
```

### 设计规范

#### 颜色
- 主色: `#0a84ff` (Apple Blue)
- 成功: `#30d158` (Apple Green)
- 警告: `#ff9f0a` (Apple Orange)
- 危险: `#ff453a` (Apple Red)

#### 动画
- 标准: `cubic-bezier(0.4, 0.0, 0.2, 1)`
- 弹性: `cubic-bezier(0.175, 0.885, 0.32, 1.275)`

## 📊 功能模块

### 核心模块
1. **仪表板** - 系统概览，实时监控
2. **CMDB** - 资产管理，配置管理
3. **监控中心** - 性能监控，健康检查
4. **告警管理** - 告警规则，通知管理

### AI智能
5. **AI日志分析** - 智能分析，故障诊断
6. **日志中心** - 日志采集，查询分析

### 自动化
7. **任务调度** - 定时任务，批量执行
8. **工作流** - 流程编排，自动化运维
9. **Ansible** - 配置管理，批量部署
10. **CI/CD** - 持续集成，持续部署

### 容器编排
11. **Kubernetes** - 容器管理，服务编排

### 扩展功能
12. **域名监控** - 域名到期，SSL证书
13. **服务拓扑** - 依赖关系，调用链路
14. **成本分析** - 资源成本，优化建议
15. **值班管理** - 排班管理，交接记录
16. **工单系统** - 工单流转，问题跟踪
17. **SQL审计** - SQL审核，性能分析

## 📚 文档

- [快速开始](VUE3快速开始.md) - 详细的安装和使用指南
- [实施进度](VUE3实施进度.md) - 开发进度和计划
- [实施总结](VUE3实施总结.md) - 技术总结和最佳实践
- [API文档](API.md) - 后端API接口文档

## 🛠️ 开发指南

### 添加新页面

1. 创建页面组件
```bash
touch web-vue/src/views/NewPage.vue
```

2. 添加路由
```javascript
// web-vue/src/router/index.js
{
  path: 'new-page',
  name: 'NewPage',
  component: () => import('../views/NewPage.vue')
}
```

3. 添加菜单
```javascript
// web-vue/src/layouts/MainLayout.vue
{
  path: '/new-page',
  label: '新页面',
  icon: 'fas fa-star'
}
```

### 添加新API

```javascript
// web-vue/src/api/newapi.js
import request from './request'

export const getData = () => {
  return request({
    url: '/api/data',
    method: 'get'
  })
}
```

## 🔧 配置

### 环境变量
```bash
# web-vue/.env
VITE_API_BASE_URL=/api
```

### API代理
```javascript
// web-vue/vite.config.js
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

## 📈 性能

### 构建产物
- HTML: 0.45 KB
- CSS: ~90 KB (gzip: 28 KB)
- JS: ~1.3 MB (gzip: 430 KB)

### 加载性能
- 首屏加载: < 2s
- 路由切换: < 100ms
- API响应: < 500ms

## 🤝 贡献

欢迎贡献代码！

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启Pull Request

## 📄 许可证

MIT License - 完全开源，自由使用

## 🙏 致谢

- [Vue.js](https://vuejs.org/) - 渐进式JavaScript框架
- [Vite](https://vitejs.dev/) - 下一代前端构建工具
- [ECharts](https://echarts.apache.org/) - 强大的图表库
- [Font Awesome](https://fontawesome.com/) - 图标库
- [Apple Design](https://developer.apple.com/design/) - 设计灵感
- [腾讯蓝鲸](https://bk.tencent.com/) - 运维平台参考

## 📞 联系方式

- GitHub Issues: [提交问题](https://github.com/yourusername/lazy-auto-ops/issues)
- Email: your.email@example.com

## 🗺️ 路线图

### v1.0 (当前)
- ✅ Vue3前端基础框架
- ✅ Apple风格组件库
- ✅ 核心页面实现
- 🔄 AI日志分析

### v1.1 (计划中)
- ⏳ 完整的监控告警
- ⏳ 任务调度系统
- ⏳ 工作流编排
- ⏳ 日志中心

### v2.0 (未来)
- ⏳ 移动端适配
- ⏳ 多租户支持
- ⏳ 插件市场
- ⏳ 国际化支持

---

**Made with ❤️ by Lazy Auto Ops Team**

**Star ⭐ this repo if you like it!**
