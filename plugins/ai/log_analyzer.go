package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogAnalysisRequest 日志分析请求
type LogAnalysisRequest struct {
	Service   string    `json:"service" binding:"required"`
	LogLevel  string    `json:"log_level"`
	TimeRange TimeRange `json:"time_range"`
	Logs      []string  `json:"logs"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// LogAnalysisResult 日志分析结果
type LogAnalysisResult struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	NeedAlert   bool      `json:"need_alert"`
	AlertLevel  string    `json:"alert_level"` // critical/warning/info
	RootCause   string    `json:"root_cause"`
	Impact      []string  `json:"impact"`
	Solutions   []string  `json:"solutions"`
	Prevention  []string  `json:"prevention"`
	Confidence  float64   `json:"confidence"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
	LogCount    int       `json:"log_count"`
	ErrorCount  int       `json:"error_count"`
	WarningCount int      `json:"warning_count"`
}

// AnalyzeLogsDetailed 详细日志分析
func (s *AIService) AnalyzeLogsDetailed(req *LogAnalysisRequest) (*LogAnalysisResult, error) {
	// 1. 预处理日志
	processedLogs := s.preprocessLogs(req.Logs)
	
	// 2. 统计信息
	errorCount, warningCount := s.countLogLevels(processedLogs)
	
	// 3. 检测异常模式
	hasAnomaly := s.detectAnomalies(processedLogs)
	
	// 4. 如果没有异常，返回正常结果
	if !hasAnomaly {
		return &LogAnalysisResult{
			ID:          generateID(),
			Service:     req.Service,
			NeedAlert:   false,
			AlertLevel:  "info",
			RootCause:   "未检测到异常",
			Confidence:  0.9,
			AnalyzedAt:  time.Now(),
			LogCount:    len(processedLogs),
			ErrorCount:  errorCount,
			WarningCount: warningCount,
		}, nil
	}
	
	// 5. 构建AI分析提示词
	prompt := s.buildLogAnalysisPrompt(req.Service, processedLogs)
	
	// 6. 调用AI分析
	aiResponse, err := s.callAI(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI分析失败: %w", err)
	}
	
	// 7. 解析AI响应
	result := s.parseLogAnalysisResponse(aiResponse)
	result.ID = generateID()
	result.Service = req.Service
	result.AnalyzedAt = time.Now()
	result.LogCount = len(processedLogs)
	result.ErrorCount = errorCount
	result.WarningCount = warningCount
	
	// 8. 计算置信度
	result.Confidence = s.calculateConfidence(result, errorCount, warningCount)
	
	return result, nil
}

// preprocessLogs 预处理日志
func (s *AIService) preprocessLogs(logs []string) []string {
	processed := make([]string, 0, len(logs))
	
	// 限制日志数量，避免token超限
	maxLogs := 50
	if len(logs) > maxLogs {
		// 优先保留ERROR和WARNING级别的日志
		errorLogs := []string{}
		warningLogs := []string{}
		otherLogs := []string{}
		
		for _, log := range logs {
			logUpper := strings.ToUpper(log)
			if strings.Contains(logUpper, "ERROR") || strings.Contains(logUpper, "FATAL") {
				errorLogs = append(errorLogs, log)
			} else if strings.Contains(logUpper, "WARN") {
				warningLogs = append(warningLogs, log)
			} else {
				otherLogs = append(otherLogs, log)
			}
		}
		
		// 组合日志
		processed = append(processed, errorLogs...)
		processed = append(processed, warningLogs...)
		
		remaining := maxLogs - len(processed)
		if remaining > 0 && len(otherLogs) > 0 {
			if len(otherLogs) > remaining {
				processed = append(processed, otherLogs[:remaining]...)
			} else {
				processed = append(processed, otherLogs...)
			}
		}
	} else {
		processed = logs
	}
	
	return processed
}

// countLogLevels 统计日志级别
func (s *AIService) countLogLevels(logs []string) (errorCount, warningCount int) {
	for _, log := range logs {
		logUpper := strings.ToUpper(log)
		if strings.Contains(logUpper, "ERROR") || strings.Contains(logUpper, "FATAL") {
			errorCount++
		} else if strings.Contains(logUpper, "WARN") {
			warningCount++
		}
	}
	return
}

// detectAnomalies 检测异常
func (s *AIService) detectAnomalies(logs []string) bool {
	errorCount, warningCount := s.countLogLevels(logs)
	
	// 简单规则：如果有ERROR或WARNING数量超过阈值，认为有异常
	if errorCount > 0 {
		return true
	}
	if warningCount > 5 {
		return true
	}
	
	// 检查关键词
	keywords := []string{
		"exception", "timeout", "connection refused", 
		"out of memory", "oom", "panic", "crash",
		"failed", "cannot", "unable to",
	}
	
	for _, log := range logs {
		logLower := strings.ToLower(log)
		for _, keyword := range keywords {
			if strings.Contains(logLower, keyword) {
				return true
			}
		}
	}
	
	return false
}

// buildLogAnalysisPrompt 构建日志分析提示词
func (s *AIService) buildLogAnalysisPrompt(service string, logs []string) string {
	logsText := strings.Join(logs, "\n")
	
	prompt := fmt.Sprintf(`你是一个专业的运维工程师，请分析以下日志并给出诊断结果。

## 服务信息
服务名称: %s
日志数量: %d条

## 日志内容
%s

## 请按以下JSON格式回答
{
  "need_alert": true/false,
  "alert_level": "critical/warning/info",
  "root_cause": "故障的根本原因，不超过100字",
  "impact": ["影响1", "影响2", "影响3"],
  "solutions": ["解决方案1", "解决方案2", "解决方案3"],
  "prevention": ["预防措施1", "预防措施2", "预防措施3"]
}

注意：
1. 只返回JSON，不要其他内容
2. root_cause要简洁明了
3. impact、solutions、prevention各提供2-5条
4. 如果没有严重问题，need_alert设为false
`, service, len(logs), logsText)
	
	return prompt
}

// parseLogAnalysisResponse 解析AI响应
func (s *AIService) parseLogAnalysisResponse(response string) *LogAnalysisResult {
	result := &LogAnalysisResult{
		NeedAlert:  false,
		AlertLevel: "info",
		RootCause:  "分析完成",
		Impact:     []string{},
		Solutions:  []string{},
		Prevention: []string{},
	}
	
	// 尝试解析JSON
	var jsonResult struct {
		NeedAlert  bool     `json:"need_alert"`
		AlertLevel string   `json:"alert_level"`
		RootCause  string   `json:"root_cause"`
		Impact     []string `json:"impact"`
		Solutions  []string `json:"solutions"`
		Prevention []string `json:"prevention"`
	}
	
	// 提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := response[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &jsonResult); err == nil {
			result.NeedAlert = jsonResult.NeedAlert
			result.AlertLevel = jsonResult.AlertLevel
			result.RootCause = jsonResult.RootCause
			result.Impact = jsonResult.Impact
			result.Solutions = jsonResult.Solutions
			result.Prevention = jsonResult.Prevention
			return result
		}
	}
	
	// 如果JSON解析失败，尝试文本解析
	result.RootCause = s.extractSection(response, "root_cause", "impact")
	result.Impact = s.extractList(response, "impact", "solutions")
	result.Solutions = s.extractList(response, "solutions", "prevention")
	result.Prevention = s.extractList(response, "prevention", "")
	
	// 判断是否需要告警
	if strings.Contains(strings.ToLower(response), "严重") || 
	   strings.Contains(strings.ToLower(response), "critical") {
		result.NeedAlert = true
		result.AlertLevel = "critical"
	} else if strings.Contains(strings.ToLower(response), "警告") || 
	          strings.Contains(strings.ToLower(response), "warning") {
		result.NeedAlert = true
		result.AlertLevel = "warning"
	}
	
	return result
}

// extractSection 提取章节内容
func (s *AIService) extractSection(text, start, end string) string {
	startIdx := strings.Index(strings.ToLower(text), strings.ToLower(start))
	if startIdx == -1 {
		return ""
	}
	
	text = text[startIdx+len(start):]
	
	if end != "" {
		endIdx := strings.Index(strings.ToLower(text), strings.ToLower(end))
		if endIdx != -1 {
			text = text[:endIdx]
		}
	}
	
	// 清理文本
	text = strings.TrimSpace(text)
	text = strings.Trim(text, ":")
	text = strings.Trim(text, "\"")
	text = strings.TrimSpace(text)
	
	return text
}

// extractList 提取列表
func (s *AIService) extractList(text, start, end string) []string {
	section := s.extractSection(text, start, end)
	if section == "" {
		return []string{}
	}
	
	lines := strings.Split(section, "\n")
	items := []string{}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 移除列表标记
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimPrefix(line, "•")
		line = strings.TrimPrefix(line, "\"")
		line = strings.TrimSuffix(line, "\"")
		line = strings.TrimSuffix(line, ",")
		
		// 移除数字标记
		if len(line) > 2 && line[1] == '.' {
			line = line[2:]
		}
		
		line = strings.TrimSpace(line)
		
		if line != "" && !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "}") {
			items = append(items, line)
		}
	}
	
	return items
}

// calculateConfidence 计算置信度
func (s *AIService) calculateConfidence(result *LogAnalysisResult, errorCount, warningCount int) float64 {
	confidence := 0.5 // 基础置信度
	
	// 日志数量越多，置信度越高
	if result.LogCount > 20 {
		confidence += 0.2
	} else if result.LogCount > 10 {
		confidence += 0.1
	}
	
	// 有明确的故障原因，置信度提高
	if result.RootCause != "" && len(result.RootCause) > 20 {
		confidence += 0.2
	}
	
	// 有解决方案，置信度提高
	if len(result.Solutions) > 0 {
		confidence += 0.1
	}
	
	// 有错误日志，置信度提高
	if errorCount > 0 {
		confidence += 0.1
	}
	
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// generateID 生成ID
func generateID() string {
	return fmt.Sprintf("analysis_%d", time.Now().UnixNano())
}
