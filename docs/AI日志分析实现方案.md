# AI日志分析实现方案

## 概述
为Lazy Auto Ops平台实现完整的AI日志分析功能，包括智能日志分析、故障诊断、根因分析和解决方案推荐。

## 已完成功能

### 1. 核心分析引擎 ✅
**文件**: `plugins/ai/log_analyzer.go`

- **日志预处理**: 智能过滤和优先级排序
  - 优先保留ERROR和WARNING级别日志
  - 限制日志数量避免token超限（最多50条）
  - 自动清理和格式化日志内容

- **异常检测**: 多维度异常识别
  - 错误关键词检测（exception, timeout, oom等）
  - 日志级别统计分析
  - 阈值告警判断

- **AI分析**: 结构化分析结果
  - 根本原因分析（root_cause）
  - 影响范围评估（impact）
  - 解决方案推荐（solutions）
  - 预防措施建议（prevention）
  - 置信度计算（confidence）

### 2. 数据模型 ✅
**文件**: `plugins/ai/models.go`

```go
type LogAnalysis struct {
    Service      string  // 服务名称
    NeedAlert    bool    // 是否需要告警
    AlertLevel   string  // 告警级别: critical/warning/info
    RootCause    string  // 根本原因
    Impact       string  // 影响范围
    Solutions    string  // 解决方案
    Prevention   string  // 预防措施
    Confidence   float64 // 置信度
    LogCount     int     // 日志总数
    ErrorCount   int     // 错误数量
    WarningCount int     // 警告数量
}
```

### 3. API接口 ✅
**文件**: `plugins/ai/handler.go`, `plugins/ai/plugin.go`

- `POST /api/ai/analyze/logs-detailed` - 详细日志分析
- `GET /api/ai/analyze/history` - 分析历史记录
- `GET /api/ai/analyze/:id` - 获取分析详情
- `POST /api/ai/analyze/logs` - 简单日志分析
- `POST /api/ai/analyze/error` - 错误分析
- `POST /api/ai/analyze/performance` - 性能分析

### 4. AI服务集成 ✅
**文件**: `plugins/ai/service.go`

- 支持多种LLM提供商：
  - OpenAI (GPT-3.5/GPT-4)
  - Azure OpenAI
  - Ollama (本地部署)
- 智能提示词构建
- JSON响应解析
- 容错处理（未配置时返回友好提示）

## 使用示例

### 1. 配置AI服务
在 `configs/config.yaml` 中配置：

```yaml
plugins:
  ai:
    enabled: true
    config:
      provider: openai  # 或 azure, ollama
      api_key: sk-xxx
      base_url: https://api.openai.com/v1
      model: gpt-3.5-turbo
```

### 2. 调用分析API

```bash
# 详细日志分析
curl -X POST http://localhost:8080/api/ai/analyze/logs-detailed \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "service": "web-api",
    "log_level": "error",
    "logs": [
      "2024-01-28 10:00:00 ERROR Connection timeout to database",
      "2024-01-28 10:00:01 ERROR Failed to execute query",
      "2024-01-28 10:00:02 WARN Retry attempt 3/3 failed"
    ]
  }'

# 响应示例
{
  "code": 0,
  "data": {
    "id": "analysis_1706414400000",
    "service": "web-api",
    "need_alert": true,
    "alert_level": "critical",
    "root_cause": "数据库连接超时导致查询失败",
    "impact": [
      "用户请求失败",
      "服务可用性下降",
      "可能影响业务流程"
    ],
    "solutions": [
      "检查数据库服务状态",
      "增加连接超时时间",
      "优化数据库连接池配置"
    ],
    "prevention": [
      "配置数据库健康检查",
      "设置连接池监控告警",
      "定期检查数据库性能"
    ],
    "confidence": 0.85,
    "log_count": 3,
    "error_count": 2,
    "warning_count": 1,
    "analyzed_at": "2024-01-28T10:00:00Z"
  }
}
```

### 3. 查询分析历史

```bash
# 获取所有分析记录
curl http://localhost:8080/api/ai/analyze/history \
  -H "Authorization: Bearer <token>"

# 按服务筛选
curl http://localhost:8080/api/ai/analyze/history?service=web-api \
  -H "Authorization: Bearer <token>"
```

## 技术特点

### 1. 智能分析
- **多级过滤**: 优先分析ERROR和WARNING日志
- **上下文理解**: 结合服务信息和时间范围
- **结构化输出**: JSON格式便于前端展示

### 2. 高可用性
- **降级处理**: AI服务不可用时返回基础分析
- **超时控制**: 避免长时间等待
- **错误恢复**: 解析失败时返回原始文本

### 3. 可扩展性
- **插件化设计**: 易于添加新的分析维度
- **多模型支持**: 可切换不同的LLM提供商
- **配置灵活**: 支持自定义提示词和参数

## 前端集成

在Vue3前端 `web-vue/src/views/LogAnalysis.vue` 中已实现：

- 日志输入和服务选择
- 实时分析进度显示
- 结构化结果展示
- 历史记录查询
- 详情查看

## 性能优化

1. **日志限制**: 最多分析50条日志，避免token超限
2. **优先级排序**: 优先分析错误和警告日志
3. **缓存结果**: 分析结果存储到数据库
4. **异步处理**: 支持后台异步分析（可扩展）

## 安全考虑

1. **认证授权**: 所有API需要JWT认证
2. **数据脱敏**: 敏感信息自动过滤（可扩展）
3. **访问控制**: 基于角色的权限管理
4. **审计日志**: 记录所有分析操作

## 后续优化方向

1. **实时分析**: 集成日志流处理
2. **批量分析**: 支持大规模日志批量处理
3. **模式学习**: 基于历史数据的异常模式识别
4. **自动告警**: 分析结果自动触发告警
5. **多语言支持**: 支持英文日志分析

## 总结

AI日志分析功能已完整实现，包括：
- ✅ 核心分析引擎
- ✅ 数据模型和存储
- ✅ RESTful API接口
- ✅ AI服务集成
- ✅ 前端界面

可以立即投入使用，只需配置AI服务的API Key即可。
