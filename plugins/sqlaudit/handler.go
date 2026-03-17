package sqlaudit

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type SQLAuditHandler struct {
	db *gorm.DB
}

func NewSQLAuditHandler(db *gorm.DB) *SQLAuditHandler {
	return &SQLAuditHandler{db: db}
}

// ListInstances 数据库实例列表
func (h *SQLAuditHandler) ListInstances(c *gin.Context) {
	var instances []DBInstance
	if err := h.db.Find(&instances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": instances})
}

// CreateInstance 创建数据库实例
func (h *SQLAuditHandler) CreateInstance(c *gin.Context) {
	var instance DBInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&instance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": instance})
}

// GetInstance 获取实例详情
func (h *SQLAuditHandler) GetInstance(c *gin.Context) {
	id := c.Param("id")
	var instance DBInstance
	if err := h.db.First(&instance, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "实例不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": instance})
}

// UpdateInstance 更新实例
func (h *SQLAuditHandler) UpdateInstance(c *gin.Context) {
	id := c.Param("id")
	var instance DBInstance
	if err := h.db.First(&instance, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "实例不存在"})
		return
	}
	var req DBInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"type":        req.Type,
		"host":        req.Host,
		"port":        req.Port,
		"username":    req.Username,
		"password":    req.Password,
		"database":    req.Database,
		"charset":     req.Charset,
		"status":      req.Status,
		"environment": req.Environment,
		"description": req.Description,
	}
	if err := h.db.Model(&instance).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&instance, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": instance})
}

// DeleteInstance 删除实例
func (h *SQLAuditHandler) DeleteInstance(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&DBInstance{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 测试连接
func (h *SQLAuditHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	var instance DBInstance
	if err := h.db.First(&instance, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "实例不存在"})
		return
	}

	db, err := h.connectDB(&instance)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "连接成功"})
}

// ListWorkOrders 工单列表
func (h *SQLAuditHandler) ListWorkOrders(c *gin.Context) {
	var orders []SQLWorkOrder
	query := h.db.Preload("Instance").Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if instanceID := c.Query("instance_id"); instanceID != "" {
		query = query.Where("instance_id = ?", instanceID)
	}

	if err := query.Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": orders})
}

// CreateWorkOrder 创建工单
func (h *SQLAuditHandler) CreateWorkOrder(c *gin.Context) {
	var order SQLWorkOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	order.Submitter = c.GetString("username")
	order.Status = 0 // 待审核

	// 获取审核规则
	var rules []SQLAuditRule
	h.db.Where("enabled = ?", true).Find(&rules)

	// 使用分析器分析SQL
	analyzer := NewSQLAnalyzer(rules)
	analyzeResult := analyzer.Analyze(order.SQLContent)

	// 设置审核结果
	order.SQLType = analyzeResult.SQLType
	order.AuditLevel = analyzeResult.Level

	// 格式化审核结果
	if len(analyzeResult.Issues) > 0 {
		var results []string
		for _, issue := range analyzeResult.Issues {
			results = append(results, fmt.Sprintf("[%s] %s", levelToString(issue.Level), issue.Message))
		}
		order.AuditResult = strings.Join(results, "\n")
	} else {
		order.AuditResult = "审核通过"
	}

	// 生成回滚SQL
	order.RollbackSQL = GenerateRollbackSQL(analyzeResult.SQLType, order.SQLContent, analyzeResult.TableNames)

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 返回详细的分析结果
	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"data":     order,
		"analysis": analyzeResult,
	})
}

// GetWorkOrder 获取工单详情
func (h *SQLAuditHandler) GetWorkOrder(c *gin.Context) {
	id := c.Param("id")
	var order SQLWorkOrder
	if err := h.db.Preload("Instance").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": order})
}

// ReviewWorkOrder 审核工单
func (h *SQLAuditHandler) ReviewWorkOrder(c *gin.Context) {
	id := c.Param("id")
	var order SQLWorkOrder
	if err := h.db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	if order.Status != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单状态不允许审核"})
		return
	}

	var req struct {
		Approved bool   `json:"approved"`
		Remark   string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{
		"reviewer":      c.GetString("username"),
		"reviewed_at":   now,
		"review_remark": req.Remark,
	}

	if req.Approved {
		updates["status"] = 1 // 审核通过
	} else {
		updates["status"] = 2 // 审核拒绝
	}

	h.db.Model(&order).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "审核完成"})
}

// ExecuteWorkOrder 执行工单
func (h *SQLAuditHandler) ExecuteWorkOrder(c *gin.Context) {
	id := c.Param("id")
	var order SQLWorkOrder
	if err := h.db.Preload("Instance").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}

	if order.Status != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单未审核通过"})
		return
	}

	// 更新状态为执行中
	h.db.Model(&order).Update("status", 3)

	// 连接数据库
	db, err := h.connectDB(order.Instance)
	if err != nil {
		h.db.Model(&order).Update("status", 5)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "连接数据库失败: " + err.Error()})
		return
	}
	defer db.Close()

	// 执行SQL
	startTime := time.Now()
	result, err := db.Exec(order.SQLContent)
	executeTime := int(time.Since(startTime).Milliseconds())

	now := time.Now()
	updates := map[string]interface{}{
		"executor":     c.GetString("username"),
		"executed_at":  now,
		"execute_time": executeTime,
	}

	if err != nil {
		updates["status"] = 5 // 执行失败
		h.db.Model(&order).Updates(updates)

		// 记录审计日志
		h.recordAuditLog(&order, 0, err.Error())

		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "执行失败: " + err.Error()})
		return
	}

	affectedRows, _ := result.RowsAffected()
	updates["status"] = 4 // 执行成功
	updates["affected_rows"] = affectedRows
	h.db.Model(&order).Updates(updates)

	// 记录审计日志
	h.recordAuditLog(&order, 1, "")

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"affected_rows": affectedRows,
		"execute_time":  executeTime,
	}})
}

// ListAuditLogs 审计日志列表
func (h *SQLAuditHandler) ListAuditLogs(c *gin.Context) {
	var logs []SQLAuditLog
	query := h.db.Order("executed_at DESC")

	if instanceID := c.Query("instance_id"); instanceID != "" {
		query = query.Where("instance_id = ?", instanceID)
	}
	if sqlType := c.Query("sql_type"); sqlType != "" {
		query = query.Where("sql_type = ?", sqlType)
	}
	if username := c.Query("username"); username != "" {
		query = query.Where("username = ?", username)
	}

	if err := query.Limit(500).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": logs})
}

// ListRules 审核规则列表
func (h *SQLAuditHandler) ListRules(c *gin.Context) {
	var rules []SQLAuditRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

// CreateRule 创建审核规则
func (h *SQLAuditHandler) CreateRule(c *gin.Context) {
	var rule SQLAuditRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// UpdateRule 更新规则
func (h *SQLAuditHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule SQLAuditRule
	if err := h.db.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	var req SQLAuditRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"type":        req.Type,
		"level":       req.Level,
		"pattern":     req.Pattern,
		"message":     req.Message,
		"suggestion":  req.Suggestion,
		"enabled":     req.Enabled,
		"description": req.Description,
	}
	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&rule, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// DeleteRule 删除规则
func (h *SQLAuditHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&SQLAuditRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// 辅助方法
func (h *SQLAuditHandler) connectDB(instance *DBInstance) (*sql.DB, error) {
	var dsn string
	switch instance.Type {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true",
			instance.Username, instance.Password, instance.Host, instance.Port, instance.Database, instance.Charset)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", instance.Type)
	}

	db, err := sql.Open(instance.Type, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func (h *SQLAuditHandler) detectSQLType(sqlContent string) string {
	sql := strings.ToUpper(strings.TrimSpace(sqlContent))
	if strings.HasPrefix(sql, "SELECT") {
		return "DQL"
	} else if strings.HasPrefix(sql, "INSERT") || strings.HasPrefix(sql, "UPDATE") || strings.HasPrefix(sql, "DELETE") {
		return "DML"
	} else if strings.HasPrefix(sql, "CREATE") || strings.HasPrefix(sql, "ALTER") || strings.HasPrefix(sql, "DROP") {
		return "DDL"
	}
	return "OTHER"
}

func (h *SQLAuditHandler) auditSQL(sqlContent string) (string, int) {
	var results []string
	maxLevel := 0

	// 获取启用的规则
	var rules []SQLAuditRule
	h.db.Where("enabled = ?", true).Find(&rules)

	// 内置规则
	builtinRules := []struct {
		pattern string
		level   int
		message string
	}{
		{`(?i)select\s+\*`, 1, "不建议使用 SELECT *，请明确指定字段"},
		{`(?i)delete\s+from\s+\w+\s*$`, 2, "DELETE 语句缺少 WHERE 条件，可能删除全表数据"},
		{`(?i)update\s+\w+\s+set\s+.*\s*$`, 2, "UPDATE 语句缺少 WHERE 条件，可能更新全表数据"},
		{`(?i)drop\s+table`, 2, "DROP TABLE 操作需要谨慎"},
		{`(?i)truncate\s+table`, 2, "TRUNCATE TABLE 操作需要谨慎"},
		{`(?i)alter\s+table.*drop`, 1, "ALTER TABLE DROP 操作需要谨慎"},
	}

	// 检查内置规则
	for _, rule := range builtinRules {
		if matched, _ := regexp.MatchString(rule.pattern, sqlContent); matched {
			results = append(results, fmt.Sprintf("[%s] %s", levelToString(rule.level), rule.message))
			if rule.level > maxLevel {
				maxLevel = rule.level
			}
		}
	}

	// 检查自定义规则
	for _, rule := range rules {
		if matched, _ := regexp.MatchString(rule.Pattern, sqlContent); matched {
			results = append(results, fmt.Sprintf("[%s] %s", levelToString(rule.Level), rule.Message))
			if rule.Level > maxLevel {
				maxLevel = rule.Level
			}
		}
	}

	if len(results) == 0 {
		return "审核通过", 0
	}

	return strings.Join(results, "\n"), maxLevel
}

func levelToString(level int) string {
	switch level {
	case 0:
		return "INFO"
	case 1:
		return "WARNING"
	case 2:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (h *SQLAuditHandler) recordAuditLog(order *SQLWorkOrder, status int, errorMsg string) {
	log := SQLAuditLog{
		InstanceID:   order.InstanceID,
		InstanceName: order.Instance.Name,
		Database:     order.Database,
		Username:     order.Executor,
		SQLType:      order.SQLType,
		SQLContent:   order.SQLContent,
		AffectedRows: order.AffectedRows,
		ExecuteTime:  order.ExecuteTime,
		Status:       status,
		ErrorMsg:     errorMsg,
		ExecutedAt:   time.Now(),
	}
	h.db.Create(&log)
}

// AnalyzeSQL 分析SQL（不创建工单）
func (h *SQLAuditHandler) AnalyzeSQL(c *gin.Context) {
	var req struct {
		SQLContent string `json:"sql_content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取审核规则
	var rules []SQLAuditRule
	h.db.Where("enabled = ?", true).Find(&rules)

	// 分析SQL
	analyzer := NewSQLAnalyzer(rules)
	result := analyzer.Analyze(req.SQLContent)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetStatistics 获取统计信息
func (h *SQLAuditHandler) GetStatistics(c *gin.Context) {
	stats := make(map[string]interface{})

	// 工单统计
	var orderStats struct {
		Total    int64
		Pending  int64
		Approved int64
		Rejected int64
		Executed int64
		Failed   int64
	}
	h.db.Model(&SQLWorkOrder{}).Count(&orderStats.Total)
	h.db.Model(&SQLWorkOrder{}).Where("status = ?", 0).Count(&orderStats.Pending)
	h.db.Model(&SQLWorkOrder{}).Where("status = ?", 1).Count(&orderStats.Approved)
	h.db.Model(&SQLWorkOrder{}).Where("status = ?", 2).Count(&orderStats.Rejected)
	h.db.Model(&SQLWorkOrder{}).Where("status = ?", 4).Count(&orderStats.Executed)
	h.db.Model(&SQLWorkOrder{}).Where("status = ?", 5).Count(&orderStats.Failed)
	stats["orders"] = orderStats

	// 审计日志统计
	var logStats struct {
		Total   int64
		Success int64
		Failed  int64
		Today   int64
	}
	h.db.Model(&SQLAuditLog{}).Count(&logStats.Total)
	h.db.Model(&SQLAuditLog{}).Where("status = ?", 1).Count(&logStats.Success)
	h.db.Model(&SQLAuditLog{}).Where("status = ?", 0).Count(&logStats.Failed)
	today := time.Now().Format("2006-01-02")
	h.db.Model(&SQLAuditLog{}).Where("DATE(executed_at) = ?", today).Count(&logStats.Today)
	stats["logs"] = logStats

	// SQL类型分布
	var typeStats []struct {
		SQLType string
		Count   int64
	}
	h.db.Model(&SQLAuditLog{}).
		Select("sql_type, COUNT(*) as count").
		Group("sql_type").
		Find(&typeStats)
	stats["types"] = typeStats

	// 数据库实例统计
	var instanceCount int64
	h.db.Model(&DBInstance{}).Count(&instanceCount)
	stats["instances"] = instanceCount

	// 审核规则统计
	var ruleCount int64
	h.db.Model(&SQLAuditRule{}).Where("enabled = ?", true).Count(&ruleCount)
	stats["rules"] = ruleCount

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}
