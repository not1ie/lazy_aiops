package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type ExecutorHandler struct {
	db         *gorm.DB
	executions sync.Map // executionID -> cancel chan
}

func NewExecutorHandler(db *gorm.DB) *ExecutorHandler {
	return &ExecutorHandler{db: db}
}

// Execute 批量执行
func (h *ExecutorHandler) Execute(c *gin.Context) {
	var req struct {
		Name        string   `json:"name"`
		Type        string   `json:"type" binding:"required"`
		Content     string   `json:"content" binding:"required"`
		HostIDs     []string `json:"host_ids" binding:"required"`
		Timeout     int      `json:"timeout"`
		Concurrency int      `json:"concurrency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = 300
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 10
	}

	targetsJSON, _ := json.Marshal(req.HostIDs)
	execution := &BatchExecution{
		Name:        req.Name,
		Type:        req.Type,
		Content:     req.Content,
		Targets:     string(targetsJSON),
		TargetCount: len(req.HostIDs),
		Timeout:     req.Timeout,
		Concurrency: req.Concurrency,
		Status:      0,
		StartedAt:   time.Now(),
		Executor:    c.GetString("username"),
	}

	if err := h.db.Create(execution).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 创建结果记录
	hosts := h.getHosts(req.HostIDs)
	for _, host := range hosts {
		result := &BatchExecutionResult{
			ExecutionID: execution.ID,
			HostID:      host.ID,
			HostIP:      host.IP,
			HostName:    host.Name,
			Status:      0,
		}
		h.db.Create(result)
	}

	// 异步执行
	cancel := make(chan struct{})
	h.executions.Store(execution.ID, cancel)
	go h.runExecution(execution, hosts, cancel)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

type hostInfo struct {
	ID         string
	Name       string
	IP         string
	Port       int
	Username   string
	Password   string
	PrivateKey string
}

func (h *ExecutorHandler) getHosts(hostIDs []string) []hostInfo {
	// 从CMDB获取主机信息
	var hosts []hostInfo
	rows, err := h.db.Table("hosts").
		Select("hosts.id, hosts.name, hosts.ip, hosts.port, credentials.username, credentials.password, credentials.private_key").
		Joins("LEFT JOIN credentials ON hosts.credential_id = credentials.id").
		Where("hosts.id IN ?", hostIDs).
		Rows()
	if err != nil {
		return hosts
	}
	defer rows.Close()

	for rows.Next() {
		var host hostInfo
		rows.Scan(&host.ID, &host.Name, &host.IP, &host.Port, &host.Username, &host.Password, &host.PrivateKey)
		if host.Port == 0 {
			host.Port = 22
		}
		hosts = append(hosts, host)
	}
	return hosts
}

func (h *ExecutorHandler) runExecution(execution *BatchExecution, hosts []hostInfo, cancel chan struct{}) {
	defer func() {
		h.executions.Delete(execution.ID)
	}()

	sem := make(chan struct{}, execution.Concurrency)
	var wg sync.WaitGroup
	var successCount, failedCount int
	var mu sync.Mutex

	for _, host := range hosts {
		select {
		case <-cancel:
			h.db.Model(execution).Update("status", 4)
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(host hostInfo) {
			defer func() {
				<-sem
				wg.Done()
			}()

			result := h.executeOnHost(execution, host)

			mu.Lock()
			if result.Status == 2 {
				successCount++
			} else {
				failedCount++
			}
			progress := (successCount + failedCount) * 100 / len(hosts)
			h.db.Model(execution).Updates(map[string]interface{}{
				"progress":      progress,
				"success_count": successCount,
				"failed_count":  failedCount,
			})
			mu.Unlock()
		}(host)
	}

	wg.Wait()

	// 更新最终状态
	now := time.Now()
	status := 1
	if failedCount > 0 && successCount > 0 {
		status = 2
	} else if failedCount > 0 && successCount == 0 {
		status = 3
	}

	h.db.Model(execution).Updates(map[string]interface{}{
		"status":      status,
		"finished_at": now,
		"duration":    int(now.Sub(execution.StartedAt).Seconds()),
	})
}

func (h *ExecutorHandler) executeOnHost(execution *BatchExecution, host hostInfo) *BatchExecutionResult {
	var result BatchExecutionResult
	h.db.Where("execution_id = ? AND host_id = ?", execution.ID, host.ID).First(&result)

	now := time.Now()
	result.StartedAt = &now
	result.Status = 1
	h.db.Save(&result)

	// 建立SSH连接
	client, err := h.connectSSH(host, execution.Timeout)
	if err != nil {
		result.Status = 3
		result.Stderr = "连接失败: " + err.Error()
		finishedAt := time.Now()
		result.FinishedAt = &finishedAt
		result.Duration = int(finishedAt.Sub(now).Seconds())
		h.db.Save(&result)
		return &result
	}
	defer client.Close()

	// 执行命令
	session, err := client.NewSession()
	if err != nil {
		result.Status = 3
		result.Stderr = "创建会话失败: " + err.Error()
		finishedAt := time.Now()
		result.FinishedAt = &finishedAt
		h.db.Save(&result)
		return &result
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(execution.Content)
	finishedAt := time.Now()
	result.FinishedAt = &finishedAt
	result.Duration = int(finishedAt.Sub(now).Seconds())
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		result.Status = 3
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		}
	} else {
		result.Status = 2
		result.ExitCode = 0
	}

	h.db.Save(&result)
	return &result
}

func (h *ExecutorHandler) connectSSH(host hostInfo, timeout int) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if host.Password != "" {
		authMethods = append(authMethods, ssh.Password(host.Password))
	}
	if host.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(host.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	config := &ssh.ClientConfig{
		User:            host.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(timeout) * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host.IP, host.Port)
	conn, err := net.DialTimeout("tcp", addr, config.Timeout)
	if err != nil {
		return nil, err
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}

// ListExecutions 执行列表
func (h *ExecutorHandler) ListExecutions(c *gin.Context) {
	var executions []BatchExecution
	if err := h.db.Order("started_at DESC").Limit(100).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": executions})
}

// GetExecution 获取执行详情
func (h *ExecutorHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var execution BatchExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// CancelExecution 取消执行
func (h *ExecutorHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	if cancel, ok := h.executions.Load(id); ok {
		close(cancel.(chan struct{}))
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "执行不存在或已结束"})
}

// GetResults 获取执行结果
func (h *ExecutorHandler) GetResults(c *gin.Context) {
	id := c.Param("id")
	var results []BatchExecutionResult
	if err := h.db.Where("execution_id = ?", id).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results})
}

// StreamResults SSE实时输出
func (h *ExecutorHandler) StreamResults(c *gin.Context) {
	id := c.Param("id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 300; i++ { // 最多5分钟
		select {
		case <-ticker.C:
			var execution BatchExecution
			h.db.First(&execution, "id = ?", id)

			var results []BatchExecutionResult
			h.db.Where("execution_id = ?", id).Find(&results)

			data, _ := json.Marshal(gin.H{
				"execution": execution,
				"results":   results,
			})

			c.SSEvent("message", string(data))
			c.Writer.Flush()

			if execution.Status != 0 {
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

// ListTemplates 模板列表
func (h *ExecutorHandler) ListTemplates(c *gin.Context) {
	var templates []CommandTemplate
	if err := h.db.Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": templates})
}

// CreateTemplate 创建模板
func (h *ExecutorHandler) CreateTemplate(c *gin.Context) {
	var template CommandTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": template})
}

// DeleteTemplate 删除模板
func (h *ExecutorHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CommandTemplate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
