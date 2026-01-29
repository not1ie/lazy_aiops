package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AIService 核心AI服务，供所有插件使用
type AIService struct {
	provider string
	apiKey   string
	baseURL  string
	model    string
}

func NewAIService(provider, apiKey, baseURL, model string) *AIService {
	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "ollama":
			baseURL = "http://localhost:11434/api"
		}
	}
	return &AIService{
		provider: provider,
		apiKey:   apiKey,
		baseURL:  baseURL,
		model:    model,
	}
}

// CallLLM 调用底层大模型
func (s *AIService) CallLLM(systemPrompt string, messages []map[string]string) (string, int, error) {
	allMessages := make([]map[string]string, 0)
	if systemPrompt != "" {
		allMessages = append(allMessages, map[string]string{"role": "system", "content": systemPrompt})
	}
	allMessages = append(allMessages, messages...)

	return s.doRequest(allMessages)
}

// CallSimple 调用简单文本 Prompt
func (s *AIService) CallSimple(prompt string) (string, int, error) {
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	return s.CallLLM("你是一个专业的运维工程师AI助手，请用中文回答。", messages)
}

func (s *AIService) doRequest(messages []map[string]string) (string, int, error) {
	if s.apiKey == "" && s.provider != "ollama" {
		return "AI服务未配置 api_key。", 0, nil
	}

	reqBody := map[string]interface{}{
		"model":    s.model,
		"messages": messages,
	}

	jsonData, _ := json.Marshal(reqBody)
	url := s.baseURL + "/chat/completions"
	if s.provider == "ollama" && !strings.HasSuffix(url, "/chat") {
		url = s.baseURL + "/chat"
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

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
		Message string `json:"message"` // Ollama style error
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", 0, err
	}

	if result.Error.Message != "" {
		return "", 0, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		if result.Message != "" {
			return "", 0, fmt.Errorf("AI error: %s", result.Message)
		}
		return "", 0, fmt.Errorf("no response from AI")
	}

	return result.Choices[0].Message.Content, result.Usage.TotalTokens, nil
}
