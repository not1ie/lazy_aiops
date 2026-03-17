package ansible

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AnsibleHandler struct {
	db         *gorm.DB
	workDir    string
	executions sync.Map
}

func NewAnsibleHandler(db *gorm.DB, workDir string) *AnsibleHandler {
	if workDir == "" {
		workDir = "data/ansible"
	}
	os.MkdirAll(workDir, 0755)
	os.MkdirAll(filepath.Join(workDir, "playbooks"), 0755)
	os.MkdirAll(filepath.Join(workDir, "inventories"), 0755)
	os.MkdirAll(filepath.Join(workDir, "roles"), 0755)
	return &AnsibleHandler{db: db, workDir: workDir}
}

// ListPlaybooks Playbook列表
func (h *AnsibleHandler) ListPlaybooks(c *gin.Context) {
	var playbooks []AnsiblePlaybook
	if err := h.db.Order("created_at DESC").Find(&playbooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": playbooks})
}

// CreatePlaybook 创建Playbook
func (h *AnsibleHandler) CreatePlaybook(c *gin.Context) {
	var playbook AnsiblePlaybook
	if err := c.ShouldBindJSON(&playbook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 保存到文件
	filename := fmt.Sprintf("%s.yml", playbook.Name)
	playbook.FilePath = filepath.Join("playbooks", filename)
	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	if err := os.WriteFile(fullPath, []byte(playbook.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
		return
	}

	if err := h.db.Create(&playbook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": playbook})
}

// GetPlaybook 获取Playbook详情
func (h *AnsibleHandler) GetPlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	// 读取文件内容
	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	content, _ := os.ReadFile(fullPath)
	playbook.Content = string(content)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": playbook})
}

// UpdatePlaybook 更新Playbook
func (h *AnsibleHandler) UpdatePlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	var req AnsiblePlaybook
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 更新文件
	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
		return
	}

	playbook.Name = req.Name
	playbook.Description = req.Description
	playbook.Content = req.Content
	playbook.Tags = req.Tags
	h.db.Save(&playbook)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": playbook})
}

// DeletePlaybook 删除Playbook
func (h *AnsibleHandler) DeletePlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	// 删除文件
	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	os.Remove(fullPath)

	h.db.Delete(&playbook)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ExecutePlaybook 执行Playbook
func (h *AnsibleHandler) ExecutePlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	var req struct {
		InventoryID string            `json:"inventory_id"`
		Hosts       string            `json:"hosts"` // 直接指定主机
		ExtraVars   map[string]string `json:"extra_vars"`
		Tags        string            `json:"tags"`
		Limit       string            `json:"limit"`
		Check       bool              `json:"check"` // dry-run
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 创建执行记录
	extraVarsJSON, _ := json.Marshal(req.ExtraVars)
	execution := &AnsibleExecution{
		PlaybookID:   playbook.ID,
		PlaybookName: playbook.Name,
		InventoryID:  req.InventoryID,
		ExtraVars:    string(extraVarsJSON),
		Tags:         req.Tags,
		Limit:        req.Limit,
		Check:        req.Check,
		Status:       0,
		StartedAt:    time.Now(),
		Executor:     c.GetString("username"),
	}
	h.db.Create(execution)

	// 异步执行
	cancel := make(chan struct{})
	h.executions.Store(execution.ID, cancel)
	go h.runPlaybook(execution, &playbook, req.InventoryID, req.Hosts, req.ExtraVars, req.Tags, req.Limit, req.Check, cancel)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

func (h *AnsibleHandler) runPlaybook(execution *AnsibleExecution, playbook *AnsiblePlaybook, inventoryID, hosts string, extraVars map[string]string, tags, limit string, check bool, cancel chan struct{}) {
	defer func() {
		h.executions.Delete(execution.ID)
	}()

	// 构建命令
	args := []string{filepath.Join(h.workDir, playbook.FilePath)}

	// 处理inventory
	if inventoryID != "" {
		var inventory AnsibleInventory
		if h.db.First(&inventory, "id = ?", inventoryID).Error == nil {
			invPath := filepath.Join(h.workDir, inventory.FilePath)
			args = append(args, "-i", invPath)
		}
	} else if hosts != "" {
		args = append(args, "-i", hosts+",")
	}

	// 额外变量
	if len(extraVars) > 0 {
		varsJSON, _ := json.Marshal(extraVars)
		args = append(args, "-e", string(varsJSON))
	}

	// Tags
	if tags != "" {
		args = append(args, "--tags", tags)
	}

	// Limit
	if limit != "" {
		args = append(args, "--limit", limit)
	}

	// Check mode
	if check {
		args = append(args, "--check")
	}

	// 执行
	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = h.workDir

	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(&output)
	cmd.Stderr = io.MultiWriter(&output)

	done := make(chan error)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		now := time.Now()
		execution.FinishedAt = &now
		execution.Duration = int(now.Sub(execution.StartedAt).Seconds())
		execution.Output = output.String()

		if err != nil {
			execution.Status = 2
		} else {
			execution.Status = 1
		}
		h.db.Save(execution)

	case <-cancel:
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		h.db.Model(execution).Update("status", 3)
	}
}

// ListExecutions 执行记录列表
func (h *AnsibleHandler) ListExecutions(c *gin.Context) {
	var executions []AnsibleExecution
	query := h.db.Order("started_at DESC")
	if playbookID := c.Query("playbook_id"); playbookID != "" {
		query = query.Where("playbook_id = ?", playbookID)
	}
	if err := query.Limit(100).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": executions})
}

// GetExecution 获取执行详情
func (h *AnsibleHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var execution AnsibleExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// CancelExecution 取消执行
func (h *AnsibleHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	if cancel, ok := h.executions.Load(id); ok {
		close(cancel.(chan struct{}))
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "执行不存在或已结束"})
}

// StreamOutput SSE实时输出
func (h *AnsibleHandler) StreamOutput(c *gin.Context) {
	id := c.Param("id")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastLen := 0
	for i := 0; i < 3600; i++ {
		select {
		case <-ticker.C:
			var execution AnsibleExecution
			h.db.First(&execution, "id = ?", id)

			if len(execution.Output) > lastLen {
				newOutput := execution.Output[lastLen:]
				lastLen = len(execution.Output)
				c.SSEvent("output", newOutput)
				c.Writer.Flush()
			}

			if execution.Status != 0 {
				c.SSEvent("done", gin.H{"status": execution.Status})
				return
			}
		case <-c.Request.Context().Done():
			return
		}
	}
}

// ListInventories Inventory列表
func (h *AnsibleHandler) ListInventories(c *gin.Context) {
	var inventories []AnsibleInventory
	if err := h.db.Find(&inventories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inventories})
}

// CreateInventory 创建Inventory
func (h *AnsibleHandler) CreateInventory(c *gin.Context) {
	var inventory AnsibleInventory
	if err := c.ShouldBindJSON(&inventory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 保存到文件
	filename := fmt.Sprintf("%s.ini", inventory.Name)
	inventory.FilePath = filepath.Join("inventories", filename)
	fullPath := filepath.Join(h.workDir, inventory.FilePath)
	if err := os.WriteFile(fullPath, []byte(inventory.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
		return
	}

	if err := h.db.Create(&inventory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inventory})
}

// GetInventory 获取Inventory详情
func (h *AnsibleHandler) GetInventory(c *gin.Context) {
	id := c.Param("id")
	var inventory AnsibleInventory
	if err := h.db.First(&inventory, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Inventory不存在"})
		return
	}

	fullPath := filepath.Join(h.workDir, inventory.FilePath)
	content, _ := os.ReadFile(fullPath)
	inventory.Content = string(content)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inventory})
}

// UpdateInventory 更新Inventory
func (h *AnsibleHandler) UpdateInventory(c *gin.Context) {
	id := c.Param("id")
	var inventory AnsibleInventory
	if err := h.db.First(&inventory, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Inventory不存在"})
		return
	}

	var req AnsibleInventory
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	fullPath := filepath.Join(h.workDir, inventory.FilePath)
	os.WriteFile(fullPath, []byte(req.Content), 0644)

	inventory.Name = req.Name
	inventory.Description = req.Description
	inventory.Content = req.Content
	h.db.Save(&inventory)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inventory})
}

// DeleteInventory 删除Inventory
func (h *AnsibleHandler) DeleteInventory(c *gin.Context) {
	id := c.Param("id")
	var inventory AnsibleInventory
	if err := h.db.First(&inventory, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Inventory不存在"})
		return
	}

	fullPath := filepath.Join(h.workDir, inventory.FilePath)
	os.Remove(fullPath)

	h.db.Delete(&inventory)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// SyncFromCMDB 从CMDB同步主机生成Inventory
func (h *AnsibleHandler) SyncFromCMDB(c *gin.Context) {
	var req struct {
		Name    string   `json:"name"`
		GroupID string   `json:"group_id"`
		HostIDs []string `json:"host_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = fmt.Sprintf("cmdb-sync-%s", time.Now().Format("20060102-150405"))
	}

	// 从CMDB获取主机
	var hosts []struct {
		Name      string
		IP        string
		Port      int
		Username  string
		GroupName string
	}

	query := h.db.Table("hosts").
		Select("hosts.name, hosts.ip, hosts.port, credentials.username, host_groups.name as group_name").
		Joins("LEFT JOIN credentials ON hosts.credential_id = credentials.id").
		Joins("LEFT JOIN host_groups ON hosts.group_id = host_groups.id")

	if req.GroupID != "" {
		query = query.Where("hosts.group_id = ?", req.GroupID)
	}
	if len(req.HostIDs) > 0 {
		query = query.Where("hosts.id IN ?", req.HostIDs)
	}
	if err := query.Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取CMDB主机失败: " + err.Error()})
		return
	}
	if len(hosts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未找到可同步的CMDB主机"})
		return
	}

	// 生成INI格式
	var content strings.Builder
	groups := make(map[string][]string)

	for _, host := range hosts {
		groupName := strings.TrimSpace(host.GroupName)
		if groupName == "" {
			groupName = "ungrouped"
		}
		groupName = sanitizeInventoryToken(groupName)
		if groupName == "" {
			groupName = "ungrouped"
		}
		hostName := sanitizeInventoryToken(strings.TrimSpace(host.Name))
		if hostName == "" {
			hostName = sanitizeInventoryToken(strings.TrimSpace(host.IP))
		}
		if hostName == "" {
			continue
		}
		port := host.Port
		if port <= 0 {
			port = 22
		}
		user := strings.TrimSpace(host.Username)
		if user == "" {
			user = "root"
		}
		line := fmt.Sprintf("%s ansible_host=%s ansible_port=%d ansible_user=%s",
			hostName, strings.TrimSpace(host.IP), port, user)
		groups[groupName] = append(groups[groupName], line)
	}
	if len(groups) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "CMDB主机数据无可用IP，无法生成Inventory"})
		return
	}

	for group, lines := range groups {
		content.WriteString(fmt.Sprintf("[%s]\n", group))
		for _, line := range lines {
			content.WriteString(line + "\n")
		}
		content.WriteString("\n")
	}

	// 保存
	filename := fmt.Sprintf("%s.ini", req.Name)
	filePath := filepath.Join("inventories", filename)
	fullPath := filepath.Join(h.workDir, filePath)
	if err := os.WriteFile(fullPath, []byte(content.String()), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Inventory写入失败: " + err.Error()})
		return
	}

	inventory := &AnsibleInventory{
		Name:        req.Name,
		Description: "从CMDB同步",
		FilePath:    filePath,
		Content:     content.String(),
		Source:      "cmdb",
	}
	if err := h.db.Create(inventory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存Inventory失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": inventory})
}

func sanitizeInventoryToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, " ", "_")
	raw = strings.ReplaceAll(raw, "/", "_")
	raw = strings.ReplaceAll(raw, "\\", "_")
	return raw
}

// ListRoles Role列表
func (h *AnsibleHandler) ListRoles(c *gin.Context) {
	var roles []AnsibleRole
	if err := h.db.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": roles})
}

// InstallRole 安装Role (从Galaxy)
func (h *AnsibleHandler) InstallRole(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"` // galaxy role name
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	args := []string{"install", req.Name, "-p", filepath.Join(h.workDir, "roles")}
	if req.Version != "" {
		args = append(args, ","+req.Version)
	}

	cmd := exec.Command("ansible-galaxy", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": string(output)})
		return
	}

	// 记录
	role := &AnsibleRole{
		Name:    req.Name,
		Version: req.Version,
		Source:  "galaxy",
		Path:    filepath.Join("roles", req.Name),
	}
	h.db.Create(role)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": role, "output": string(output)})
}

// ValidatePlaybook 验证Playbook语法
func (h *AnsibleHandler) ValidatePlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	cmd := exec.Command("ansible-playbook", "--syntax-check", fullPath)
	output, err := cmd.CombinedOutput()

	valid := err == nil
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"valid":  valid,
			"output": string(output),
		},
	})
}

// ParsePlaybook 解析Playbook获取变量
func (h *AnsibleHandler) ParsePlaybook(c *gin.Context) {
	id := c.Param("id")
	var playbook AnsiblePlaybook
	if err := h.db.First(&playbook, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Playbook不存在"})
		return
	}

	fullPath := filepath.Join(h.workDir, playbook.FilePath)
	content, _ := os.ReadFile(fullPath)

	// 简单解析变量 {{ var }}
	variables := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		// 查找 {{ xxx }}
		for {
			start := strings.Index(line, "{{")
			if start == -1 {
				break
			}
			end := strings.Index(line[start:], "}}")
			if end == -1 {
				break
			}
			varName := strings.TrimSpace(line[start+2 : start+end])
			if !strings.Contains(varName, ".") && !strings.Contains(varName, "|") {
				variables = append(variables, varName)
			}
			line = line[start+end+2:]
		}
	}

	// 去重
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, v := range variables {
		if !seen[v] {
			seen[v] = true
			unique = append(unique, v)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"variables": unique}})
}
