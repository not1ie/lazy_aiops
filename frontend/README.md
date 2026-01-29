# Lazy Auto Ops - Vue3 前端

> AI驱动的轻量级运维平台 - 前端应用

## 技术栈

- **框架**: Vue 3.3+
- **构建工具**: Vite 5.0+
- **路由**: Vue Router 4
- **状态管理**: Pinia
- **UI风格**: Apple Design System
- **图表**: ECharts 5
- **图标**: Font Awesome 6
- **HTTP**: Axios

## 快速开始

### 安装依赖
```bash
npm install
```

### 开发模式
```bash
npm run dev
```

访问: http://localhost:5173

### 构建生产版本
```bash
npm run build
```

构建产物在 `dist/` 目录

## 项目结构

```
src/
├── api/              # API接口
│   ├── request.js    # Axios封装
│   └── auth.js       # 认证接口
├── assets/           # 静态资源
│   └── styles/       # 样式文件
├── components/       # 通用组件
│   ├── AppleButton.vue
│   ├── AppleCard.vue
│   └── AppleInput.vue
├── router/           # 路由配置
├── stores/           # 状态管理
│   └── user.js       # 用户状态
├── views/            # 页面
│   ├── Login.vue     # 登录页
│   └── Dashboard.vue # 仪表板
├── App.vue           # 根组件
└── main.js           # 入口文件
```

## 默认账号

- 用户名: `admin`
- 密码: `admin123`

## 开发说明

### API代理

开发环境下，所有 `/api` 请求会被代理到 `http://localhost:8080`

### 组件使用

```vue
<template>
  <AppleButton type="primary" @click="handleClick">
    点击我
  </AppleButton>
  
  <AppleCard title="标题" hoverable>
    内容
  </AppleCard>
  
  <AppleInput v-model="value" label="标签" />
</template>
```

## License

MIT
