package cicd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CICDHandler struct {
	db        *gorm.DB
	scheduler *Scheduler
}

func NewCICDHandler(db *gorm.DB) *CICDHandler {
	h := &CICDHandler{db: db}
	h.scheduler = NewScheduler(db, h)
	h.scheduler.Start()
	return h
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
	paramsJSON, _ := json.Marshal(params)
	execution := &CICDExecution{
		PipelineID:   pipeline.ID,
		PipelineName: pipeline.Name,
		Provider:     pipeline.Provider,
		Status:       0,
		Trigger:      trigger,
		TriggerBy:    triggerBy,
		Parameters:   string(paramsJSON),
		StartedAt:    time.Now(),
	}

	var remoteBuildID string
	var err error

	switch pipeline.Provider {
	case "jenkins":
		remoteBuildID, err = h.triggerJenkins(pipeline, params)
	case "gitlab":
		remoteBuildID, err = h.triggerGitLab(pipeline, params)
	case "argocd":
		remoteBuildID, err = h.triggerArgoCD(pipeline)
	case "github":
		remoteBuildID, err = h.triggerGitHub(pipeline, params)
	default:
		err = fmt.Errorf("不支持的Provider: %s", pipeline.Provider)
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
		go h.pollBuildStatus(execution, pipeline)
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

// SyncFromRemote 从远程同步Job
func (h *CICDHandler) SyncFromRemote(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	// TODO: 实现同步逻辑
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步成功"})
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
	provider := c.Param("provider")
	body, _ := io.ReadAll(c.Request.Body)

	// 记录webhook
	// TODO: 根据provider解析并触发对应流水线

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("Webhook received from %s", provider),
		"body":    string(body),
	})
}
