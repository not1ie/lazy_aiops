package cicd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/gorm"
)

type CICDHandler struct {
	db        *gorm.DB
	secretKey string
	scheduler *Scheduler
}

func NewCICDHandler(db *gorm.DB, secretKey string) *CICDHandler {
	h := &CICDHandler{db: db, secretKey: secretKey}
	h.scheduler = NewScheduler(db, h)
	h.scheduler.Start()
	return h
}

type CICDCredentialOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Username string `json:"username"`
}

func pickCredentialSecret(cred *cmdb.Credential) string {
	if cred == nil {
		return ""
	}
	candidates := []string{cred.Password, cred.SecretKey, cred.AccessKey, cred.Passphrase}
	for _, item := range candidates {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}

func (h *CICDHandler) applyCredential(pipeline *CICDPipeline) error {
	if pipeline == nil || strings.TrimSpace(pipeline.CredentialID) == "" {
		return nil
	}

	var cred cmdb.Credential
	if err := h.db.First(&cred, "id = ?", pipeline.CredentialID).Error; err != nil {
		return fmt.Errorf("凭据不存在或不可用")
	}
	if err := cmdb.DecryptCredentialFields(h.secretKey, &cred); err != nil {
		return fmt.Errorf("凭据解密失败: %w", err)
	}

	token := pickCredentialSecret(&cred)
	switch strings.ToLower(strings.TrimSpace(pipeline.Provider)) {
	case "jenkins":
		if strings.TrimSpace(cred.Username) != "" {
			pipeline.JenkinsUser = cred.Username
		}
		if strings.TrimSpace(token) != "" {
			pipeline.JenkinsToken = token
		}
	case "gitlab":
		if strings.TrimSpace(token) != "" {
			pipeline.GitLabToken = token
		}
	case "argocd":
		if strings.TrimSpace(token) != "" {
			pipeline.ArgoCDToken = token
		}
	case "github":
		if strings.TrimSpace(token) != "" {
			pipeline.GitHubToken = token
		}
	}
	return nil
}

func (h *CICDHandler) attachCredentialNames(pipelines []CICDPipeline) error {
	if len(pipelines) == 0 {
		return nil
	}
	ids := make([]string, 0, len(pipelines))
	idSet := make(map[string]struct{}, len(pipelines))
	for i := range pipelines {
		id := strings.TrimSpace(pipelines[i].CredentialID)
		if id == "" {
			continue
		}
		if _, ok := idSet[id]; ok {
			continue
		}
		idSet[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	var creds []cmdb.Credential
	if err := h.db.Select("id", "name").Where("id IN ?", ids).Find(&creds).Error; err != nil {
		return err
	}
	nameMap := make(map[string]string, len(creds))
	for i := range creds {
		nameMap[creds[i].ID] = creds[i].Name
	}
	for i := range pipelines {
		if name, ok := nameMap[pipelines[i].CredentialID]; ok {
			pipelines[i].CredentialName = name
		}
	}
	return nil
}

func (h *CICDHandler) ListCredentials(c *gin.Context) {
	var creds []cmdb.Credential
	if err := h.db.Order("created_at DESC").Find(&creds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	options := make([]CICDCredentialOption, 0, len(creds))
	for i := range creds {
		options = append(options, CICDCredentialOption{
			ID:       creds[i].ID,
			Name:     creds[i].Name,
			Type:     creds[i].Type,
			Username: creds[i].Username,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": options})
}

// ListPipelines 流水线列表
func (h *CICDHandler) ListPipelines(c *gin.Context) {
	var pipelines []CICDPipeline
	query := h.db.Order("created_at DESC")
	if provider := c.Query("provider"); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.attachCredentialNames(pipelines); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipelines})
}

// CreatePipeline 创建流水线
func (h *CICDHandler) CreatePipeline(c *gin.Context) {
	var pipeline CICDPipeline
	if err := c.ShouldBindJSON(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// GetPipeline 获取流水线详情
func (h *CICDHandler) GetPipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	if strings.TrimSpace(pipeline.CredentialID) != "" {
		var cred cmdb.Credential
		if err := h.db.Select("id", "name").First(&cred, "id = ?", pipeline.CredentialID).Error; err == nil {
			pipeline.CredentialName = cred.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// UpdatePipeline 更新流水线
func (h *CICDHandler) UpdatePipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	if err := c.ShouldBindJSON(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// DeletePipeline 删除流水线
func (h *CICDHandler) DeletePipeline(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDPipeline{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TriggerPipeline 触发流水线
func (h *CICDHandler) TriggerPipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	var req struct {
		Parameters map[string]string `json:"parameters"`
	}
	c.ShouldBindJSON(&req)

	execution, err := h.triggerBuild(&pipeline, req.Parameters, "manual", c.GetString("username"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// triggerBuild 触发构建
func (h *CICDHandler) triggerBuild(pipeline *CICDPipeline, params map[string]string, trigger, triggerBy string) (*CICDExecution, error) {
	resolvedPipeline := *pipeline
	if err := h.applyCredential(&resolvedPipeline); err != nil {
		return nil, err
	}

	paramsJSON, _ := json.Marshal(params)
	execution := &CICDExecution{
		PipelineID:   resolvedPipeline.ID,
		PipelineName: resolvedPipeline.Name,
		Provider:     resolvedPipeline.Provider,
		Status:       0,
		Trigger:      trigger,
		TriggerBy:    triggerBy,
		Parameters:   string(paramsJSON),
		StartedAt:    time.Now(),
	}

	var remoteBuildID string
	var err error

	switch resolvedPipeline.Provider {
	case "jenkins":
		remoteBuildID, err = h.triggerJenkins(&resolvedPipeline, params)
	case "gitlab":
		remoteBuildID, err = h.triggerGitLab(&resolvedPipeline, params)
	case "argocd":
		remoteBuildID, err = h.triggerArgoCD(&resolvedPipeline)
	case "github":
		remoteBuildID, err = h.triggerGitHub(&resolvedPipeline, params)
	default:
		err = fmt.Errorf("不支持的Provider: %s", resolvedPipeline.Provider)
	}

	if err != nil {
		execution.Status = 2
		execution.Logs = err.Error()
	} else {
		execution.RemoteBuildID = remoteBuildID
	}

	h.db.Create(execution)

	// 异步轮询状态
	if err == nil {
		go h.pollBuildStatus(execution, &resolvedPipeline)
	}

	return execution, err
}

// triggerJenkins 触发Jenkins构建
func (h *CICDHandler) triggerJenkins(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("%s/job/%s/build", strings.TrimSuffix(pipeline.JenkinsURL, "/"), pipeline.JenkinsJob)
	if len(params) > 0 {
		url = fmt.Sprintf("%s/job/%s/buildWithParameters", strings.TrimSuffix(pipeline.JenkinsURL, "/"), pipeline.JenkinsJob)
	}

	req, _ := http.NewRequest("POST", url, nil)
	req.SetBasicAuth(pipeline.JenkinsUser, pipeline.JenkinsToken)

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Jenkins返回错误: %d, %s", resp.StatusCode, string(body))
	}

	// 获取队列ID
	queueURL := resp.Header.Get("Location")
	if queueURL != "" {
		return queueURL, nil
	}
	return "triggered", nil
}

// triggerGitLab 触发GitLab Pipeline
func (h *CICDHandler) triggerGitLab(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipeline",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), pipeline.GitLabProjectID)

	payload := map[string]interface{}{
		"ref": pipeline.GitLabRef,
	}
	if len(params) > 0 {
		variables := make([]map[string]string, 0)
		for k, v := range params {
			variables = append(variables, map[string]string{"key": k, "value": v})
		}
		payload["variables"] = variables
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID int `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID == 0 {
		return "", fmt.Errorf("GitLab触发失败")
	}
	return fmt.Sprintf("%d", result.ID), nil
}

// triggerArgoCD 触发ArgoCD同步
func (h *CICDHandler) triggerArgoCD(pipeline *CICDPipeline) (string, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync",
		strings.TrimSuffix(pipeline.ArgoCDURL, "/"), pipeline.ArgoCDApp)

	req, _ := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+pipeline.ArgoCDToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ArgoCD同步失败: %d", resp.StatusCode)
	}
	return "sync-" + time.Now().Format("20060102150405"), nil
}

// triggerGitHub 触发GitHub Actions
func (h *CICDHandler) triggerGitHub(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches",
		pipeline.GitHubRepo, pipeline.GitHubWorkflow)

	payload := map[string]interface{}{
		"ref": "main",
	}
	if len(params) > 0 {
		payload["inputs"] = params
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "token "+pipeline.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GitHub Actions触发失败: %d", resp.StatusCode)
	}
	return "dispatch-" + time.Now().Format("20060102150405"), nil
}

// pollBuildStatus 轮询构建状态
func (h *CICDHandler) pollBuildStatus(execution *CICDExecution, pipeline *CICDPipeline) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 360; i++ { // 最多1小时
		<-ticker.C

		var status int
		var logs string

		switch pipeline.Provider {
		case "jenkins":
			status, logs = h.getJenkinsStatus(pipeline, execution.RemoteBuildID)
		case "gitlab":
			status, logs = h.getGitLabStatus(pipeline, execution.RemoteBuildID)
		default:
			status = 1
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status": status,
			"logs":   logs,
		}
		if status != 0 {
			updates["finished_at"] = now
			updates["duration"] = int(now.Sub(execution.StartedAt).Seconds())
		}
		h.db.Model(execution).Updates(updates)

		if status != 0 {
			return
		}
	}
}

func (h *CICDHandler) getJenkinsStatus(pipeline *CICDPipeline, queueURL string) (int, string) {
	// 简化实现
	return 1, "Build completed"
}

func (h *CICDHandler) getGitLabStatus(pipeline *CICDPipeline, pipelineID string) (int, string) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipelines/%s",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), pipeline.GitLabProjectID, pipelineID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	switch result.Status {
	case "success":
		return 1, "Pipeline success"
	case "failed":
		return 2, "Pipeline failed"
	case "canceled":
		return 3, "Pipeline canceled"
	default:
		return 0, "Running: " + result.Status
	}
}

func (h *CICDHandler) upsertCICDJob(job *CICDJob) error {
	var existing CICDJob
	err := h.db.Where("pipeline_id = ? AND remote_id = ?", job.PipelineID, job.RemoteID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.db.Create(job).Error
		}
		return err
	}

	existing.Name = job.Name
	existing.LastBuildNo = job.LastBuildNo
	existing.LastStatus = job.LastStatus
	existing.LastBuildAt = job.LastBuildAt
	return h.db.Save(&existing).Error
}

func (h *CICDHandler) syncJenkinsJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.JenkinsURL) == "" || strings.TrimSpace(pipeline.JenkinsJob) == "" {
		return nil, fmt.Errorf("Jenkins配置不完整")
	}

	apiURL := fmt.Sprintf("%s/job/%s/api/json",
		strings.TrimSuffix(pipeline.JenkinsURL, "/"),
		url.PathEscape(pipeline.JenkinsJob))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.JenkinsUser != "" || pipeline.JenkinsToken != "" {
		req.SetBasicAuth(pipeline.JenkinsUser, pipeline.JenkinsToken)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jenkins同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		Name      string `json:"name"`
		Color     string `json:"color"`
		LastBuild struct {
			Number    int   `json:"number"`
			Timestamp int64 `json:"timestamp"`
		} `json:"lastBuild"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	lastStatus := "unknown"
	color := strings.ToLower(result.Color)
	switch {
	case strings.Contains(color, "blue") || strings.Contains(color, "green"):
		lastStatus = "success"
	case strings.Contains(color, "red"):
		lastStatus = "failed"
	case strings.Contains(color, "disabled"):
		lastStatus = "disabled"
	case strings.Contains(color, "anime"):
		lastStatus = "running"
	}

	var lastBuildAt *time.Time
	if result.LastBuild.Timestamp > 0 {
		t := time.UnixMilli(result.LastBuild.Timestamp)
		lastBuildAt = &t
	}

	name := strings.TrimSpace(result.Name)
	if name == "" {
		name = pipeline.JenkinsJob
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.JenkinsJob,
		LastBuildNo: result.LastBuild.Number,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncGitLabJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.GitLabURL) == "" || strings.TrimSpace(pipeline.GitLabProjectID) == "" {
		return nil, fmt.Errorf("GitLab配置不完整")
	}

	projectID := url.PathEscape(pipeline.GitLabProjectID)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/pipelines?per_page=1",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), projectID)
	if ref := strings.TrimSpace(pipeline.GitLabRef); ref != "" {
		apiURL += "&ref=" + url.QueryEscape(ref)
	}

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var runs []struct {
		ID        int    `json:"id"`
		Status    string `json:"status"`
		Ref       string `json:"ref"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&runs); err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return &CICDJob{
			PipelineID: pipeline.ID,
			Name:       pipeline.Name,
			RemoteID:   pipeline.GitLabProjectID,
			LastStatus: "unknown",
		}, nil
	}

	run := runs[0]
	lastStatus := strings.TrimSpace(run.Status)
	if lastStatus == "" {
		lastStatus = "unknown"
	}
	var lastBuildAt *time.Time
	for _, raw := range []string{run.UpdatedAt, run.CreatedAt} {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			lastBuildAt = &t
			break
		}
	}

	name := pipeline.Name
	if strings.TrimSpace(run.Ref) != "" {
		name = fmt.Sprintf("%s (%s)", pipeline.Name, run.Ref)
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.GitLabProjectID,
		LastBuildNo: run.ID,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncGitHubJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.GitHubRepo) == "" || strings.TrimSpace(pipeline.GitHubWorkflow) == "" {
		return nil, fmt.Errorf("GitHub配置不完整")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/runs?per_page=1",
		strings.TrimSpace(pipeline.GitHubRepo), url.PathEscape(strings.TrimSpace(pipeline.GitHubWorkflow)))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+pipeline.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		WorkflowRuns []struct {
			ID         int64  `json:"id"`
			RunNumber  int    `json:"run_number"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			UpdatedAt  string `json:"updated_at"`
			CreatedAt  string `json:"created_at"`
		} `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.WorkflowRuns) == 0 {
		return &CICDJob{
			PipelineID: pipeline.ID,
			Name:       pipeline.Name,
			RemoteID:   pipeline.GitHubWorkflow,
			LastStatus: "unknown",
		}, nil
	}

	run := result.WorkflowRuns[0]
	status := strings.TrimSpace(run.Conclusion)
	if status == "" {
		status = strings.TrimSpace(run.Status)
	}
	if status == "" {
		status = "unknown"
	}

	var lastBuildAt *time.Time
	for _, raw := range []string{run.UpdatedAt, run.CreatedAt} {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			lastBuildAt = &t
			break
		}
	}

	name := strings.TrimSpace(run.Name)
	if name == "" {
		name = pipeline.Name
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.GitHubWorkflow,
		LastBuildNo: run.RunNumber,
		LastStatus:  status,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncArgoCDJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.ArgoCDURL) == "" || strings.TrimSpace(pipeline.ArgoCDApp) == "" {
		return nil, fmt.Errorf("ArgoCD配置不完整")
	}

	apiURL := fmt.Sprintf("%s/api/v1/applications/%s",
		strings.TrimSuffix(pipeline.ArgoCDURL, "/"), url.PathEscape(strings.TrimSpace(pipeline.ArgoCDApp)))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.ArgoCDToken != "" {
		req.Header.Set("Authorization", "Bearer "+pipeline.ArgoCDToken)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ArgoCD同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			OperationState struct {
				Phase      string `json:"phase"`
				FinishedAt string `json:"finishedAt"`
			} `json:"operationState"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	lastStatus := strings.TrimSpace(result.Status.OperationState.Phase)
	if lastStatus == "" {
		lastStatus = strings.TrimSpace(result.Status.Sync.Status)
	}
	if lastStatus == "" {
		lastStatus = strings.TrimSpace(result.Status.Health.Status)
	}
	if lastStatus == "" {
		lastStatus = "unknown"
	}

	var lastBuildAt *time.Time
	if t, err := time.Parse(time.RFC3339, result.Status.OperationState.FinishedAt); err == nil {
		lastBuildAt = &t
	}

	name := strings.TrimSpace(result.Metadata.Name)
	if name == "" {
		name = pipeline.ArgoCDApp
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.ArgoCDApp,
		LastBuildNo: 0,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

// SyncFromRemote 从远程同步Job
func (h *CICDHandler) SyncFromRemote(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	if err := h.applyCredential(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var (
		job *CICDJob
		err error
	)

	switch strings.ToLower(strings.TrimSpace(pipeline.Provider)) {
	case "jenkins":
		job, err = h.syncJenkinsJob(&pipeline)
	case "gitlab":
		job, err = h.syncGitLabJob(&pipeline)
	case "github":
		job, err = h.syncGitHubJob(&pipeline)
	case "argocd":
		job, err = h.syncArgoCDJob(&pipeline)
	default:
		err = fmt.Errorf("不支持的Provider: %s", pipeline.Provider)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "同步失败：未获取到Job信息"})
		return
	}
	if err := h.upsertCICDJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步成功", "data": job})
}

// ListExecutions 执行记录列表
func (h *CICDHandler) ListExecutions(c *gin.Context) {
	var executions []CICDExecution
	query := h.db.Order("started_at DESC")
	if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
		query = query.Where("pipeline_id = ?", pipelineID)
	}
	if err := query.Limit(100).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": executions})
}

// GetExecution 获取执行详情
func (h *CICDHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var execution CICDExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// GetExecutionLogs 获取执行日志
func (h *CICDHandler) GetExecutionLogs(c *gin.Context) {
	id := c.Param("id")
	var execution CICDExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"logs": execution.Logs}})
}

// CancelExecution 取消执行
func (h *CICDHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&CICDExecution{}).Where("id = ? AND status = 0", id).Update("status", 3)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
}

// ListSchedules 定时发布列表
func (h *CICDHandler) ListSchedules(c *gin.Context) {
	var schedules []CICDSchedule
	if err := h.db.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedules})
}

// CreateSchedule 创建定时发布
func (h *CICDHandler) CreateSchedule(c *gin.Context) {
	var schedule CICDSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// UpdateSchedule 更新定时发布
func (h *CICDHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule CICDSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "定时任务不存在"})
		return
	}
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// DeleteSchedule 删除定时发布
func (h *CICDHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDSchedule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ToggleSchedule 启用/禁用定时发布
func (h *CICDHandler) ToggleSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule CICDSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "定时任务不存在"})
		return
	}
	schedule.Enabled = !schedule.Enabled
	h.db.Save(&schedule)
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// ListReleases 发布记录列表
func (h *CICDHandler) ListReleases(c *gin.Context) {
	var releases []CICDRelease
	query := h.db.Order("created_at DESC")
	if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
		query = query.Where("pipeline_id = ?", pipelineID)
	}
	if err := query.Find(&releases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": releases})
}

// CreateRelease 创建发布记录
func (h *CICDHandler) CreateRelease(c *gin.Context) {
	var release CICDRelease
	if err := c.ShouldBindJSON(&release); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	release.Operator = c.GetString("username")
	if release.Status == 1 {
		now := time.Now()
		release.ReleaseAt = &now
	}
	if err := h.db.Create(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": release})
}

// UpdateRelease 更新发布记录
func (h *CICDHandler) UpdateRelease(c *gin.Context) {
	id := c.Param("id")
	var release CICDRelease
	if err := h.db.First(&release, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布记录不存在"})
		return
	}
	if err := c.ShouldBindJSON(&release); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if release.Status == 1 && release.ReleaseAt == nil {
		now := time.Now()
		release.ReleaseAt = &now
	}
	if err := h.db.Save(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": release})
}

// DeleteRelease 删除发布记录
func (h *CICDHandler) DeleteRelease(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDRelease{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// HandleWebhook 处理Webhook
func (h *CICDHandler) HandleWebhook(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	body, _ := io.ReadAll(c.Request.Body)

	event := parseWebhookEvent(provider, body, c.GetHeader("X-GitHub-Event"))
	if event.Actor == "" {
		event.Actor = "webhook:" + provider
	}

	var pipelines []CICDPipeline
	if err := h.db.Where("provider = ? AND status = ?", provider, 1).Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	triggered := make([]string, 0, len(pipelines))
	failed := make([]string, 0)
	skipped := 0
	for i := range pipelines {
		pipeline := &pipelines[i]
		if !event.matchesPipeline(pipeline) {
			skipped++
			continue
		}
		execution, err := h.triggerBuild(pipeline, event.Params, "webhook", event.Actor)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", pipeline.Name, err))
			continue
		}
		triggered = append(triggered, execution.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("Webhook received from %s", provider),
		"data": gin.H{
			"provider":   provider,
			"event":      event.Name,
			"ref":        event.Ref,
			"repo":       event.Repo,
			"actor":      event.Actor,
			"matched":    len(triggered),
			"failed":     len(failed),
			"skipped":    skipped,
			"executions": triggered,
			"errors":     failed,
		},
	})
}

type cicdWebhookEvent struct {
	Name   string
	Repo   string
	Ref    string
	Actor  string
	Params map[string]string
}

func parseWebhookEvent(provider string, body []byte, githubEvent string) cicdWebhookEvent {
	event := cicdWebhookEvent{
		Name:   strings.TrimSpace(githubEvent),
		Params: map[string]string{},
	}

	switch provider {
	case "github":
		var payload struct {
			Ref        string `json:"ref"`
			Workflow   string `json:"workflow"`
			Action     string `json:"action"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Sender struct {
				Login string `json:"login"`
			} `json:"sender"`
		}
		_ = json.Unmarshal(body, &payload)
		event.Ref = normalizeGitRef(payload.Ref)
		event.Repo = strings.TrimSpace(payload.Repository.FullName)
		event.Actor = strings.TrimSpace(payload.Sender.Login)
		if event.Name == "" {
			event.Name = strings.TrimSpace(payload.Action)
		}
		if wf := strings.TrimSpace(payload.Workflow); wf != "" {
			event.Params["workflow"] = wf
		}
	case "gitlab":
		var payload struct {
			ObjectKind string `json:"object_kind"`
			Ref        string `json:"ref"`
			Project    struct {
				PathWithNamespace string `json:"path_with_namespace"`
			} `json:"project"`
			UserUsername string `json:"user_username"`
		}
		_ = json.Unmarshal(body, &payload)
		event.Name = strings.TrimSpace(payload.ObjectKind)
		event.Ref = normalizeGitRef(payload.Ref)
		event.Repo = strings.TrimSpace(payload.Project.PathWithNamespace)
		event.Actor = strings.TrimSpace(payload.UserUsername)
	default:
		event.Name = provider
	}

	if event.Ref != "" {
		event.Params["ref"] = event.Ref
	}
	if event.Name != "" {
		event.Params["event"] = event.Name
	}
	return event
}

func (e cicdWebhookEvent) matchesPipeline(p *CICDPipeline) bool {
	if p == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(p.Provider)) {
	case "github":
		if e.Repo != "" && strings.TrimSpace(p.GitHubRepo) != "" && !strings.EqualFold(strings.TrimSpace(e.Repo), strings.TrimSpace(p.GitHubRepo)) {
			return false
		}
		if e.Ref != "" && strings.TrimSpace(p.GitHubWorkflow) != "" {
			// workflow 字段用于 API dispatch，需要时再按 workflow 过滤。
			if wf := strings.TrimSpace(e.Params["workflow"]); wf != "" && !strings.EqualFold(wf, strings.TrimSpace(p.GitHubWorkflow)) {
				return false
			}
		}
	case "gitlab":
		project := strings.TrimSpace(p.GitLabProjectID)
		if e.Repo != "" && project != "" && strings.Contains(project, "/") &&
			!strings.EqualFold(strings.TrimSpace(e.Repo), project) {
			return false
		}
		if e.Ref != "" && strings.TrimSpace(p.GitLabRef) != "" &&
			!strings.EqualFold(strings.TrimSpace(e.Ref), strings.TrimSpace(p.GitLabRef)) {
			return false
		}
	}
	return true
}

func normalizeGitRef(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "refs/heads/")
	raw = strings.TrimPrefix(raw, "refs/tags/")
	return raw
}
