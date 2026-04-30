package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AIService 核心AI服务，供所有插件使用
type AIService struct {
	mu            sync.RWMutex
	provider      string
	apiKey        string
	baseURL       string
	model         string
	authType      string
	extraHeaders  map[string]string
	timeoutSecond int
}

type AIConfigSnapshot struct {
	Provider      string            `json:"provider"`
	BaseURL       string            `json:"base_url"`
	Model         string            `json:"model"`
	AuthType      string            `json:"auth_type"`
	TimeoutSecond int               `json:"timeout_second"`
	ExtraHeaders  map[string]string `json:"extra_headers"`
	HasAPIKey     bool              `json:"has_api_key"`
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
		provider:      provider,
		apiKey:        apiKey,
		baseURL:       baseURL,
		model:         model,
		authType:      "bearer",
		extraHeaders:  map[string]string{},
		timeoutSecond: 60,
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

func (s *AIService) UpdateConfig(provider, apiKey, baseURL, model, authType string, extraHeaders map[string]string, timeoutSecond int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider = strings.TrimSpace(provider)
	baseURL = strings.TrimSpace(baseURL)
	model = strings.TrimSpace(model)
	authType = strings.TrimSpace(strings.ToLower(authType))
	if provider == "" {
		provider = "openai"
	}
	if baseURL == "" {
		switch provider {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "ollama":
			baseURL = "http://localhost:11434/api"
		}
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	if authType == "" {
		authType = "bearer"
	}
	if timeoutSecond <= 0 {
		timeoutSecond = 60
	}

	s.provider = provider
	s.apiKey = apiKey
	s.baseURL = baseURL
	s.model = model
	s.authType = authType
	s.timeoutSecond = timeoutSecond
	s.extraHeaders = map[string]string{}
	for k, v := range extraHeaders {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		s.extraHeaders[key] = val
	}
}

func (s *AIService) SnapshotConfig() AIConfigSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	headers := map[string]string{}
	for k, v := range s.extraHeaders {
		headers[k] = v
	}
	return AIConfigSnapshot{
		Provider:      s.provider,
		BaseURL:       s.baseURL,
		Model:         s.model,
		AuthType:      s.authType,
		TimeoutSecond: s.timeoutSecond,
		ExtraHeaders:  headers,
		HasAPIKey:     strings.TrimSpace(s.apiKey) != "",
	}
}

func (s *AIService) doRequest(messages []map[string]string) (string, int, error) {
	s.mu.RLock()
	provider := s.provider
	apiKey := s.apiKey
	baseURL := s.baseURL
	model := s.model
	authType := s.authType
	timeoutSecond := s.timeoutSecond
	extraHeaders := map[string]string{}
	for k, v := range s.extraHeaders {
		extraHeaders[k] = v
	}
	s.mu.RUnlock()

	if provider == "gemini" || strings.Contains(strings.ToLower(baseURL), "generativelanguage.googleapis.com") {
		return s.doGeminiRequest(messages, provider, apiKey, baseURL, model, authType, timeoutSecond, extraHeaders)
	}

	if authType == "bearer" && apiKey == "" && provider != "ollama" {
		return "AI服务未配置 api_key。", 0, nil
	}
	if timeoutSecond <= 0 {
		timeoutSecond = 60
	}

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	jsonData, _ := json.Marshal(reqBody)
	url := strings.TrimRight(baseURL, "/")
	if strings.Contains(url, "/chat/completions") || strings.HasSuffix(url, "/chat") {
		// keep as is
	} else if provider == "ollama" {
		url = url + "/chat"
	} else {
		url = url + "/chat/completions"
	}

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	switch authType {
	case "bearer":
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	case "x-api-key":
		if apiKey != "" {
			req.Header.Set("x-api-key", apiKey)
		}
	case "api-key":
		if apiKey != "" {
			req.Header.Set("api-key", apiKey)
		}
	case "none":
		// no-op
	default:
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSecond) * time.Second,
	}
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

func (s *AIService) doGeminiRequest(messages []map[string]string, provider, apiKey, baseURL, model, authType string, timeoutSecond int, extraHeaders map[string]string) (string, int, error) {
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.5-flash"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if timeoutSecond <= 0 {
		timeoutSecond = 60
	}
	if strings.TrimSpace(apiKey) == "" {
		return "", 0, fmt.Errorf("gemini API key 未配置")
	}

	var textBuilder strings.Builder
	for _, msg := range messages {
		role := strings.ToUpper(strings.TrimSpace(msg["role"]))
		if role == "" {
			role = "USER"
		}
		textBuilder.WriteString("[" + role + "]\n")
		textBuilder.WriteString(strings.TrimSpace(msg["content"]))
		textBuilder.WriteString("\n\n")
	}
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": strings.TrimSpace(textBuilder.String())},
				},
			},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	endpoint := strings.TrimSpace(baseURL)
	if !strings.Contains(endpoint, ":generateContent") {
		endpoint = strings.TrimRight(endpoint, "/")
		modelPath := strings.TrimSpace(model)
		if !strings.HasPrefix(modelPath, "models/") {
			modelPath = "models/" + modelPath
		}
		endpoint = endpoint + "/" + modelPath + ":generateContent"
	}

	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	if authType == "x-api-key" {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	q := req.URL.Query()
	if q.Get("key") == "" {
		q.Set("key", apiKey)
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{Timeout: time.Duration(timeoutSecond) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", 0, err
	}
	if result.Error.Message != "" {
		return "", 0, fmt.Errorf("gemini API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 {
		return "", 0, fmt.Errorf("gemini no response")
	}
	parts := result.Candidates[0].Content.Parts
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		if strings.TrimSpace(item.Text) != "" {
			out = append(out, strings.TrimSpace(item.Text))
		}
	}
	if len(out) == 0 {
		return "", 0, fmt.Errorf("gemini empty response")
	}
	return strings.Join(out, "\n"), result.UsageMetadata.TotalTokenCount, nil
}
