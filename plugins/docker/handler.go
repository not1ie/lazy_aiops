package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/gorm"
)

type DockerHandler struct {
	db *gorm.DB
}

func NewDockerHandler(db *gorm.DB) *DockerHandler {
	return &DockerHandler{db: db}
}

// ================= Host Management =================

// ListHosts 主机列表
func (h *DockerHandler) ListHosts(c *gin.Context) {
	var hosts []DockerHost
	if err := h.db.Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": hosts})
}

// AddHost 添加主机
func (h *DockerHandler) AddHost(c *gin.Context) {
	var req struct {
		HostID string `json:"host_id" binding:"required"`
		Name   string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	host := DockerHost{
		Name:   req.Name,
		HostID: req.HostID,
		Status: "unknown",
	}
	if err := h.db.Create(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

// DeleteHost 删除主机
func (h *DockerHandler) DeleteHost(c *gin.Context) {
	id := c.Param("id")
	h.db.Delete(&DockerHost{}, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 测试连接并返回原始输出 (用于调试)
func (h *DockerHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "获取SSH配置失败: " + err.Error()})
		return
	}

	// 1. Basic Info (Human)
	cmdInfo := "docker info"
	outInfo, errInfo, _ := client.Execute(cmdInfo)

	// 2. Info JSON (Sync)
	cmdInfoJson := "docker system info --format '{{json .}}'"
	outInfoJson, errInfoJson, _ := client.Execute(cmdInfoJson)

	// 3. PS JSON (List)
	cmdPsJson := "docker ps -a --format '{{json .}}'"
	outPsJson, errPsJson, _ := client.Execute(cmdPsJson)
	
	result := gin.H{
		"step1_info": gin.H{
			"cmd": cmdInfo,
			"out": outInfo,
			"err": errInfo,
		},
		"step2_sync": gin.H{
			"cmd": cmdInfoJson,
			"out": outInfoJson,
			"err": errInfoJson,
		},
		"step3_list": gin.H{
			"cmd": cmdPsJson,
			"out": outPsJson,
			"err": errPsJson,
		},
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// SyncHosts 强制同步所有主机状态
func (h *DockerHandler) SyncHosts(c *gin.Context) {
	var hosts []DockerHost
	if err := h.db.Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			h.syncHostInternal(id)
		}(host.ID)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步完成"})
}

// GetHostInfo 获取主机概览信息 (Docker Info)
func (h *DockerHandler) GetHostInfo(c *gin.Context) {
	id := c.Param("id")
	
	// 强制同步一次
	if err := h.syncHostInternal(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "同步失败: " + err.Error()})
		return
	}

	// 从数据库重新获取更新后的信息
	var host DockerHost
	h.db.First(&host, "id = ?", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

// syncHostInternal 内部同步逻辑
func (h *DockerHandler) syncHostInternal(id string) error {
	client, err := h.getClient(id)
	if err != nil {
		h.db.Model(&DockerHost{}).Where("id = ?", id).Update("status", "offline")
		return err
	}

	// 使用 docker system info 更稳健
	stdout, stderr, err := client.Execute("docker system info --format '{{json .}}'")
	
	// 即使 err != nil，有时候 stdout 也有内容（比如有警告）
	// 但如果 err != nil 且 stderr 有严重错误，那肯定是挂了
	if err != nil && stdout == "" {
		h.db.Model(&DockerHost{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status": "error", 
			"version": fmt.Sprintf("Error: %s", stderr),
		})
		return err
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		// JSON 解析失败
		h.db.Model(&DockerHost{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status": "error",
			"version": "JSON Parse Error",
		})
		return fmt.Errorf("JSON parse failed: %v | Output: %s", err, stdout)
	}
	
	updates := map[string]interface{}{
		"status": "online",
	}
	
	if v, ok := info["Containers"].(float64); ok {
		updates["container_count"] = int(v)
	}
	if v, ok := info["Images"].(float64); ok {
		updates["image_count"] = int(v)
	}
	if v, ok := info["ServerVersion"].(string); ok {
		updates["version"] = v
	}

	return h.db.Model(&DockerHost{}).Where("id = ?", id).Updates(updates).Error
}

// ContainerLogs 获取容器日志
func (h *DockerHandler) ContainerLogs(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	tail := c.DefaultQuery("tail", "100")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 使用 2>&1 合并 stdout 和 stderr
	cmd := fmt.Sprintf("docker logs --tail %s %s 2>&1", tail, containerID)
	stdout, _, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stdout})
}

// ================= Container Management =================

// ListContainers 容器列表
func (h *DockerHandler) ListContainers(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 远程执行 docker ps
	stdout, stderr, err := client.Execute("docker ps -a --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}

	rawList := parseJSONList(stdout)
	var containers []DockerContainer

	for _, item := range rawList {
		// 容错处理：Docker CLI 返回的 Key 可能是 ID, Names, Image 等 (PascalCase)
		// 我们将其映射到 DockerContainer (camelCase json tags)
		
		id, _ := item["ID"].(string)
		
		// Names 可能是 "name1,name2" 字符串，我们需要转为数组
		namesStr, _ := item["Names"].(string)
		names := strings.Split(namesStr, ",")
		
		image, _ := item["Image"].(string)
		state, _ := item["State"].(string)
		status, _ := item["Status"].(string)
		ports, _ := item["Ports"].(string)
		createdStr, _ := item["CreatedAt"].(string)
		
		// 尝试解析时间 (Docker time format is tricky, keep it simple or raw string for now if needed, 
		// but model expects int64 timestamp. Let's change model to string for display simplicity or parse it)
		// For now, let's keep Created as int64 0 to avoid parsing errors, or update model.
		// Actually, let's update the model to use string for 'created' to be safe, 
		// OR just pass the string in a new field.
		// To match existing frontend expectation, let's see.
		// Frontend uses `c.Created` (Turn 36).
		
		containers = append(containers, DockerContainer{
			ID:      id,
			Names:   names,
			Image:   image,
			State:   state,
			Status:  status,
			Ports:   ports,
			Created: createdStr,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": containers})
}

// InspectContainer 容器详情 (Env, Network, Mounts)
func (h *DockerHandler) InspectContainer(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stdout, _, err := client.Execute(fmt.Sprintf("docker inspect %s", containerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// docker inspect 返回的是一个数组，我们只取第一个
	var result []interface{}
	json.Unmarshal([]byte(stdout), &result)
	
	if len(result) > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": result[0]})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "容器未找到"})
	}
}

// ContainerAction 容器操作 (Start/Stop/Restart/Remove)
func (h *DockerHandler) ContainerAction(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	action := c.Param("action")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var cmd string
	switch action {
	case "start", "stop", "restart":
		cmd = fmt.Sprintf("docker %s %s", action, containerID)
	case "remove":
		cmd = fmt.Sprintf("docker rm -f %s", containerID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的操作"})
		return
	}

	_, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "操作成功"})
}

// ================= Image Management =================

// ListImages 镜像列表
func (h *DockerHandler) ListImages(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stdout, stderr, err := client.Execute("docker images --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}

	images := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": images})
}

// PullImage 拉取镜像
func (h *DockerHandler) PullImage(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 异步拉取，避免阻塞？或者同步等待返回结果？Portainer 通常是长连接或轮询。
	// 这里简化为同步等待，超时设长一点。
	// 注意：docker pull 输出到 stdout
	cmd := fmt.Sprintf("docker pull %s", req.Image)
	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("拉取失败: %s | %s", err.Error(), stderr)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "拉取成功", "output": stdout})
}

// RemoveImage 删除镜像
func (h *DockerHandler) RemoveImage(c *gin.Context) {
	id := c.Param("id")
	imageID := c.Param("image_id")
	
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	_, stderr, err := client.Execute(fmt.Sprintf("docker rmi %s", imageID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ================= Network Management =================

// ListNetworks 网络列表
func (h *DockerHandler) ListNetworks(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stdout, _, err := client.Execute("docker network ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	networks := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": networks})
}

// CommandExecutor 命令执行接口
type CommandExecutor interface {
	Execute(cmd string) (string, string, error)
}

// LocalExecutor 本地执行器
type LocalExecutor struct{}

func (l *LocalExecutor) Execute(cmd string) (string, string, error) {
	// 使用 sh -c 执行命令
	c := exec.Command("sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// ================= Helper Functions =================

func (h *DockerHandler) getClient(dockerHostID string) (CommandExecutor, error) {
	var dHost DockerHost
	if err := h.db.First(&dHost, "id = ?", dockerHostID).Error; err != nil {
		return nil, fmt.Errorf("Docker主机不存在")
	}

	// 支持 Local 模式
	if dHost.HostID == "local" {
		return &LocalExecutor{}, nil
	}

	var host cmdb.Host
	if err := h.db.Preload("Credential").First(&host, "id = ?", dHost.HostID).Error; err != nil {
		return nil, fmt.Errorf("关联主机不存在")
	}

	if host.Credential == nil {
		return nil, fmt.Errorf("主机未配置凭据")
	}

	return &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: host.Credential.Username,
		Password: host.Credential.Password,
		Key:      host.Credential.PrivateKey,
		Timeout:  10 * time.Second,
	}, nil
}

// parseJSONList 解析Docker CLI返回的逐行JSON
func parseJSONList(output string) []map[string]interface{} {
	var list []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			list = append(list, item)
		}
	}
	return list
}