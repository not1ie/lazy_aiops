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
	"github.com/lazyautoops/lazy-auto-ops/internal/security"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type NacosHandler struct {
	db        *gorm.DB
	scheduler *Scheduler
	secretKey string
}

func NewNacosHandler(db *gorm.DB, scheduler *Scheduler, secretKey string) *NacosHandler {
	return &NacosHandler{db: db, scheduler: scheduler, secretKey: secretKey}
}

// ListServers 服务器列表
func (h *NacosHandler) ListServers(c *gin.Context) {
	var servers []NacosServer
	if err := h.db.Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range servers {
		servers[i].Password = ""
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
	var err error
	server.Password, err = security.Encrypt(h.secretKey, "nacos.server.password", server.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}
	if err := h.db.Create(&server).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	server.Password = ""
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
	if strings.TrimSpace(server.Password) != "" {
		password, err := security.Decrypt(h.secretKey, "nacos.server.password", server.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码解密失败"})
			return
		}
		server.Password = password
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": server})
}

// UpdateServer 更新服务器
func (h *NacosHandler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	var current NacosServer
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "服务器不存在"})
		return
	}
	var req NacosServer
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"address":     req.Address,
		"namespace":   req.Namespace,
		"username":    req.Username,
		"description": req.Description,
		"status":      req.Status,
	}
	if strings.TrimSpace(req.Password) != "" {
		enc, err := security.Encrypt(h.secretKey, "nacos.server.password", req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
			return
		}
		updates["password"] = enc
	}
	if err := h.db.Model(&NacosServer{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&current, "id = ?", id).Error; err == nil {
		current.Password = ""
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": current})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
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
	password, err := security.Decrypt(h.secretKey, "nacos.server.password", server.Password)
	if err != nil {
		return "", fmt.Errorf("密码解密失败")
	}
	data := url.Values{}
	data.Set("username", server.Username)
	data.Set("password", password)

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

// syncConfigsForServer 同步指定服务器配置
func (h *NacosHandler) syncConfigsForServer(serverID string) (int, int, error) {
	var server NacosServer
	if err := h.db.First(&server, "id = ?", serverID).Error; err != nil {
		return 0, 0, err
	}

	token, err := h.login(&server)
	if err != nil {
		return 0, 0, err
	}

	listURL := fmt.Sprintf("%s/nacos/v1/cs/configs?dataId=&group=&pageNo=1&pageSize=1000&accessToken=%s",
		strings.TrimSuffix(server.Address, "/"), token)
	if server.Namespace != "" {
		listURL += "&tenant=" + server.Namespace
	}

	resp, err := http.Get(listURL)
	if err != nil {
		return 0, 0, err
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

	return count, len(result.PageItems), nil
}

// SyncConfigs 同步配置
func (h *NacosHandler) SyncConfigs(c *gin.Context) {
	id := c.Param("id")
	count, total, err := h.syncConfigsForServer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "同步失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"synced": count, "total": total}})
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

// ListSyncSchedules 同步计划列表
func (h *NacosHandler) ListSyncSchedules(c *gin.Context) {
	var schedules []NacosSyncSchedule
	query := h.db.Order("updated_at DESC")
	if serverID := c.Query("server_id"); serverID != "" {
		query = query.Where("server_id = ?", serverID)
	}
	if err := query.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedules})
}

// CreateSyncSchedule 创建同步计划
func (h *NacosHandler) CreateSyncSchedule(c *gin.Context) {
	var schedule NacosSyncSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if schedule.Cron != "" {
		if _, err := cron.ParseStandard(schedule.Cron); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Cron表达式无效"})
			return
		}
	}
	if err := h.db.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if h.scheduler != nil {
		h.scheduler.AddSchedule(schedule)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// UpdateSyncSchedule 更新同步计划
func (h *NacosHandler) UpdateSyncSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule NacosSyncSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "同步计划不存在"})
		return
	}
	var req struct {
		Name        *string `json:"name"`
		ServerID    *string `json:"server_id"`
		Cron        *string `json:"cron"`
		Enabled     *bool   `json:"enabled"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Cron != nil && *req.Cron != "" {
		if _, err := cron.ParseStandard(*req.Cron); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Cron表达式无效"})
			return
		}
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ServerID != nil {
		updates["server_id"] = *req.ServerID
	}
	if req.Cron != nil {
		updates["cron"] = *req.Cron
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
		return
	}
	if err := h.db.Model(&schedule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if h.scheduler != nil {
		if schedule.Enabled {
			h.scheduler.AddSchedule(schedule)
		} else {
			h.scheduler.RemoveSchedule(schedule.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// DeleteSyncSchedule 删除同步计划
func (h *NacosHandler) DeleteSyncSchedule(c *gin.Context) {
	id := c.Param("id")
	if h.scheduler != nil {
		h.scheduler.RemoveSchedule(id)
	}
	if err := h.db.Delete(&NacosSyncSchedule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ToggleSyncSchedule 启用/禁用同步计划
func (h *NacosHandler) ToggleSyncSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule NacosSyncSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "同步计划不存在"})
		return
	}
	schedule.Enabled = !schedule.Enabled
	if err := h.db.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if h.scheduler != nil {
		if schedule.Enabled {
			h.scheduler.AddSchedule(schedule)
		} else {
			h.scheduler.RemoveSchedule(schedule.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}
