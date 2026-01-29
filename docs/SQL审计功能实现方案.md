# SQL审计功能实现方案

## 概述
完整的SQL审计系统，提供SQL工单管理、智能审核、执行审计和风险控制功能。

---

## 🎯 核心功能

### 1. 数据库实例管理 ✅
- 支持多种数据库类型（MySQL/PostgreSQL/Oracle/SQL Server）
- 实例连接配置和测试
- 环境标识（dev/test/staging/prod）
- 实例状态监控

### 2. SQL工单系统 ✅
- 工单创建和提交
- 自动SQL审核
- 工单审批流程
- SQL执行和回滚
- 定时执行支持

### 3. 智能SQL分析器 ✅
- SQL类型识别（DDL/DML/DQL/DCL）
- 表名自动提取
- 风险等级评估
- 性能问题检测
- 安全隐患识别
- 自动生成建议

### 4. 审核规则引擎 ✅
- 内置审核规则
- 自定义规则支持
- 正则表达式匹配
- 规则优先级
- 规则启用/禁用

### 5. 审计日志 ✅
- 完整的执行记录
- 执行时间统计
- 影响行数记录
- 错误信息记录
- 多维度查询

### 6. 统计分析 ✅
- 工单统计
- 执行统计
- SQL类型分布
- 风险趋势分析

---

## 📊 数据模型

### DBInstance - 数据库实例
```go
type DBInstance struct {
    Name        string  // 实例名称
    Type        string  // 数据库类型
    Host        string  // 主机地址
    Port        int     // 端口
    Username    string  // 用户名
    Password    string  // 密码（加密存储）
    Database    string  // 数据库名
    Charset     string  // 字符集
    Status      int     // 状态
    Environment string  // 环境标识
    Description string  // 描述
}
```

### SQLWorkOrder - SQL工单
```go
type SQLWorkOrder struct {
    Title        string     // 工单标题
    InstanceID   string     // 实例ID
    Database     string     // 数据库名
    SQLType      string     // SQL类型
    SQLContent   string     // SQL内容
    Status       int        // 状态
    AuditResult  string     // 审核结果
    AuditLevel   int        // 审核级别
    AffectedRows int64      // 影响行数
    ExecuteTime  int        // 执行时间(ms)
    RollbackSQL  string     // 回滚SQL
    Submitter    string     // 提交人
    Reviewer     string     // 审核人
    Executor     string     // 执行人
    ReviewedAt   *time.Time // 审核时间
    ExecutedAt   *time.Time // 执行时间
    ScheduledAt  *time.Time // 定时执行时间
}
```

### SQLAuditLog - 审计日志
```go
type SQLAuditLog struct {
    InstanceID   string    // 实例ID
    InstanceName string    // 实例名称
    Database     string    // 数据库名
    Username     string    // 执行用户
    ClientIP     string    // 客户端IP
    SQLType      string    // SQL类型
    SQLContent   string    // SQL内容
    AffectedRows int64     // 影响行数
    ExecuteTime  int       // 执行时间(ms)
    Status       int       // 状态
    ErrorMsg     string    // 错误信息
    ExecutedAt   time.Time // 执行时间
}
```

### SQLAuditRule - 审核规则
```go
type SQLAuditRule struct {
    Name        string  // 规则名称
    Type        string  // 规则类型
    Level       int     // 级别: 0-info 1-warning 2-error
    Pattern     string  // 正则表达式
    Message     string  // 提示信息
    Suggestion  string  // 建议
    Enabled     bool    // 是否启用
    Description string  // 描述
}
```

---

## 🔍 SQL分析器

### 分析维度

#### 1. SQL类型识别
- DQL: SELECT查询
- DML: INSERT/UPDATE/DELETE
- DDL: CREATE/ALTER/DROP/TRUNCATE
- DCL: GRANT/REVOKE

#### 2. 风险评估
- **Critical**: DDL操作 + 严重问题
- **High**: DML操作 + 严重问题
- **Medium**: 多个警告
- **Low**: 少量警告或无问题

#### 3. 内置检查规则

**性能规则**:
- ❌ SELECT * - 不建议查询所有字段
- ❌ 无LIMIT - 可能返回大量数据
- ❌ LIKE前缀通配符 - 导致索引失效
- ❌ OR条件 - 可能导致索引失效
- ❌ != 或 <> - 可能导致索引失效

**安全规则**:
- 🚫 DELETE无WHERE - 删除全表数据
- 🚫 UPDATE无WHERE - 更新全表数据
- 🚫 DROP TABLE - 删除表
- 🚫 TRUNCATE TABLE - 清空表
- ⚠️ ALTER TABLE DROP - 删除列或索引

#### 4. 分析结果
```go
type AnalyzeResult struct {
    Pass         bool              // 是否通过
    Level        int               // 级别
    Issues       []Issue           // 问题列表
    Suggestions  []string          // 建议列表
    SQLType      string            // SQL类型
    TableNames   []string          // 涉及的表
    RiskLevel    string            // 风险等级
    Metrics      map[string]interface{} // 指标
}
```

---

## 📡 API接口

### 数据库实例管理
```
GET    /api/v1/sqlaudit/instances          # 实例列表
POST   /api/v1/sqlaudit/instances          # 创建实例
GET    /api/v1/sqlaudit/instances/:id      # 实例详情
PUT    /api/v1/sqlaudit/instances/:id      # 更新实例
DELETE /api/v1/sqlaudit/instances/:id      # 删除实例
POST   /api/v1/sqlaudit/instances/:id/test # 测试连接
```

### SQL工单管理
```
GET    /api/v1/sqlaudit/orders             # 工单列表
POST   /api/v1/sqlaudit/orders             # 创建工单
GET    /api/v1/sqlaudit/orders/:id         # 工单详情
POST   /api/v1/sqlaudit/orders/:id/review  # 审核工单
POST   /api/v1/sqlaudit/orders/:id/execute # 执行工单
```

### 审计日志
```
GET    /api/v1/sqlaudit/logs               # 审计日志列表
```

### 审核规则
```
GET    /api/v1/sqlaudit/rules              # 规则列表
POST   /api/v1/sqlaudit/rules              # 创建规则
PUT    /api/v1/sqlaudit/rules/:id          # 更新规则
DELETE /api/v1/sqlaudit/rules/:id          # 删除规则
```

### SQL分析
```
POST   /api/v1/sqlaudit/analyze            # 分析SQL
GET    /api/v1/sqlaudit/statistics         # 统计信息
```

---

## 💡 使用示例

### 1. 创建数据库实例

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/instances \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "生产数据库",
    "type": "mysql",
    "host": "192.168.1.100",
    "port": 3306,
    "username": "root",
    "password": "password",
    "database": "mydb",
    "charset": "utf8mb4",
    "environment": "prod",
    "description": "生产环境主库"
  }'
```

### 2. 测试数据库连接

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/instances/{id}/test \
  -H "Authorization: Bearer <token>"
```

### 3. 分析SQL（不创建工单）

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/analyze \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "sql_content": "DELETE FROM users WHERE status = 0"
  }'
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "pass": true,
    "level": 0,
    "issues": [],
    "suggestions": [
      "建议在执行前先使用 SELECT 验证影响范围",
      "建议在事务中执行，以便出错时回滚"
    ],
    "sql_type": "DML-DELETE",
    "table_names": ["users"],
    "risk_level": "low",
    "metrics": {
      "sql_length": 35,
      "table_count": 1,
      "issue_count": 0,
      "has_where": true,
      "has_limit": false
    }
  }
}
```

### 4. 创建SQL工单

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "title": "清理无效用户数据",
    "instance_id": "instance-id-here",
    "database": "mydb",
    "sql_content": "DELETE FROM users WHERE status = 0 AND created_at < DATE_SUB(NOW(), INTERVAL 1 YEAR)"
  }'
```

**响应包含**:
- 工单信息
- 自动审核结果
- 风险评估
- 建议列表

### 5. 审核工单

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/orders/{id}/review \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "approved": true,
    "remark": "审核通过，可以执行"
  }'
```

### 6. 执行工单

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/orders/{id}/execute \
  -H "Authorization: Bearer <token>"
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "affected_rows": 1523,
    "execute_time": 245
  }
}
```

### 7. 查询审计日志

```bash
# 查询所有日志
curl http://localhost:8080/api/v1/sqlaudit/logs \
  -H "Authorization: Bearer <token>"

# 按SQL类型筛选
curl "http://localhost:8080/api/v1/sqlaudit/logs?sql_type=DML-DELETE" \
  -H "Authorization: Bearer <token>"

# 按用户筛选
curl "http://localhost:8080/api/v1/sqlaudit/logs?username=admin" \
  -H "Authorization: Bearer <token>"
```

### 8. 获取统计信息

```bash
curl http://localhost:8080/api/v1/sqlaudit/statistics \
  -H "Authorization: Bearer <token>"
```

**响应示例**:
```json
{
  "code": 0,
  "data": {
    "orders": {
      "total": 156,
      "pending": 5,
      "approved": 120,
      "rejected": 8,
      "executed": 115,
      "failed": 3
    },
    "logs": {
      "total": 2341,
      "success": 2298,
      "failed": 43,
      "today": 87
    },
    "types": [
      {"sql_type": "DQL", "count": 1523},
      {"sql_type": "DML-UPDATE", "count": 456},
      {"sql_type": "DML-DELETE", "count": 234}
    ],
    "instances": 8,
    "rules": 12
  }
}
```

---

## 🔒 安全特性

### 1. 审核机制
- 自动SQL审核
- 人工审批流程
- 多级审核支持
- 审核记录追踪

### 2. 风险控制
- 风险等级评估
- 高风险操作拦截
- 生产环境保护
- 回滚SQL生成

### 3. 权限控制
- 基于角色的访问控制
- 实例级别权限
- 操作权限分离
- 审计日志记录

### 4. 数据保护
- 密码加密存储
- 敏感信息脱敏
- 连接信息保护
- 审计日志保留

---

## 📈 工单状态流转

```
创建工单 → 待审核(0)
    ↓
审核 → 审核通过(1) / 审核拒绝(2)
    ↓
执行 → 执行中(3) → 执行成功(4) / 执行失败(5)
    ↓
回滚 → 已回滚(6)
```

---

## 🎯 最佳实践

### 1. 工单提交
- 提供清晰的标题和说明
- 在测试环境先验证SQL
- 评估影响范围
- 准备回滚方案

### 2. SQL编写
- 避免使用 SELECT *
- 添加 WHERE 条件
- 使用 LIMIT 限制返回行数
- 避免全表扫描
- 合理使用索引

### 3. 审核流程
- 仔细检查SQL语句
- 评估业务影响
- 确认执行时间
- 准备应急预案

### 4. 执行操作
- 选择业务低峰期
- 在事务中执行
- 监控执行进度
- 及时处理异常

---

## 🔧 配置说明

### 启用SQL审计插件

在 `configs/config.yaml` 中配置：

```yaml
plugins:
  sqlaudit:
    enabled: true
    config:
      # 自动审核级别: 0-仅记录 1-警告 2-拦截
      auto_review_level: 1
      # 是否需要人工审核
      require_manual_review: true
      # 生产环境强制审核
      prod_force_review: true
```

### 自定义审核规则

```bash
curl -X POST http://localhost:8080/api/v1/sqlaudit/rules \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "禁止在生产环境DROP",
    "type": "security",
    "level": 2,
    "pattern": "(?i)drop\\s+table",
    "message": "生产环境禁止DROP TABLE操作",
    "suggestion": "请联系DBA处理",
    "enabled": true,
    "description": "保护生产环境数据安全"
  }'
```

---

## 📊 监控指标

### 关键指标
- 工单提交量
- 审核通过率
- 执行成功率
- 平均执行时间
- 风险SQL占比
- 审计日志量

### 告警规则
- 执行失败率超过阈值
- 高风险SQL频繁出现
- 审核拒绝率异常
- 执行时间过长

---

## 🎉 总结

SQL审计功能已完整实现，包括：

✅ **数据库实例管理** - 多数据库支持  
✅ **SQL工单系统** - 完整的审批流程  
✅ **智能分析器** - 自动风险评估  
✅ **审核规则引擎** - 灵活的规则配置  
✅ **审计日志** - 完整的操作记录  
✅ **统计分析** - 多维度数据统计  

**可以立即投入使用！** 🚀

---

**更新时间**: 2024-01-28  
**版本**: v1.0.0  
**Made with ❤️ by Lazy Auto Ops Team**
