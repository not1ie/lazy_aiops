package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type AIService struct {
	db       *gorm.DB
	provider string
	apiKey   string
	baseURL  string
	model    string
}

func NewAIService(db *gorm.DB, provider, apiKey, baseURL, model string) *AIService {
	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "azure":
			// Azure需要自定义
		case "ollama":
			baseURL = "http://localhost:11434/api"
		}
	}
	return &AIService{
		db:       db,
		provider: provider,
		apiKey:   apiKey,
		baseURL:  baseURL,
		model:    model,
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

	// 调用LLM
	reply, tokens, err := s.callLLM(history, context)
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

	reply, _, err := s.callLLMSimple(prompt)
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

	reply, _, err := s.callLLMSimple(prompt)
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

	reply, _, err := s.callLLMSimple(prompt)
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

	reply, _, err := s.callLLMSimple(prompt)
	return reply, err
}

// callAI 调用AI进行分析（用于日志分析）
func (s *AIService) callAI(prompt string) (string, error) {
	reply, _, err := s.callLLMSimple(prompt)
	return reply, err
}

func (s *AIService) callLLM(history []ChatMessage, context string) (string, int, error) {
	messages := make([]map[string]string, 0)

	// 系统提示
	systemPrompt := "你是Lazy Auto Ops运维平台的AI助手，专注于帮助用户解决运维问题、分析日志、诊断故障。请用中文回答。"
	if context != "" {
		systemPrompt += "\n\n上下文信息:\n" + context
	}
	messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})

	// 历史消息
	for _, msg := range history {
		messages = append(messages, map[string]string{"role": msg.Role, "content": msg.Content})
	}

	return s.doRequest(messages)
}

func (s *AIService) callLLMSimple(prompt string) (string, int, error) {
	messages := []map[string]string{
		{"role": "system", "content": "你是一个专业的运维工程师AI助手，请用中文回答。"},
		{"role": "user", "content": prompt},
	}
	return s.doRequest(messages)
}

func (s *AIService) doRequest(messages []map[string]string) (string, int, error) {
	if s.apiKey == "" {
		// 没有配置API Key时返回模拟响应
		return "AI服务未配置，请在配置文件中设置 api_key。\n\n示例配置:\n```yaml\nplugins:\n  ai:\n    enabled: true\n    config:\n      provider: openai\n      api_key: your-api-key\n      model: gpt-3.5-turbo\n```", 0, nil
	}

	reqBody := map[string]interface{}{
		"model":    s.model,
		"messages": messages,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", 0, err
	}

	if result.Error.Message != "" {
		return "", 0, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from AI")
	}

	return result.Choices[0].Message.Content, result.Usage.TotalTokens, nil
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
		}, nil
	}
	return &resp, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
