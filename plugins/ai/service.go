package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/gorm"
)

type AIService struct {
	db   *gorm.DB
	core *core.Core
}

func NewAIService(db *gorm.DB, c *core.Core) *AIService {
	return &AIService{
		db:   db,
		core: c,
	}
}

// Chat 对话
func (s *AIService) Chat(sessionID, userID, message, context string) (*ChatResponse, error) {
	// 获取或创建会话
	var session ChatSession
	if sessionID != "" {
		s.db.First(&session, "id = ?", sessionID)
	}
	if session.ID == "" {
		session = ChatSession{
			UserID:  userID,
			Title:   truncate(message, 50),
			Type:    "chat",
			Context: context,
		}
		s.db.Create(&session)
	}

	// 保存用户消息
	userMsg := ChatMessage{
		SessionID: session.ID,
		Role:      "user",
		Content:   message,
	}
	s.db.Create(&userMsg)

	// 获取历史消息构建上下文
	var history []ChatMessage
	s.db.Where("session_id = ?", session.ID).Order("created_at").Limit(20).Find(&history)

	// 调用核心LLM服务
	messages := make([]map[string]string, 0)
	for _, msg := range history {
		messages = append(messages, map[string]string{"role": msg.Role, "content": msg.Content})
	}

	systemPrompt := "你是Lazy Auto Ops运维平台的AI助手，专注于帮助用户解决运维问题、分析日志、诊断故障。请用中文回答。"
	if context != "" {
		systemPrompt += "\n\n上下文信息:\n" + context
	}

	reply, tokens, err := s.core.AI.CallLLM(systemPrompt, messages)
	if err != nil {
		return nil, err
	}

	// 保存助手回复
	assistantMsg := ChatMessage{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   reply,
		TokenUsed: tokens,
	}
	s.db.Create(&assistantMsg)

	return &ChatResponse{
		SessionID: session.ID,
		Reply:     reply,
		TokenUsed: tokens,
	}, nil
}

// AnalyzeLogs 分析日志
func (s *AIService) AnalyzeLogs(logs, context string) (*AnalyzeResponse, error) {
	prompt := fmt.Sprintf(`你是一个专业的运维工程师，请分析以下日志内容，找出潜在问题并给出建议。

日志内容:
%s

%s

请以JSON格式返回分析结果，包含以下字段:
- summary: 简要总结
- issues: 问题列表，每个问题包含 type, description, location, severity
- suggestions: 建议列表
- severity: 整体严重程度 (critical/warning/info)`, logs, context)

	reply, _, err := s.core.AI.CallSimple(prompt)
	if err != nil {
		return nil, err
	}

	return s.parseAnalyzeResponse(reply)
}

// AnalyzeError 分析错误
func (s *AIService) AnalyzeError(errorMsg, stackTrace, context string) (*AnalyzeResponse, error) {
	prompt := fmt.Sprintf(`你是一个专业的运维工程师，请分析以下错误信息，找出根本原因并给出修复建议。

错误信息:
%s

堆栈跟踪:
%s

%s

请以JSON格式返回分析结果。`, errorMsg, stackTrace, context)

	reply, _, err := s.core.AI.CallSimple(prompt)
	if err != nil {
		return nil, err
	}

	return s.parseAnalyzeResponse(reply)
}

// SuggestFix 建议修复方案
func (s *AIService) SuggestFix(issue, context string) (string, error) {
	prompt := fmt.Sprintf(`你是一个专业的运维工程师，请针对以下问题给出具体的修复方案和命令。

问题描述:
%s

%s

请给出详细的修复步骤和相关命令。`, issue, context)

	reply, _, err := s.core.AI.CallSimple(prompt)
	return reply, err
}

// SuggestOptimize 建议优化方案
func (s *AIService) SuggestOptimize(target, metrics, context string) (string, error) {
	prompt := fmt.Sprintf(`你是一个专业的运维工程师，请针对以下系统/服务给出优化建议。

优化目标:
%s

当前指标:
%s

%s

请给出具体的优化建议和配置调整。`, target, metrics, context)

	reply, _, err := s.core.AI.CallSimple(prompt)
	return reply, err
}

// callAI 内部辅助方法，调用AI进行分析
func (s *AIService) callAI(prompt string) (string, error) {
	reply, _, err := s.core.AI.CallSimple(prompt)
	return reply, err
}

func (s *AIService) parseAnalyzeResponse(reply string) (*AnalyzeResponse, error) {
	// 尝试从回复中提取JSON
	reply = strings.TrimSpace(reply)
	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start >= 0 && end > start {
		reply = reply[start : end+1]
	}

	var resp AnalyzeResponse
	if err := json.Unmarshal([]byte(reply), &resp); err != nil {
		// 解析失败时返回原始文本
		return &AnalyzeResponse{
			Summary:     reply,
			Severity:    "info",
			Suggestions: []string{},
			Issues:      []Issue{},
		},
		 nil
	}
	return &resp, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}