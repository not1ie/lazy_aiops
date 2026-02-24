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
	"net/url"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
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
		Labels        map[string]string `json:"labels"`
		Networks      []string          `json:"networks"`
		Mounts        []string          `json:"mounts"`
		Entrypoint    string            `json:"entrypoint"`
		Command       []string          `json:"command"`
		Privileged    bool              `json:"privileged"`
		CapAdd        []string          `json:"cap_add"`
		NetworkMode   string            `json:"network_mode"`
		DNS           []string          `json:"dns"`
		ExtraHosts    []string          `json:"extra_hosts"`
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
		cmdBuilder.WriteString(fmt.Sprintf(" --name %s", shellEscape(req.Name)))
	}

	for _, p := range req.Ports {
		if p != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" -p %s", shellEscape(p)))
		}
	}

	for k, v := range req.Env {
		cmdBuilder.WriteString(fmt.Sprintf(" -e %s", shellEscape(fmt.Sprintf("%s=%s", k, v))))
	}

	for k, v := range req.Labels {
		cmdBuilder.WriteString(fmt.Sprintf(" --label %s", shellEscape(fmt.Sprintf("%s=%s", k, v))))
	}

	for _, n := range req.Networks {
		if strings.TrimSpace(n) != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" --network %s", shellEscape(n)))
		}
	}
	if req.NetworkMode != "" && len(req.Networks) == 0 {
		cmdBuilder.WriteString(fmt.Sprintf(" --network %s", shellEscape(req.NetworkMode)))
	}

	if req.Privileged {
		cmdBuilder.WriteString(" --privileged")
	}

	for _, cap := range req.CapAdd {
		cap = strings.TrimSpace(cap)
		if cap != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" --cap-add %s", shellEscape(cap)))
		}
	}

	for _, dns := range req.DNS {
		dns = strings.TrimSpace(dns)
		if dns != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" --dns %s", shellEscape(dns)))
		}
	}

	for _, host := range req.ExtraHosts {
		host = strings.TrimSpace(host)
		if host != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" --add-host %s", shellEscape(host)))
		}
	}

	for _, m := range req.Mounts {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if strings.Contains(m, "type=") {
			cmdBuilder.WriteString(fmt.Sprintf(" --mount %s", shellEscape(m)))
		} else {
			cmdBuilder.WriteString(fmt.Sprintf(" -v %s", shellEscape(m)))
		}
	}

	if req.Entrypoint != "" {
		cmdBuilder.WriteString(fmt.Sprintf(" --entrypoint %s", shellEscape(req.Entrypoint)))
	}

	if req.AutoRemove {
		cmdBuilder.WriteString(" --rm")
	} else if req.RestartPolicy != "" {
		cmdBuilder.WriteString(fmt.Sprintf(" --restart %s", shellEscape(req.RestartPolicy)))
	}

	cmdBuilder.WriteString(fmt.Sprintf(" %s", shellEscape(req.Image)))

	for _, arg := range req.Command {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			cmdBuilder.WriteString(fmt.Sprintf(" %s", shellEscape(arg)))
		}
	}

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

// DeployStackFromGit 从 Git 仓库部署 Stack
func (h *DockerHandler) DeployStackFromGit(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string `json:"name"`
		Repo        string `json:"repo"`
		Branch      string `json:"branch"`
		ComposePath string `json:"compose_path"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		Token       string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Repo) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name/repo 参数错误"})
		return
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = "main"
	}
	composePath := strings.TrimSpace(req.ComposePath)
	if composePath == "" {
		composePath = "docker-compose.yml"
	}
	composePath, err := sanitizeRepoPath(composePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	repoURL := strings.TrimSpace(req.Repo)
	if req.Token != "" || req.Username != "" || req.Password != "" {
		user := req.Username
		pass := req.Password
		if req.Token != "" && user == "" {
			user = "oauth2"
			pass = req.Token
		}
		repoURL, err = withBasicAuthURL(repoURL, user, pass)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "repo 参数错误"})
			return
		}
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	stackName := sanitizeStackName(req.Name)
	repoDir := fmt.Sprintf("/tmp/lazy-aiops-repo-%d", time.Now().UnixNano())
	composeFile := path.Join(repoDir, composePath)

	cmd := fmt.Sprintf("rm -rf %s && git clone --depth=1 --branch %s %s %s",
		shellEscape(repoDir),
		shellEscape(branch),
		shellEscape(repoURL),
		shellEscape(repoDir),
	)
	cmd += fmt.Sprintf(" && docker stack deploy -c %s %s", shellEscape(composeFile), shellEscape(stackName))
	cmd += fmt.Sprintf(" && rm -rf %s", shellEscape(repoDir))

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

// ListEvents 事件列表
func (h *DockerHandler) ListEvents(c *gin.Context) {
	id := c.Param("id")
	since := strings.TrimSpace(c.DefaultQuery("since", "1h"))
	until := strings.TrimSpace(c.Query("until"))
	filterType := strings.TrimSpace(c.Query("type"))
	filterAction := strings.TrimSpace(c.Query("event"))
	filterContainer := strings.TrimSpace(c.Query("container"))
	filterImage := strings.TrimSpace(c.Query("image"))
	filterVolume := strings.TrimSpace(c.Query("volume"))
	filterNetwork := strings.TrimSpace(c.Query("network"))
	filterNode := strings.TrimSpace(c.Query("node"))
	filterService := strings.TrimSpace(c.Query("service"))

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	cmd := "docker events --format '{{json .}}'"
	if since != "" {
		cmd += fmt.Sprintf(" --since %s", shellEscape(since))
	}
	if until != "" {
		cmd += fmt.Sprintf(" --until %s", shellEscape(until))
	}
	if filterType != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("type="+filterType))
	}
	if filterAction != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("event="+filterAction))
	}
	if filterContainer != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("container="+filterContainer))
	}
	if filterImage != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("image="+filterImage))
	}
	if filterVolume != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("volume="+filterVolume))
	}
	if filterNetwork != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("network="+filterNetwork))
	}
	if filterNode != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("node="+filterNode))
	}
	if filterService != "" {
		cmd += fmt.Sprintf(" --filter %s", shellEscape("service="+filterService))
	}

	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	events := parseJSONList(stdout)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": events})
}

// BuildImage 构建镜像
func (h *DockerHandler) BuildImage(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Tag        string            `json:"tag"`
		Dockerfile string            `json:"dockerfile"`
		ContextTar string            `json:"context_tar"`
		BuildArgs  map[string]string `json:"build_args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" || strings.TrimSpace(req.Dockerfile) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "tag/dockerfile 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	tmpDir := fmt.Sprintf("/tmp/lazy-aiops-build-%d", time.Now().UnixNano())
	encodedDockerfile := base64.StdEncoding.EncodeToString([]byte(req.Dockerfile))
	cmd := fmt.Sprintf("mkdir -p %s && printf '%s' | base64 -d > %s/Dockerfile",
		shellEscape(tmpDir),
		encodedDockerfile,
		shellEscape(tmpDir),
	)
	if strings.TrimSpace(req.ContextTar) != "" {
		encodedCtx := req.ContextTar
		cmd += fmt.Sprintf(" && printf '%s' | base64 -d | tar -xf - -C %s",
			encodedCtx,
			shellEscape(tmpDir),
		)
	}

	buildCmd := fmt.Sprintf("docker build -t %s -f %s/Dockerfile %s",
		shellEscape(req.Tag),
		shellEscape(tmpDir),
		shellEscape(tmpDir),
	)
	if len(req.BuildArgs) > 0 {
		args := make([]string, 0, len(req.BuildArgs))
		for k, v := range req.BuildArgs {
			if strings.TrimSpace(k) == "" {
				continue
			}
			args = append(args, fmt.Sprintf("--build-arg %s", shellEscape(fmt.Sprintf("%s=%s", k, v))))
		}
		if len(args) > 0 {
			buildCmd = fmt.Sprintf("docker build %s -t %s -f %s/Dockerfile %s",
				strings.Join(args, " "),
				shellEscape(req.Tag),
				shellEscape(tmpDir),
				shellEscape(tmpDir),
			)
		}
	}

	cmd += " && " + buildCmd
	cmd += fmt.Sprintf(" && rm -rf %s", shellEscape(tmpDir))

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

// LoadImage 导入镜像
func (h *DockerHandler) LoadImage(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Tar string `json:"tar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tar) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "tar 参数错误"})
		return
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	tmp := fmt.Sprintf("/tmp/lazy-aiops-image-%d.tar", time.Now().UnixNano())
	cmd := fmt.Sprintf("printf '%s' | base64 -d > %s && docker load -i %s && rm -f %s",
		req.Tar,
		shellEscape(tmp),
		shellEscape(tmp),
		shellEscape(tmp),
	)
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

// ListNetworkUsage 网络使用情况
func (h *DockerHandler) ListNetworkUsage(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	netIDs, _, err := client.Execute("docker network ls -q")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	ids := strings.Fields(netIDs)
	if len(ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []map[string]interface{}{}})
		return
	}

	cmd := fmt.Sprintf("docker network inspect %s", strings.Join(ids, " "))
	stdout, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析失败"})
		return
	}

	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		name, _ := item["Name"].(string)
		idVal, _ := item["Id"].(string)
		driver, _ := item["Driver"].(string)
		scope, _ := item["Scope"].(string)
		containers := make([]string, 0)
		if raw, ok := item["Containers"].(map[string]interface{}); ok {
			for _, v := range raw {
				if m, ok := v.(map[string]interface{}); ok {
					if n, ok := m["Name"].(string); ok && n != "" {
						containers = append(containers, n)
					}
				}
			}
		}
		result = append(result, map[string]interface{}{
			"id":         idVal,
			"name":       name,
			"driver":     driver,
			"scope":      scope,
			"containers": containers,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListVolumeUsage 卷使用情况
func (h *DockerHandler) ListVolumeUsage(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	volStdout, _, err := client.Execute("docker volume ls --format '{{.Name}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	volNames := strings.Fields(volStdout)
	usageMap := make(map[string][]string)
	for _, v := range volNames {
		usageMap[v] = []string{}
	}

	contStdout, _, err := client.Execute("docker ps -aq")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	contIDs := strings.Fields(contStdout)
	if len(contIDs) > 0 {
		inspectCmd := fmt.Sprintf("docker inspect %s", strings.Join(contIDs, " "))
		inspectOut, stderr, err := client.Execute(inspectCmd)
		if err != nil {
			msg := strings.TrimSpace(stderr)
			if msg == "" {
				msg = err.Error()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
			return
		}
		var containers []map[string]interface{}
		if err := json.Unmarshal([]byte(inspectOut), &containers); err == nil {
			for _, item := range containers {
				name, _ := item["Name"].(string)
				name = strings.TrimPrefix(name, "/")
				mounts, _ := item["Mounts"].([]interface{})
				for _, m := range mounts {
					mv, ok := m.(map[string]interface{})
					if !ok {
						continue
					}
					if typ, _ := mv["Type"].(string); typ != "volume" {
						continue
					}
					volName, _ := mv["Name"].(string)
					if volName == "" {
						continue
					}
					usageMap[volName] = append(usageMap[volName], name)
				}
			}
		}
	}

	result := make([]map[string]interface{}, 0, len(usageMap))
	for name, containers := range usageMap {
		result = append(result, map[string]interface{}{
			"name":       name,
			"containers": containers,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
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

type serviceInspect struct {
	Spec serviceSpec `json:"Spec"`
}

type serviceSpec struct {
	Labels       map[string]string `json:"Labels"`
	Mode         serviceMode       `json:"Mode"`
	TaskTemplate serviceTask       `json:"TaskTemplate"`
	EndpointSpec serviceEndpoint   `json:"EndpointSpec"`
}

type serviceMode struct {
	Replicated serviceReplicated `json:"Replicated"`
}

type serviceReplicated struct {
	Replicas uint64 `json:"Replicas"`
}

type serviceTask struct {
	ContainerSpec serviceContainerSpec `json:"ContainerSpec"`
	Networks      []serviceNetwork     `json:"Networks"`
	Placement     servicePlacement     `json:"Placement"`
}

type serviceContainerSpec struct {
	Env    []string          `json:"Env"`
	Labels map[string]string `json:"Labels"`
	Image  string            `json:"Image"`
}

type serviceNetwork struct {
	Target string `json:"Target"`
}

type servicePlacement struct {
	Constraints []string `json:"Constraints"`
}

type serviceEndpoint struct {
	Ports []servicePort `json:"Ports"`
}

type servicePort struct {
	PublishedPort int    `json:"PublishedPort"`
	TargetPort    int    `json:"TargetPort"`
	Protocol      string `json:"Protocol"`
	PublishMode   string `json:"PublishMode"`
}

func parseEnvKeys(env []string) []string {
	keys := make([]string, 0, len(env))
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// CreateService 创建 Swarm 服务
func (h *DockerHandler) CreateService(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name                  string            `json:"name" binding:"required"`
		Image                 string            `json:"image" binding:"required"`
		Mode                  string            `json:"mode"` // replicated/global
		EndpointMode          string            `json:"endpoint_mode"`
		Replicas              *int              `json:"replicas"`
		Ports                 []string          `json:"ports"`
		Env                   map[string]string `json:"env"`
		Labels                map[string]string `json:"labels"`
		Networks              []string          `json:"networks"`
		Constraints           []string          `json:"constraints"`
		PlacementPrefs        []string          `json:"placement_prefs"`
		MaxReplicasPerNode    *int              `json:"max_replicas_per_node"`
		Mounts                []string          `json:"mounts"`
		Command               []string          `json:"command"`
		RestartCondition      string            `json:"restart_condition"`
		UpdateParallelism     *int              `json:"update_parallelism"`
		UpdateDelay           string            `json:"update_delay"`
		UpdateFailureAction   string            `json:"update_failure_action"`
		UpdateOrder           string            `json:"update_order"`
		RollbackParallelism   *int              `json:"rollback_parallelism"`
		RollbackDelay         string            `json:"rollback_delay"`
		RollbackFailureAction string            `json:"rollback_failure_action"`
		RollbackOrder         string            `json:"rollback_order"`
		LimitCPU              string            `json:"limit_cpu"`
		LimitMemory           string            `json:"limit_memory"`
		ReserveCPU            string            `json:"reserve_cpu"`
		ReserveMemory         string            `json:"reserve_memory"`
		MaxReplicasPerNode    *int              `json:"max_replicas_per_node"`
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

	var cmdBuilder strings.Builder
	cmdBuilder.WriteString("docker service create")
	cmdBuilder.WriteString(" --name " + shellEscape(req.Name))
	if strings.ToLower(req.Mode) == "global" {
		cmdBuilder.WriteString(" --mode global")
	} else if req.Replicas != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --replicas %d", *req.Replicas))
	}
	if req.EndpointMode != "" {
		cmdBuilder.WriteString(" --endpoint-mode " + shellEscape(req.EndpointMode))
	}
	for _, p := range req.Ports {
		p = strings.TrimSpace(p)
		if p != "" {
			cmdBuilder.WriteString(" --publish " + shellEscape(p))
		}
	}
	for k, v := range req.Env {
		cmdBuilder.WriteString(" --env " + shellEscape(fmt.Sprintf("%s=%s", k, v)))
	}
	for k, v := range req.Labels {
		cmdBuilder.WriteString(" --label " + shellEscape(fmt.Sprintf("%s=%s", k, v)))
	}
	for _, n := range req.Networks {
		n = strings.TrimSpace(n)
		if n != "" {
			cmdBuilder.WriteString(" --network " + shellEscape(n))
		}
	}
	for _, cst := range req.Constraints {
		cst = strings.TrimSpace(cst)
		if cst != "" {
			cmdBuilder.WriteString(" --constraint " + shellEscape(cst))
		}
	}
	for _, pref := range req.PlacementPrefs {
		pref = strings.TrimSpace(pref)
		if pref != "" {
			cmdBuilder.WriteString(" --placement-pref " + shellEscape(pref))
		}
	}
	if req.MaxReplicasPerNode != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --replicas-max-per-node %d", *req.MaxReplicasPerNode))
	}
	for _, m := range req.Mounts {
		m = strings.TrimSpace(m)
		if m != "" {
			cmdBuilder.WriteString(" --mount " + shellEscape(m))
		}
	}
	if req.RestartCondition != "" {
		cmdBuilder.WriteString(" --restart-condition " + shellEscape(req.RestartCondition))
	}
	if req.UpdateParallelism != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --update-parallelism %d", *req.UpdateParallelism))
	}
	if req.UpdateDelay != "" {
		cmdBuilder.WriteString(" --update-delay " + shellEscape(req.UpdateDelay))
	}
	if req.UpdateFailureAction != "" {
		cmdBuilder.WriteString(" --update-failure-action " + shellEscape(req.UpdateFailureAction))
	}
	if req.UpdateOrder != "" {
		cmdBuilder.WriteString(" --update-order " + shellEscape(req.UpdateOrder))
	}
	if req.RollbackParallelism != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --rollback-parallelism %d", *req.RollbackParallelism))
	}
	if req.RollbackDelay != "" {
		cmdBuilder.WriteString(" --rollback-delay " + shellEscape(req.RollbackDelay))
	}
	if req.RollbackFailureAction != "" {
		cmdBuilder.WriteString(" --rollback-failure-action " + shellEscape(req.RollbackFailureAction))
	}
	if req.RollbackOrder != "" {
		cmdBuilder.WriteString(" --rollback-order " + shellEscape(req.RollbackOrder))
	}
	if req.LimitCPU != "" {
		cmdBuilder.WriteString(" --limit-cpu " + shellEscape(req.LimitCPU))
	}
	if req.LimitMemory != "" {
		cmdBuilder.WriteString(" --limit-memory " + shellEscape(req.LimitMemory))
	}
	if req.ReserveCPU != "" {
		cmdBuilder.WriteString(" --reserve-cpu " + shellEscape(req.ReserveCPU))
	}
	if req.ReserveMemory != "" {
		cmdBuilder.WriteString(" --reserve-memory " + shellEscape(req.ReserveMemory))
	}
	if req.MaxReplicasPerNode != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --replicas-max-per-node %d", *req.MaxReplicasPerNode))
	}
	cmdBuilder.WriteString(" " + shellEscape(req.Image))
	for _, arg := range req.Command {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			cmdBuilder.WriteString(" " + shellEscape(arg))
		}
	}

	stdout, stderr, err := client.Execute(cmdBuilder.String())
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "service_id": strings.TrimSpace(stdout)})
}

// UpdateService 更新 Swarm 服务
func (h *DockerHandler) UpdateService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	var req struct {
		Image                 string            `json:"image"`
		Mode                  string            `json:"mode"`
		EndpointMode          string            `json:"endpoint_mode"`
		Replicas              *int              `json:"replicas"`
		Env                   map[string]string `json:"env"`
		Labels                map[string]string `json:"labels"`
		Ports                 []string          `json:"ports"`
		Networks              []string          `json:"networks"`
		Constraints           []string          `json:"constraints"`
		ResetEnv              bool              `json:"reset_env"`
		ResetLabels           bool              `json:"reset_labels"`
		ResetPorts            bool              `json:"reset_ports"`
		ResetNetworks         bool              `json:"reset_networks"`
		ResetConstraints      bool              `json:"reset_constraints"`
		RestartCondition      string            `json:"restart_condition"`
		UpdateParallelism     *int              `json:"update_parallelism"`
		UpdateDelay           string            `json:"update_delay"`
		UpdateFailureAction   string            `json:"update_failure_action"`
		UpdateOrder           string            `json:"update_order"`
		RollbackParallelism   *int              `json:"rollback_parallelism"`
		RollbackDelay         string            `json:"rollback_delay"`
		RollbackFailureAction string            `json:"rollback_failure_action"`
		RollbackOrder         string            `json:"rollback_order"`
		LimitCPU              string            `json:"limit_cpu"`
		LimitMemory           string            `json:"limit_memory"`
		ReserveCPU            string            `json:"reserve_cpu"`
		ReserveMemory         string            `json:"reserve_memory"`
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

	var current serviceInspect
	if req.ResetEnv || req.ResetLabels || req.ResetPorts || req.ResetNetworks || req.ResetConstraints {
		stdout, stderr, err := client.Execute(fmt.Sprintf("docker service inspect %s --format '{{json .}}'", serviceID))
		if err != nil {
			msg := strings.TrimSpace(stderr)
			if msg == "" {
				msg = err.Error()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
			return
		}
		_ = json.Unmarshal([]byte(stdout), &current)
	}

	var cmdBuilder strings.Builder
	cmdBuilder.WriteString("docker service update")
	if req.Image != "" {
		cmdBuilder.WriteString(" --image " + shellEscape(req.Image))
	}
	if strings.ToLower(req.Mode) == "global" {
		cmdBuilder.WriteString(" --mode global")
	} else if strings.ToLower(req.Mode) == "replicated" {
		cmdBuilder.WriteString(" --mode replicated")
	}
	if req.EndpointMode != "" {
		cmdBuilder.WriteString(" --endpoint-mode " + shellEscape(req.EndpointMode))
	}
	if req.Replicas != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --replicas %d", *req.Replicas))
	}
	if req.ResetEnv {
		for _, key := range parseEnvKeys(current.Spec.TaskTemplate.ContainerSpec.Env) {
			cmdBuilder.WriteString(" --env-rm " + shellEscape(key))
		}
	}
	for k, v := range req.Env {
		cmdBuilder.WriteString(" --env-add " + shellEscape(fmt.Sprintf("%s=%s", k, v)))
	}
	if req.ResetLabels {
		for key := range current.Spec.Labels {
			cmdBuilder.WriteString(" --label-rm " + shellEscape(key))
		}
	}
	for k, v := range req.Labels {
		cmdBuilder.WriteString(" --label-add " + shellEscape(fmt.Sprintf("%s=%s", k, v)))
	}
	if req.ResetPorts {
		for _, p := range current.Spec.EndpointSpec.Ports {
			if p.PublishedPort > 0 {
				cmdBuilder.WriteString(" --publish-rm " + shellEscape(strconv.Itoa(p.PublishedPort)))
			} else if p.TargetPort > 0 {
				cmdBuilder.WriteString(" --publish-rm " + shellEscape(strconv.Itoa(p.TargetPort)))
			}
		}
	}
	for _, p := range req.Ports {
		p = strings.TrimSpace(p)
		if p != "" {
			cmdBuilder.WriteString(" --publish-add " + shellEscape(p))
		}
	}
	if req.ResetNetworks {
		for _, n := range current.Spec.TaskTemplate.Networks {
			if n.Target != "" {
				cmdBuilder.WriteString(" --network-rm " + shellEscape(n.Target))
			}
		}
	}
	for _, n := range req.Networks {
		n = strings.TrimSpace(n)
		if n != "" {
			cmdBuilder.WriteString(" --network-add " + shellEscape(n))
		}
	}
	if req.ResetConstraints {
		for _, cst := range current.Spec.TaskTemplate.Placement.Constraints {
			cst = strings.TrimSpace(cst)
			if cst != "" {
				cmdBuilder.WriteString(" --constraint-rm " + shellEscape(cst))
			}
		}
	}
	for _, cst := range req.Constraints {
		cst = strings.TrimSpace(cst)
		if cst != "" {
			cmdBuilder.WriteString(" --constraint-add " + shellEscape(cst))
		}
	}
	if req.RestartCondition != "" {
		cmdBuilder.WriteString(" --restart-condition " + shellEscape(req.RestartCondition))
	}
	if req.UpdateParallelism != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --update-parallelism %d", *req.UpdateParallelism))
	}
	if req.UpdateDelay != "" {
		cmdBuilder.WriteString(" --update-delay " + shellEscape(req.UpdateDelay))
	}
	if req.UpdateFailureAction != "" {
		cmdBuilder.WriteString(" --update-failure-action " + shellEscape(req.UpdateFailureAction))
	}
	if req.UpdateOrder != "" {
		cmdBuilder.WriteString(" --update-order " + shellEscape(req.UpdateOrder))
	}
	if req.RollbackParallelism != nil {
		cmdBuilder.WriteString(fmt.Sprintf(" --rollback-parallelism %d", *req.RollbackParallelism))
	}
	if req.RollbackDelay != "" {
		cmdBuilder.WriteString(" --rollback-delay " + shellEscape(req.RollbackDelay))
	}
	if req.RollbackFailureAction != "" {
		cmdBuilder.WriteString(" --rollback-failure-action " + shellEscape(req.RollbackFailureAction))
	}
	if req.RollbackOrder != "" {
		cmdBuilder.WriteString(" --rollback-order " + shellEscape(req.RollbackOrder))
	}
	if req.LimitCPU != "" {
		cmdBuilder.WriteString(" --limit-cpu " + shellEscape(req.LimitCPU))
	}
	if req.LimitMemory != "" {
		cmdBuilder.WriteString(" --limit-memory " + shellEscape(req.LimitMemory))
	}
	if req.ReserveCPU != "" {
		cmdBuilder.WriteString(" --reserve-cpu " + shellEscape(req.ReserveCPU))
	}
	if req.ReserveMemory != "" {
		cmdBuilder.WriteString(" --reserve-memory " + shellEscape(req.ReserveMemory))
	}

	cmdBuilder.WriteString(" " + shellEscape(serviceID))
	_, stderr, err := client.Execute(cmdBuilder.String())
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// RemoveService 删除 Swarm 服务
func (h *DockerHandler) RemoveService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_, stderr, err := client.Execute(fmt.Sprintf("docker service rm %s", serviceID))
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

	_, stderr, err := client.Execute(fmt.Sprintf("docker stack rm %s", shellEscape(stack)))
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
	cmd := fmt.Sprintf("docker service scale %s=%d", shellEscape(serviceID), req.Replicas)
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
	cmd := fmt.Sprintf("docker service update --image %s %s", shellEscape(req.Image), shellEscape(serviceID))
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
	cmd := fmt.Sprintf("docker service update --force %s", shellEscape(serviceID))
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已触发重启"})
}

// RollbackService 回滚服务
func (h *DockerHandler) RollbackService(c *gin.Context) {
	id := c.Param("id")
	serviceID := c.Param("service_id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	cmd := fmt.Sprintf("docker service rollback %s", shellEscape(serviceID))
	_, stderr, err := client.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已触发回滚"})
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

func sanitizeRepoPath(p string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(p))
	if clean == "." || clean == "" {
		return "docker-compose.yml", nil
	}
	if filepath.IsAbs(clean) {
		return "", errors.New("compose_path 不能为绝对路径")
	}
	if strings.HasPrefix(clean, "..") {
		return "", errors.New("compose_path 非法")
	}
	return clean, nil
}

func withBasicAuthURL(raw, user, pass string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return raw, err
	}
	if u.Scheme == "" {
		return raw, errors.New("repo URL 无效")
	}
	u.User = url.UserPassword(user, pass)
	return u.String(), nil
}

func shellEscape(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
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
