package notify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotifyHandler struct {
	db     *gorm.DB
	sender *Sender
}

func NewNotifyHandler(db *gorm.DB) *NotifyHandler {
	return &NotifyHandler{db: db, sender: NewSender()}
}

// ListChannels 渠道列表
func (h *NotifyHandler) ListChannels(c *gin.Context) {
	var channels []NotifyChannel
	if err := h.db.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": channels})
}

// CreateChannel 创建渠道
func (h *NotifyHandler) CreateChannel(c *gin.Context) {
	var channel NotifyChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": channel})
}

// UpdateChannel 更新渠道
func (h *NotifyHandler) UpdateChannel(c *gin.Context) {
	id := c.Param("id")
	var channel NotifyChannel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "渠道不存在"})
		return
	}
	var req NotifyChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":         req.Name,
		"type":         req.Type,
		"webhook":      req.Webhook,
		"secret":       req.Secret,
		"app_id":       req.AppID,
		"app_secret":   req.AppSecret,
		"smtp_host":    req.SMTPHost,
		"smtp_port":    req.SMTPPort,
		"smtp_user":    req.SMTPUser,
		"smtp_pass":    req.SMTPPass,
		"sms_provider": req.SMSProvider,
		"sms_sign":     req.SMSSign,
		"sms_template": req.SMSTemplate,
		"enabled":      req.Enabled,
		"description":  req.Description,
	}
	if err := h.db.Model(&channel).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&channel, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": channel})
}

// DeleteChannel 删除渠道
func (h *NotifyHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&NotifyChannel{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestChannel 测试渠道
func (h *NotifyHandler) TestChannel(c *gin.Context) {
	id := c.Param("id")
	var channel NotifyChannel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "渠道不存在"})
		return
	}

	var req struct {
		Receiver string `json:"receiver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	title := "Lazy Auto Ops 测试通知"
	content := "这是一条测试消息，如果您收到此消息，说明通知渠道配置正确。\n\n发送时间: " + time.Now().Format("2006-01-02 15:04:05")

	if err := h.sender.Send(&channel, title, content, req.Receiver); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "发送失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发送成功"})
}

// SendNotify 发送通知
func (h *NotifyHandler) SendNotify(c *gin.Context) {
	var req struct {
		ChannelID string `json:"channel_id" binding:"required"`
		Title     string `json:"title" binding:"required"`
		Content   string `json:"content" binding:"required"`
		Receiver  string `json:"receiver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var channel NotifyChannel
	if err := h.db.First(&channel, "id = ?", req.ChannelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "渠道不存在"})
		return
	}

	// 创建记录
	record := NotifyRecord{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Title:       req.Title,
		Content:     req.Content,
		Receiver:    req.Receiver,
		Status:      0,
		Source:      "manual",
	}
	h.db.Create(&record)

	// 发送
	if err := h.sender.Send(&channel, req.Title, req.Content, req.Receiver); err != nil {
		h.db.Model(&record).Updates(map[string]interface{}{
			"status":    2,
			"error_msg": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	now := time.Now()
	h.db.Model(&record).Updates(map[string]interface{}{
		"status":  1,
		"sent_at": now,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发送成功"})
}

// ListRecords 通知记录
func (h *NotifyHandler) ListRecords(c *gin.Context) {
	var records []NotifyRecord
	query := h.db.Order("created_at DESC")

	if channelID := c.Query("channel_id"); channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Limit(200).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// ListGroups 通知组列表
func (h *NotifyHandler) ListGroups(c *gin.Context) {
	var groups []NotifyGroup
	if err := h.db.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateGroup 创建通知组
func (h *NotifyHandler) CreateGroup(c *gin.Context) {
	var group NotifyGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": group})
}

// UpdateGroup 更新通知组
func (h *NotifyHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var group NotifyGroup
	if err := h.db.First(&group, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知组不存在"})
		return
	}
	var req NotifyGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"channels":    req.Channels,
		"users":       req.Users,
	}
	if err := h.db.Model(&group).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&group, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": group})
}

// DeleteGroup 删除通知组
func (h *NotifyHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&NotifyGroup{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestGroup 测试通知组
func (h *NotifyHandler) TestGroup(c *gin.Context) {
	id := c.Param("id")
	var group NotifyGroup
	if err := h.db.First(&group, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "通知组不存在"})
		return
	}

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Receiver string `json:"receiver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var channelIDs []string
	if group.Channels != "" {
		_ = json.Unmarshal([]byte(group.Channels), &channelIDs)
	}
	if len(channelIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "通知组未配置渠道"})
		return
	}

	var failed []string
	for _, cid := range channelIDs {
		var channel NotifyChannel
		if err := h.db.First(&channel, "id = ?", cid).Error; err != nil {
			failed = append(failed, cid+":not_found")
			continue
		}
		if err := h.sender.Send(&channel, req.Title, req.Content, req.Receiver); err != nil {
			failed = append(failed, cid+":"+err.Error())
		}
	}

	if len(failed) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "部分发送失败", "data": failed})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发送成功"})
}

// ListTemplates 模板列表
func (h *NotifyHandler) ListTemplates(c *gin.Context) {
	var templates []NotifyTemplate
	if err := h.db.Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": templates})
}

// CreateTemplate 创建模板
func (h *NotifyHandler) CreateTemplate(c *gin.Context) {
	var template NotifyTemplate
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

// UpdateTemplate 更新模板
func (h *NotifyHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	var template NotifyTemplate
	if err := h.db.First(&template, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}
	var req NotifyTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":         req.Name,
		"type":         req.Type,
		"title":        req.Title,
		"content":      req.Content,
		"channel_type": req.ChannelType,
		"enabled":      req.Enabled,
	}
	if err := h.db.Model(&template).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&template, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": template})
}

// DeleteTemplate 删除模板
func (h *NotifyHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&NotifyTemplate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
