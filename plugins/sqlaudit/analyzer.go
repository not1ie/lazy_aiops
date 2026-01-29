package sqlaudit

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLAnalyzer SQL分析器
type SQLAnalyzer struct {
	rules []SQLAuditRule
}

// NewSQLAnalyzer 创建SQL分析器
func NewSQLAnalyzer(rules []SQLAuditRule) *SQLAnalyzer {
	return &SQLAnalyzer{rules: rules}
}

// AnalyzeResult 分析结果
type AnalyzeResult struct {
	Pass         bool              `json:"pass"`
	Level        int               `json:"level"` // 0:通过 1:警告 2:错误
	Issues       []Issue           `json:"issues"`
	Suggestions  []string          `json:"suggestions"`
	SQLType      string            `json:"sql_type"`
	TableNames   []string          `json:"table_names"`
	AffectedRows int64             `json:"affected_rows_estimate"`
	RiskLevel    string            `json:"risk_level"` // low, medium, high, critical
	Metrics      map[string]interface{} `json:"metrics"`
}

// Issue 问题
type Issue struct {
	Type        string `json:"type"`
	Level       int    `json:"level"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
}

// Analyze 分析SQL
func (a *SQLAnalyzer) Analyze(sqlContent string) *AnalyzeResult {
	result := &AnalyzeResult{
		Pass:        true,
		Level:       0,
		Issues:      []Issue{},
		Suggestions: []string{},
		Metrics:     make(map[string]interface{}),
	}

	// 1. 检测SQL类型
	result.SQLType = a.detectSQLType(sqlContent)

	// 2. 提取表名
	result.TableNames = a.extractTableNames(sqlContent)

	// 3. 应用审核规则
	a.applyRules(sqlContent, result)

	// 4. 内置规则检查
	a.checkBuiltinRules(sqlContent, result)

	// 5. 计算风险等级
	result.RiskLevel = a.calculateRiskLevel(result)

	// 6. 生成建议
	a.generateSuggestions(result)

	// 7. 收集指标
	a.collectMetrics(sqlContent, result)

	return result
}

// detectSQLType 检测SQL类型
func (a *SQLAnalyzer) detectSQLType(sql string) string {
	sql = strings.ToUpper(strings.TrimSpace(sql))
	
	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return "DQL"
	case strings.HasPrefix(sql, "INSERT"):
		return "DML-INSERT"
	case strings.HasPrefix(sql, "UPDATE"):
		return "DML-UPDATE"
	case strings.HasPrefix(sql, "DELETE"):
		return "DML-DELETE"
	case strings.HasPrefix(sql, "CREATE"):
		return "DDL-CREATE"
	case strings.HasPrefix(sql, "ALTER"):
		return "DDL-ALTER"
	case strings.HasPrefix(sql, "DROP"):
		return "DDL-DROP"
	case strings.HasPrefix(sql, "TRUNCATE"):
		return "DDL-TRUNCATE"
	case strings.HasPrefix(sql, "GRANT"):
		return "DCL-GRANT"
	case strings.HasPrefix(sql, "REVOKE"):
		return "DCL-REVOKE"
	default:
		return "OTHER"
	}
}

// extractTableNames 提取表名
func (a *SQLAnalyzer) extractTableNames(sql string) []string {
	tables := []string{}
	
	// 简单的表名提取（可以使用SQL解析器做更精确的提取）
	patterns := []string{
		`(?i)from\s+(\w+)`,
		`(?i)join\s+(\w+)`,
		`(?i)into\s+(\w+)`,
		`(?i)update\s+(\w+)`,
		`(?i)table\s+(\w+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(sql, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tableName := match[1]
				if !contains(tables, tableName) {
					tables = append(tables, tableName)
				}
			}
		}
	}
	
	return tables
}

// applyRules 应用审核规则
func (a *SQLAnalyzer) applyRules(sql string, result *AnalyzeResult) {
	for _, rule := range a.rules {
		if !rule.Enabled {
			continue
		}
		
		matched, _ := regexp.MatchString(rule.Pattern, sql)
		if matched {
			issue := Issue{
				Type:       rule.Type,
				Level:      rule.Level,
				Message:    rule.Message,
				Suggestion: rule.Suggestion,
			}
			result.Issues = append(result.Issues, issue)
			
			if rule.Level > result.Level {
				result.Level = rule.Level
			}
			
			if rule.Level >= 2 {
				result.Pass = false
			}
		}
	}
}

// checkBuiltinRules 检查内置规则
func (a *SQLAnalyzer) checkBuiltinRules(sql string, result *AnalyzeResult) {
	sqlUpper := strings.ToUpper(sql)
	
	// 1. SELECT * 检查
	if matched, _ := regexp.MatchString(`(?i)select\s+\*`, sql); matched {
		result.Issues = append(result.Issues, Issue{
			Type:       "performance",
			Level:      1,
			Message:    "不建议使用 SELECT *",
			Suggestion: "明确指定需要查询的字段，避免不必要的数据传输",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
	
	// 2. 无WHERE条件的DELETE
	if matched, _ := regexp.MatchString(`(?i)delete\s+from\s+\w+\s*;?\s*$`, sql); matched {
		result.Issues = append(result.Issues, Issue{
			Type:       "security",
			Level:      2,
			Message:    "DELETE 语句缺少 WHERE 条件",
			Suggestion: "添加 WHERE 条件限制删除范围，避免误删全表数据",
		})
		result.Level = 2
		result.Pass = false
	}
	
	// 3. 无WHERE条件的UPDATE
	if matched, _ := regexp.MatchString(`(?i)update\s+\w+\s+set\s+[^;]+;?\s*$`, sql); matched {
		if !strings.Contains(sqlUpper, "WHERE") {
			result.Issues = append(result.Issues, Issue{
				Type:       "security",
				Level:      2,
				Message:    "UPDATE 语句缺少 WHERE 条件",
				Suggestion: "添加 WHERE 条件限制更新范围，避免误更新全表数据",
			})
			result.Level = 2
			result.Pass = false
		}
	}
	
	// 4. DROP TABLE 检查
	if strings.Contains(sqlUpper, "DROP TABLE") {
		result.Issues = append(result.Issues, Issue{
			Type:       "security",
			Level:      2,
			Message:    "DROP TABLE 操作风险极高",
			Suggestion: "请确认是否真的需要删除表，建议先备份数据",
		})
		result.Level = 2
		result.Pass = false
	}
	
	// 5. TRUNCATE TABLE 检查
	if strings.Contains(sqlUpper, "TRUNCATE TABLE") {
		result.Issues = append(result.Issues, Issue{
			Type:       "security",
			Level:      2,
			Message:    "TRUNCATE TABLE 会清空表数据且无法回滚",
			Suggestion: "请确认是否真的需要清空表，建议使用 DELETE 以便回滚",
		})
		result.Level = 2
		result.Pass = false
	}
	
	// 6. ALTER TABLE DROP 检查
	if matched, _ := regexp.MatchString(`(?i)alter\s+table.*drop`, sql); matched {
		result.Issues = append(result.Issues, Issue{
			Type:       "security",
			Level:      1,
			Message:    "ALTER TABLE DROP 操作会删除列或索引",
			Suggestion: "请确认是否真的需要删除，建议先备份数据",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
	
	// 7. 没有LIMIT的大查询
	if strings.Contains(sqlUpper, "SELECT") && !strings.Contains(sqlUpper, "LIMIT") {
		result.Issues = append(result.Issues, Issue{
			Type:       "performance",
			Level:      1,
			Message:    "SELECT 查询未使用 LIMIT",
			Suggestion: "建议添加 LIMIT 限制返回行数，避免查询过多数据",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
	
	// 8. 使用 OR 条件
	if strings.Contains(sqlUpper, " OR ") {
		result.Issues = append(result.Issues, Issue{
			Type:       "performance",
			Level:      1,
			Message:    "使用 OR 条件可能导致索引失效",
			Suggestion: "考虑使用 UNION 或 IN 替代 OR",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
	
	// 9. 使用 != 或 <> 
	if strings.Contains(sql, "!=") || strings.Contains(sql, "<>") {
		result.Issues = append(result.Issues, Issue{
			Type:       "performance",
			Level:      1,
			Message:    "使用 != 或 <> 可能导致索引失效",
			Suggestion: "考虑使用其他条件或优化查询",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
	
	// 10. LIKE 前缀通配符
	if matched, _ := regexp.MatchString(`(?i)like\s+['"]%`, sql); matched {
		result.Issues = append(result.Issues, Issue{
			Type:       "performance",
			Level:      1,
			Message:    "LIKE 使用前缀通配符会导致索引失效",
			Suggestion: "避免使用 LIKE '%xxx'，考虑使用全文索引",
		})
		if result.Level < 1 {
			result.Level = 1
		}
	}
}

// calculateRiskLevel 计算风险等级
func (a *SQLAnalyzer) calculateRiskLevel(result *AnalyzeResult) string {
	// 基于问题级别和数量计算风险
	errorCount := 0
	warningCount := 0
	
	for _, issue := range result.Issues {
		if issue.Level >= 2 {
			errorCount++
		} else if issue.Level == 1 {
			warningCount++
		}
	}
	
	// DDL操作风险较高
	if strings.HasPrefix(result.SQLType, "DDL") {
		if errorCount > 0 {
			return "critical"
		}
		return "high"
	}
	
	// DML操作
	if strings.HasPrefix(result.SQLType, "DML") {
		if errorCount > 0 {
			return "high"
		}
		if warningCount > 2 {
			return "medium"
		}
		if warningCount > 0 {
			return "low"
		}
	}
	
	// DQL操作
	if result.SQLType == "DQL" {
		if warningCount > 3 {
			return "medium"
		}
		if warningCount > 0 {
			return "low"
		}
	}
	
	return "low"
}

// generateSuggestions 生成建议
func (a *SQLAnalyzer) generateSuggestions(result *AnalyzeResult) {
	suggestions := []string{}
	
	// 基于SQL类型的建议
	switch result.SQLType {
	case "DML-DELETE", "DML-UPDATE":
		suggestions = append(suggestions, "建议在执行前先使用 SELECT 验证影响范围")
		suggestions = append(suggestions, "建议在事务中执行，以便出错时回滚")
	case "DDL-DROP", "DDL-TRUNCATE":
		suggestions = append(suggestions, "强烈建议先备份相关数据")
		suggestions = append(suggestions, "建议在业务低峰期执行")
	case "DDL-ALTER":
		suggestions = append(suggestions, "建议先在测试环境验证")
		suggestions = append(suggestions, "注意ALTER TABLE可能锁表，影响业务")
	}
	
	// 基于问题的建议
	if result.Level >= 2 {
		suggestions = append(suggestions, "存在严重问题，建议修改后再执行")
	} else if result.Level == 1 {
		suggestions = append(suggestions, "存在性能或安全隐患，建议优化")
	}
	
	result.Suggestions = suggestions
}

// collectMetrics 收集指标
func (a *SQLAnalyzer) collectMetrics(sql string, result *AnalyzeResult) {
	result.Metrics["sql_length"] = len(sql)
	result.Metrics["table_count"] = len(result.TableNames)
	result.Metrics["issue_count"] = len(result.Issues)
	result.Metrics["has_where"] = strings.Contains(strings.ToUpper(sql), "WHERE")
	result.Metrics["has_limit"] = strings.Contains(strings.ToUpper(sql), "LIMIT")
	result.Metrics["has_index_hint"] = strings.Contains(strings.ToUpper(sql), "USE INDEX") || 
		strings.Contains(strings.ToUpper(sql), "FORCE INDEX")
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// GenerateRollbackSQL 生成回滚SQL（简化版）
func GenerateRollbackSQL(sqlType, sqlContent string, tableNames []string) string {
	switch sqlType {
	case "DML-DELETE":
		// DELETE的回滚需要先查询要删除的数据
		if len(tableNames) > 0 {
			return fmt.Sprintf("-- 回滚DELETE需要先备份数据\n-- SELECT * FROM %s WHERE ...", tableNames[0])
		}
	case "DML-UPDATE":
		// UPDATE的回滚需要记录原始值
		if len(tableNames) > 0 {
			return fmt.Sprintf("-- 回滚UPDATE需要先记录原始值\n-- SELECT * FROM %s WHERE ...", tableNames[0])
		}
	case "DML-INSERT":
		// INSERT的回滚是DELETE
		if len(tableNames) > 0 {
			return fmt.Sprintf("-- DELETE FROM %s WHERE ...", tableNames[0])
		}
	case "DDL-CREATE":
		// CREATE的回滚是DROP
		if len(tableNames) > 0 {
			return fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableNames[0])
		}
	case "DDL-DROP":
		return "-- DROP操作无法回滚，请确保已备份数据"
	case "DDL-ALTER":
		return "-- ALTER操作的回滚需要根据具体修改内容确定"
	}
	return "-- 无法自动生成回滚SQL"
}
