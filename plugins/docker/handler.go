package docker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

type DockerHandler struct {
	db   *gorm.DB
	auth *core.AuthService
}

var dockerUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewDockerHandler(db *gorm.DB, auth *core.AuthService) *DockerHandler {
	return &DockerHandler{db: db, auth: auth}
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
			"status":  "error",
			"version": fmt.Sprintf("Error: %s", stderr),
		})
		return err
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		// JSON 解析失败
		h.db.Model(&DockerHost{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":  "error",
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
	since := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "0")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 使用 2>&1 合并 stdout 和 stderr
	cmd := fmt.Sprintf("docker logs --tail %s", tail)
	if since != "" {
		cmd += fmt.Sprintf(" --since %s", since)
	}
	if timestamps == "1" || strings.ToLower(timestamps) == "true" {
		cmd += " --timestamps"
	}
	cmd = fmt.Sprintf("%s %s 2>&1", cmd, containerID)
	stdout, _, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stdout})
}

// ExecContainer 执行容器命令（非交互）
func (h *DockerHandler) ExecContainer(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	var req struct {
		Command string `json:"command"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Command) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请输入要执行的命令"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	cmd := fmt.Sprintf("docker exec %s sh -c %q 2>&1", containerID, req.Command)
	stdout, _, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "data": stdout})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stdout})
}

// ExecContainerWS WebSocket 交互终端
func (h *DockerHandler) ExecContainerWS(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	token := c.Query("token")
	shell := c.DefaultQuery("shell", "/bin/sh")

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
		return
	}
	if h.auth != nil {
		if _, err := h.auth.ValidateToken(token); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token无效"})
			return
		}
	}

	if !isAllowedShell(shell) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Shell不支持"})
		return
	}

	conn, err := dockerUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	var dHost DockerHost
	if err := h.db.First(&dHost, "id = ?", id).Error; err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Docker主机不存在"))
		conn.Close()
		return
	}

	if dHost.HostID == "local" {
		h.handleLocalExecWS(conn, containerID, shell)
		return
	}

	var host cmdb.Host
	if err := h.db.Preload("Credential").First(&host, "id = ?", dHost.HostID).Error; err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("关联主机不存在"))
		conn.Close()
		return
	}
	if host.Credential == nil {
		conn.WriteMessage(websocket.TextMessage, []byte("主机未配置凭据"))
		conn.Close()
		return
	}

	h.handleSSHExecWS(conn, &host, containerID, shell)
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
		imageID, _ := item["ImageID"].(string)
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
			ImageID: imageID,
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

// CreateContainer 创建并启动容器
func (h *DockerHandler) CreateContainer(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name          string            `json:"name"`
		Image         string            `json:"image" binding:"required"`
		Ports         []string          `json:"ports"` // Format: "8080:80"
		Env           map[string]string `json:"env"`
		RestartPolicy string            `json:"restart_policy"`
		AutoRemove    bool              `json:"auto_remove"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 构建 docker run 命令
	var cmdBuilder strings.Builder
	cmdBuilder.WriteString("docker run -d")

	if req.Name != "" {
		cmdBuilder.WriteString(fmt.Sprintf(" --name %s", req.Name))
	}

	for _, p := range req.Ports {
		if p != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" -p %s", p))
		}
	}

	for k, v := range req.Env {
		cmdBuilder.WriteString(fmt.Sprintf(" -e %s=%s", k, v))
	}

	if req.AutoRemove {
		cmdBuilder.WriteString(" --rm")
	} else if req.RestartPolicy != "" {
		cmdBuilder.WriteString(fmt.Sprintf(" --restart %s", req.RestartPolicy))
	}

	cmdBuilder.WriteString(fmt.Sprintf(" %s", req.Image))

	// 执行
	stdout, stderr, err := client.Execute(cmdBuilder.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("启动失败: %s | %s", err.Error(), stderr)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "容器创建成功", "container_id": strings.TrimSpace(stdout)})
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

// ListContainerStats 容器资源概览
func (h *DockerHandler) ListContainerStats(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stdout, stderr, err := client.Execute("docker stats --no-stream --format '{{json .}}'")
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	stats := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
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

	stdout, stderr, err := client.Execute("docker images --no-trunc --format '{{.ID}}||{{.Repository}}||{{.Tag}}||{{.CreatedAt}}||{{.CreatedSince}}||{{.Size}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}

	images := make([]map[string]string, 0)
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "||")
		if len(parts) < 6 {
			continue
		}
		images = append(images, map[string]string{
			"ID":           parts[0],
			"Repository":   parts[1],
			"Tag":          parts[2],
			"CreatedAt":    parts[3],
			"CreatedSince": parts[4],
			"Size":         parts[5],
		})
	}
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
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// PruneImages 清理悬挂镜像
func (h *DockerHandler) PruneImages(c *gin.Context) {
	id := c.Param("id")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	cmd := "docker image prune -f --filter dangling=true"

	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"output": stdout}})
}

// ================= Volume Management =================

// ListVolumes 卷列表
func (h *DockerHandler) ListVolumes(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker volume ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	vols := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": vols})
}

// InspectVolume 卷详情
func (h *DockerHandler) InspectVolume(c *gin.Context) {
	id := c.Param("id")
	volume := c.Param("volume")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker volume inspect %s", volume))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}
	if len(result) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "卷未找到"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result[0]})
}

// CreateVolume 创建卷
func (h *DockerHandler) CreateVolume(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name   string            `json:"name"`
		Driver string            `json:"driver"`
		Labels map[string]string `json:"labels"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var cmd strings.Builder
	cmd.WriteString("docker volume create")
	if strings.TrimSpace(req.Driver) != "" {
		cmd.WriteString(fmt.Sprintf(" --driver %s", req.Driver))
	}
	for k, v := range req.Labels {
		cmd.WriteString(fmt.Sprintf(" --label %s=%s", k, v))
	}
	cmd.WriteString(fmt.Sprintf(" %s", req.Name))

	_, stderr, err := client.Execute(cmd.String())
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// RemoveVolume 删除卷
func (h *DockerHandler) RemoveVolume(c *gin.Context) {
	id := c.Param("id")
	volume := c.Param("volume")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_, stderr, err := client.Execute(fmt.Sprintf("docker volume rm %s", volume))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ================= Secret/Config Management =================

// ListSecrets secret 列表
func (h *DockerHandler) ListSecrets(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker secret ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	secrets := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": secrets})
}

// InspectSecret secret 详情
func (h *DockerHandler) InspectSecret(c *gin.Context) {
	id := c.Param("id")
	secret := c.Param("secret")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker secret inspect %s", secret))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}
	if len(result) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Secret未找到"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result[0]})
}

// CreateSecret 创建 secret
func (h *DockerHandler) CreateSecret(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(req.Data))
	cmd := fmt.Sprintf("printf '%s' | base64 -d | docker secret create %s -", encoded, req.Name)
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// RemoveSecret 删除 secret
func (h *DockerHandler) RemoveSecret(c *gin.Context) {
	id := c.Param("id")
	secret := c.Param("secret")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_, stderr, err := client.Execute(fmt.Sprintf("docker secret rm %s", secret))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListConfigs config 列表
func (h *DockerHandler) ListConfigs(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker config ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	configs := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// InspectConfig config 详情
func (h *DockerHandler) InspectConfig(c *gin.Context) {
	id := c.Param("id")
	config := c.Param("config")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker config inspect %s", config))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}
	if len(result) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Config未找到"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result[0]})
}

// CreateConfig 创建 config
func (h *DockerHandler) CreateConfig(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(req.Data))
	cmd := fmt.Sprintf("printf '%s' | base64 -d | docker config create %s -", encoded, req.Name)
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// RemoveConfig 删除 config
func (h *DockerHandler) RemoveConfig(c *gin.Context) {
	id := c.Param("id")
	config := c.Param("config")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_, stderr, err := client.Execute(fmt.Sprintf("docker config rm %s", config))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ================= Node Management =================

// ListNodes Swarm 节点列表
func (h *DockerHandler) ListNodes(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker node ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	nodes := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

// ================= Stack Deploy =================

// DeployStack 部署/更新 Stack
func (h *DockerHandler) DeployStack(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name    string `json:"name"`
		Compose string `json:"compose"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Compose) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name/compose 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stackName := sanitizeStackName(req.Name)
	encoded := base64.StdEncoding.EncodeToString([]byte(req.Compose))
	tmp := fmt.Sprintf("/tmp/lazy-aiops-stack-%d.yml", time.Now().UnixNano())
	cmd := fmt.Sprintf("printf '%s' | base64 -d > %s && docker stack deploy -c %s %s", encoded, tmp, tmp, stackName)
	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"output": stdout}})
}

// ================= Registry Management =================

// ListRegistries 仓库列表
func (h *DockerHandler) ListRegistries(c *gin.Context) {
	var regs []DockerRegistry
	if err := h.db.Order("created_at DESC").Find(&regs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range regs {
		regs[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": regs})
}

// CreateRegistry 创建仓库
func (h *DockerHandler) CreateRegistry(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
		Insecure bool   `json:"insecure"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.URL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	reg := DockerRegistry{
		Name:     req.Name,
		URL:      req.URL,
		Username: req.Username,
		Password: req.Password,
		Insecure: req.Insecure,
	}
	if err := h.db.Create(&reg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	reg.Password = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": reg})
}

// DeleteRegistry 删除仓库
func (h *DockerHandler) DeleteRegistry(c *gin.Context) {
	id := c.Param("registry_id")
	h.db.Delete(&DockerRegistry{}, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// LoginRegistry 登录仓库（当前主机）
func (h *DockerHandler) LoginRegistry(c *gin.Context) {
	hostID := c.Param("id")
	registryID := c.Param("registry_id")

	var reg DockerRegistry
	if err := h.db.First(&reg, "id = ?", registryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	client, err := h.getClient(hostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user := strings.TrimSpace(reg.Username)
	pass := reg.Password
	url := strings.TrimSpace(reg.URL)
	if user == "" || pass == "" || url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仓库凭据不完整"})
		return
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(pass))
	cmd := fmt.Sprintf("printf '%s' | base64 -d | docker login %s -u %s --password-stdin", encoded, url, user)
	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		_ = h.upsertRegistryLogin(hostID, registryID, "failed", msg)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	_ = h.upsertRegistryLogin(hostID, registryID, "success", "login ok")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"output": stdout}})
}

// ListRegistriesForHost 按主机返回登录状态
func (h *DockerHandler) ListRegistriesForHost(c *gin.Context) {
	hostID := c.Param("id")
	var regs []DockerRegistry
	if err := h.db.Order("created_at DESC").Find(&regs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	var logins []DockerRegistryLogin
	_ = h.db.Where("docker_host_id = ?", hostID).Find(&logins).Error
	loginMap := make(map[string]DockerRegistryLogin)
	for _, l := range logins {
		loginMap[l.RegistryID] = l
	}
	resp := make([]gin.H, 0, len(regs))
	for _, r := range regs {
		login := loginMap[r.ID]
		resp = append(resp, gin.H{
			"id":            r.ID,
			"name":          r.Name,
			"url":           r.URL,
			"username":      r.Username,
			"insecure":      r.Insecure,
			"login_status":  login.Status,
			"last_login_at": login.LastLoginAt,
			"login_message": login.Message,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func (h *DockerHandler) upsertRegistryLogin(hostID, registryID, status, msg string) error {
	var rec DockerRegistryLogin
	err := h.db.Where("docker_host_id = ? AND registry_id = ?", hostID, registryID).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rec = DockerRegistryLogin{
				DockerHostID: hostID,
				RegistryID:   registryID,
			}
		} else {
			return err
		}
	}
	rec.Status = status
	rec.Message = msg
	rec.LastLoginAt = time.Now()
	return h.db.Save(&rec).Error
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

// InspectNetwork 查看网络详情
func (h *DockerHandler) InspectNetwork(c *gin.Context) {
	id := c.Param("id")
	networkID := c.Param("network_id")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stdout, stderr, err := client.Execute(fmt.Sprintf("docker network inspect %s", networkID))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	var result []interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}
	if len(result) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "网络未找到"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result[0]})
}

// RemoveNetwork 删除网络
func (h *DockerHandler) RemoveNetwork(c *gin.Context) {
	id := c.Param("id")
	networkID := c.Param("network_id")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	_, stderr, err := client.Execute(fmt.Sprintf("docker network rm %s", networkID))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ================= Swarm Management =================

// ListServices Swarm 服务列表
func (h *DockerHandler) ListServices(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker service ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	services := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": services})
}

// InspectService 服务详情
func (h *DockerHandler) InspectService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker service inspect %s --format '{{json .}}'", serviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}

// ListServiceTasks 服务任务列表
func (h *DockerHandler) ListServiceTasks(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker service ps %s --no-trunc --format '{{json .}}'", serviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	tasks := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tasks})
}

// ServiceLogs 服务日志
func (h *DockerHandler) ServiceLogs(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	tail := c.DefaultQuery("tail", "200")
	since := c.Query("since")
	timestamps := c.DefaultQuery("timestamps", "1")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	cmd := "docker service logs --raw"
	if timestamps == "1" || strings.ToLower(timestamps) == "true" {
		cmd += " --timestamps"
	}
	if since != "" {
		cmd += fmt.Sprintf(" --since %s", since)
	}
	cmd += fmt.Sprintf(" --tail %s %s", tail, serviceID)

	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stdout})
}

// ListStacks Stack 列表
func (h *DockerHandler) ListStacks(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute("docker stack ls --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	stacks := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stacks})
}

// ListStackServices Stack 服务列表
func (h *DockerHandler) ListStackServices(c *gin.Context) {
	id := c.Param("id")
	stack := c.Param("stack")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	stdout, stderr, err := client.Execute(fmt.Sprintf("docker stack services %s --format '{{json .}}'", stack))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("执行失败: %s | %s", err.Error(), stderr)})
		return
	}
	services := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": services})
}

// RemoveStack 删除Stack
func (h *DockerHandler) RemoveStack(c *gin.Context) {
	id := c.Param("id")
	stack := c.Param("stack")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	_, stderr, err := client.Execute(fmt.Sprintf("docker stack rm %s", stack))
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ScaleService 调整服务副本数
func (h *DockerHandler) ScaleService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	var req struct {
		Replicas int `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Replicas < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "replicas 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	cmd := fmt.Sprintf("docker service scale %s=%d", serviceID, req.Replicas)
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已提交扩缩容"})
}

// UpdateServiceImage 更新服务镜像
func (h *DockerHandler) UpdateServiceImage(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	var req struct {
		Image string `json:"image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Image) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "image 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	cmd := fmt.Sprintf("docker service update --image %s %s", req.Image, serviceID)
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已提交镜像更新"})
}

// RestartService 滚动重启服务
func (h *DockerHandler) RestartService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	cmd := fmt.Sprintf("docker service update --force %s", serviceID)
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已触发重启"})
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

func sanitizeStackName(name string) string {
	allowed := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}
	clean := strings.Map(allowed, name)
	if clean == "" {
		return "stack"
	}
	return clean
}

func isAllowedShell(shell string) bool {
	switch shell {
	case "/bin/sh", "/bin/bash", "/bin/ash":
		return true
	default:
		return false
	}
}

func (h *DockerHandler) handleLocalExecWS(conn *websocket.Conn, containerID, shell string) {
	defer conn.Close()
	cmd := fmt.Sprintf("docker exec -i %s %s", containerID, shell)
	proc := exec.Command("sh", "-c", cmd)
	stdin, err := proc.StdinPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stdin失败"))
		return
	}
	stdout, err := proc.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stdout失败"))
		return
	}
	stderr, err := proc.StderrPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stderr失败"))
		return
	}
	if err := proc.Start(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("启动失败: "+err.Error()))
		return
	}

	go streamToWS(conn, stdout)
	go streamToWS(conn, stderr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if isResizeMessage(message) {
			continue
		}
		_, _ = stdin.Write(message)
	}
	_ = proc.Process.Kill()
}

func (h *DockerHandler) handleSSHExecWS(conn *websocket.Conn, host *cmdb.Host, containerID, shell string) {
	defer conn.Close()
	client, err := dialSSH(host)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("SSH连接失败: "+err.Error()))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("创建会话失败: "+err.Error()))
		return
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 40, 120, modes); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("请求PTY失败: "+err.Error()))
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stdin失败"))
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stdout失败"))
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("打开stderr失败"))
		return
	}

	cmd := fmt.Sprintf("docker exec -it %s %s", containerID, shell)
	if err := session.Start(cmd); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("执行失败: "+err.Error()))
		return
	}

	go streamToWS(conn, stdout)
	go streamToWS(conn, stderr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if resize, ok := parseResizeMessage(message); ok {
			_ = session.WindowChange(resize.Rows, resize.Cols)
			continue
		}
		_, _ = stdin.Write(message)
	}
}

type resizePayload struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

func parseResizeMessage(message []byte) (resizePayload, bool) {
	var payload resizePayload
	if err := json.Unmarshal(message, &payload); err != nil {
		return payload, false
	}
	if payload.Type == "resize" && payload.Cols > 0 && payload.Rows > 0 {
		return payload, true
	}
	return payload, false
}

func isResizeMessage(message []byte) bool {
	_, ok := parseResizeMessage(message)
	return ok
}

func streamToWS(conn *websocket.Conn, reader io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			_ = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
		}
	}
}

func dialSSH(host *cmdb.Host) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	if host.Credential == nil {
		return nil, fmt.Errorf("主机未配置凭据")
	}
	if host.Credential.Password != "" {
		authMethods = append(authMethods, ssh.Password(host.Credential.Password))
	}
	if host.Credential.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(host.Credential.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	cfg := &ssh.ClientConfig{
		User:            host.Credential.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host.IP, host.Port)
	conn, err := net.DialTimeout("tcp", addr, cfg.Timeout)
	if err != nil {
		return nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}
