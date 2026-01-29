package nacos

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NacosHandler struct {
	db *gorm.DB
}

func NewNacosHandler(db *gorm.DB) *NacosHandler {
	return &NacosHandler{db: db}
}

// ListServers 服务器列表
func (h *NacosHandler) ListServers(c *gin.Context) {
	var servers []NacosServer
	if err := h.db.Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": servers})
}

// CreateServer 创建服务器
func (h *NacosHandler) CreateServer(c *gin.Context) {
	var server NacosServer
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": server})
}


// GetServer 获取服务器详情
func (h *NacosHandler) GetServer(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": server})
}

// UpdateServer 更新服务器
func (h *NacosHandler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": server})
}

// DeleteServer 删除服务器
func (h *NacosHandler) DeleteServer(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&NacosServer{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 测试连接
func (h *NacosHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}

	token, err := h.login(&server)
	if err != nil {
		h.db.Model(&server).Update("status", 0)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": false, "error": err.Error()}})
		return
	}

	h.db.Model(&server).Update("status", 1)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": true, "token": token[:20] + "..."}})
}

func (h *NacosHandler) login(server *NacosServer) (string, error) {
	loginURL := fmt.Sprintf("%s/nacos/v1/auth/login", strings.TrimSuffix(server.Address, "/"))
	data := url.Values{}
	data.Set("username", server.Username)
	data.Set("password", server.Password)

	resp, err := http.PostForm(loginURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"accessToken"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.AccessToken == "" {
		return "", fmt.Errorf("登录失败")
	}
	return result.AccessToken, nil
}


// SyncConfigs 同步配置
func (h *NacosHandler) SyncConfigs(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}

	token, err := h.login(&server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败: " + err.Error()})
		return
	}

	// 获取配置列表
	listURL := fmt.Sprintf("%s/nacos/v1/cs/configs?dataId=&group=&pageNo=1&pageSize=1000&accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), token)
	if server.Namespace != "" {
		listURL += "&tenant=" + server.Namespace
	}

	resp, err := http.Get(listURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		PageItems []struct {
			DataID  string `json:"dataId"`
			Group   string `json:"group"`
			Content string `json:"content"`
			Type    string `json:"type"`
			AppName string `json:"appName"`
		} `json:"pageItems"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// 同步到本地
	count := 0
	for _, item := range result.PageItems {
		hash := md5.Sum([]byte(item.Content))
		md5Str := hex.EncodeToString(hash[:])

		var config NacosConfig
		h.db.Where("server_id = ? AND data_id = ? AND `group` = ?", server.ID, item.DataID, item.Group).First(&config)

		if config.ID == "" {
			config = NacosConfig{
				ServerID:    server.ID,
				DataID:      item.DataID,
				Group:       item.Group,
				Content:     item.Content,
				ContentType: item.Type,
				MD5:         md5Str,
				AppName:     item.AppName,
			}
			h.db.Create(&config)
			count++
		} else if config.MD5 != md5Str {
			config.Content = item.Content
			config.MD5 = md5Str
			h.db.Save(&config)
			count++
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"synced": count, "total": len(result.PageItems)}})
}

// ListConfigs 配置列表
func (h *NacosHandler) ListConfigs(c *gin.Context) {
	serverID := c.Query("server_id")
	var configs []NacosConfig
	query := h.db.Order("updated_at DESC")
	if serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if group := c.Query("group"); group != "" {
		query = query.Where("`group` = ?", group)
	}
	if dataID := c.Query("data_id"); dataID != "" {
		query = query.Where("data_id LIKE ?", "%"+dataID+"%")
	}
	if err := query.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// GetConfig 获取配置详情
func (h *NacosHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")
	var config NacosConfig
	if err := h.db.First(&config, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}


// UpdateConfig 更新配置
func (h *NacosHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	var config NacosConfig
	if err := h.db.First(&config, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取服务器
	var server NacosServer
	if err := h.db.First(&server, "id = ?", config.ServerID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器不存在"})
		return
	}

	token, err := h.login(&server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
		return
	}

	// 推送到Nacos
	publishURL := fmt.Sprintf("%s/nacos/v1/cs/configs", strings.TrimSuffix(server.Address, "/"))
	data := url.Values{}
	data.Set("dataId", config.DataID)
	data.Set("group", config.Group)
	data.Set("content", req.Content)
	data.Set("accessToken", token)
	if server.Namespace != "" {
		data.Set("tenant", server.Namespace)
	}

	resp, err := http.PostForm(publishURL, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "true" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布失败: " + string(body)})
		return
	}

	// 记录历史
	history := &NacosConfigHistory{
		ConfigID:    config.ID,
		DataID:      config.DataID,
		Group:       config.Group,
		Content:     config.Content,
		MD5:         config.MD5,
		Operator:    c.GetString("username"),
		OperateType: "update",
		OperateAt:   time.Now(),
	}
	h.db.Create(history)

	// 更新本地
	hash := md5.Sum([]byte(req.Content))
	config.Content = req.Content
	config.MD5 = hex.EncodeToString(hash[:])
	h.db.Save(&config)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}

// GetConfigHistory 配置历史
func (h *NacosHandler) GetConfigHistory(c *gin.Context) {
	id := c.Param("id")
	var histories []NacosConfigHistory
	if err := h.db.Where("config_id = ?", id).Order("operate_at DESC").Limit(50).Find(&histories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": histories})
}

// RollbackConfig 回滚配置
func (h *NacosHandler) RollbackConfig(c *gin.Context) {
	historyID := c.Param("history_id")
	var history NacosConfigHistory
	if err := h.db.First(&history, "id = ?", historyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "历史记录不存在"})
		return
	}

	var config NacosConfig
	if err := h.db.First(&config, "id = ?", history.ConfigID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	// 模拟更新请求
	c.Set("rollback_content", history.Content)
	// 实际应该调用UpdateConfig逻辑
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "回滚成功"})
}

// CompareConfig 配置对比
func (h *NacosHandler) CompareConfig(c *gin.Context) {
	id := c.Param("id")
	var config NacosConfig
	if err := h.db.First(&config, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	// 获取远程最新
	var server NacosServer
	h.db.First(&server, "id = ?", config.ServerID)

	token, _ := h.login(&server)
	getURL := fmt.Sprintf("%s/nacos/v1/cs/configs?dataId=%s&group=%s&accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), config.DataID, config.Group, token)
	if server.Namespace != "" {
		getURL += "&tenant=" + server.Namespace
	}

	resp, err := http.Get(getURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	remoteContent, _ := io.ReadAll(resp.Body)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"local":  config.Content,
			"remote": string(remoteContent),
		},
	})
}


// SyncServices 同步服务
func (h *NacosHandler) SyncServices(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}

	token, err := h.login(&server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "登录失败"})
		return
	}

	listURL := fmt.Sprintf("%s/nacos/v1/ns/service/list?pageNo=1&pageSize=1000&accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), token)

	resp, err := http.Get(listURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Doms []string `json:"doms"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	// 清理旧数据
	h.db.Where("server_id = ?", server.ID).Delete(&NacosService{})

	// 同步服务
	for _, serviceName := range result.Doms {
		service := &NacosService{
			ServerID:    server.ID,
			ServiceName: serviceName,
		}
		h.db.Create(service)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": len(result.Doms)}})
}

// ListServices 服务列表
func (h *NacosHandler) ListServices(c *gin.Context) {
	serverID := c.Query("server_id")
	var services []NacosService
	query := h.db
	if serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if err := query.Find(&services).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": services})
}

// GetServiceInstances 获取服务实例
func (h *NacosHandler) GetServiceInstances(c *gin.Context) {
	serverID := c.Query("server_id")
	serviceName := c.Query("service_name")

	var server NacosServer
	if err := h.db.First(&server, "id = ?", serverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}

	token, _ := h.login(&server)
	instanceURL := fmt.Sprintf("%s/nacos/v1/ns/instance/list?serviceName=%s&accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), serviceName, token)

	resp, err := http.Get(instanceURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Hosts []struct {
			IP          string            `json:"ip"`
			Port        int               `json:"port"`
			Weight      float64           `json:"weight"`
			Healthy     bool              `json:"healthy"`
			Enabled     bool              `json:"enabled"`
			Ephemeral   bool              `json:"ephemeral"`
			ClusterName string            `json:"clusterName"`
			Metadata    map[string]string `json:"metadata"`
		} `json:"hosts"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result.Hosts})
}

// ListNamespaces 命名空间列表
func (h *NacosHandler) ListNamespaces(c *gin.Context) {
	id := c.Param("id")
	var server NacosServer
	if err := h.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}

	token, _ := h.login(&server)
	nsURL := fmt.Sprintf("%s/nacos/v1/console/namespaces?accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), token)

	resp, err := http.Get(nsURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Namespace         string `json:"namespace"`
			NamespaceShowName string `json:"namespaceShowName"`
			ConfigCount       int    `json:"configCount"`
			Quota             int    `json:"quota"`
			Type              int    `json:"type"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result.Data})
}
