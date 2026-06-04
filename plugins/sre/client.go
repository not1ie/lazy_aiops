package sre

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SreClient 是访问 lazysre HTTP 服务的客户端
// lazysre 真实 API 基于 /v1/platform/* 和 /v1/tasks/*
type SreClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSreClient 创建新的 SRE 客户端
func NewSreClient(baseURL string) *SreClient {
	return &SreClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Ping 健康检查 -> GET /health
func (c *SreClient) Ping(ctx context.Context) (*SrePingResponse, error) {
	resp, err := c.get(ctx, "/health")
	if err != nil {
		return nil, err
	}
	var result SrePingResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return &SrePingResponse{Status: "ok"}, nil
	}
	return &result, nil
}

// GetOverview 获取平台概览 -> GET /v1/platform/overview
func (c *SreClient) GetOverview(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/overview")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析概览失败: %w", err)
	}
	return result, nil
}

// GetBriefing 获取环境快速总览 -> GET /v1/platform/briefing
func (c *SreClient) GetBriefing(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/briefing")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析 briefing 失败: %w", err)
	}
	return result, nil
}

// ListTools 获取可用工具列表 -> GET /v1/platform/tools
func (c *SreClient) ListTools(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/tools")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析工具列表失败: %w", err)
	}
	return result, nil
}

// ListWorkflows 获取 Workflow/Skill 列表 -> GET /v1/platform/workflows
func (c *SreClient) ListWorkflows(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/workflows")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析 workflow 列表失败: %w", err)
	}
	return result, nil
}

// ListTemplates 获取模板列表 -> GET /v1/platform/templates
func (c *SreClient) ListTemplates(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/templates")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析模板列表失败: %w", err)
	}
	return result, nil
}

// QuickStart 自然语言快速启动任务 -> POST /v1/platform/quickstart
func (c *SreClient) QuickStart(ctx context.Context, req *SreDiagnoseRequest) (map[string]interface{}, error) {
	// 转换为 lazysre quickstart 格式
	payload := map[string]interface{}{
		"instruction": req.Query,
		"context":     req.Context,
	}
	if req.ContextHint != nil {
		payload["context_hint"] = req.ContextHint
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, "/v1/platform/quickstart", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析 quickstart 结果失败: %w", err)
	}
	return result, nil
}

// ListRuns 获取执行记录列表 -> GET /v1/platform/runs
func (c *SreClient) ListRuns(ctx context.Context, limit int) (map[string]interface{}, error) {
	url := fmt.Sprintf("/v1/platform/runs?limit=%d", limit)
	resp, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析执行记录失败: %w", err)
	}
	return result, nil
}

// GetRun 获取单次执行详情 -> GET /v1/platform/runs/{run_id}
func (c *SreClient) GetRun(ctx context.Context, runID string) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/runs/"+runID)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析执行详情失败: %w", err)
	}
	return result, nil
}

// ApproveRun 审批执行 -> POST /v1/platform/runs/{run_id}/approval
func (c *SreClient) ApproveRun(ctx context.Context, runID string, req *SreApproveRequest) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"approved": req.Approved,
		"comment":  req.Comment,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := c.post(ctx, "/v1/platform/runs/"+runID+"/approval", body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析审批结果失败: %w", err)
	}
	return result, nil
}

// ListTasks 获取任务列表 -> GET /v1/tasks
func (c *SreClient) ListTasks(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/tasks")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析任务列表失败: %w", err)
	}
	return result, nil
}

// GetArtifacts 获取产物列表 -> GET /v1/platform/artifacts
func (c *SreClient) GetArtifacts(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/v1/platform/artifacts")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析产物列表失败: %w", err)
	}
	return result, nil
}

// IsAvailable 检查 lazysre 服务是否可用
func (c *SreClient) IsAvailable(ctx context.Context) bool {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := c.Ping(ctx2)
	return err == nil
}

// --- 内部 HTTP 辅助方法 ---

func (c *SreClient) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	return c.do(req)
}

func (c *SreClient) post(ctx context.Context, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path,
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *SreClient) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 lazysre 失败 [%s %s]: %w",
			req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("lazysre 返回错误 %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
