package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

// ListContainers 容器列表
func (h *DockerHandler) ListContainers(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 远程执行 docker ps
	stdout, _, err := client.Execute("docker ps -a --format '{{json .}}'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	containers := make([]DockerContainer, 0)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err == nil {
			containers = append(containers, DockerContainer{
				ID:     raw["ID"].(string),
				Names:  []string{raw["Names"].(string)},
				Image:  raw["Image"].(string),
				State:  raw["State"].(string),
				Status: raw["Status"].(string),
				Ports:  raw["Ports"].(string),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": containers})
}

// Action 容器操作
func (h *DockerHandler) Action(c *gin.Context) {
	id := c.Param("id")
	containerID := c.Param("container_id")
	action := c.Param("action") // start, stop, restart

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if action != "start" && action != "stop" && action != "restart" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的操作"})
		return
	}

	_, stderr, err := client.Execute(fmt.Sprintf("docker %s %s", action, containerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": stderr})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "操作成功"})
}

func (h *DockerHandler) getClient(dockerHostID string) (*core.SSHClient, error) {
	var dHost DockerHost
	if err := h.db.First(&dHost, "id = ?", dockerHostID).Error; err != nil {
		return nil, fmt.Errorf("Docker主机不存在")
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
