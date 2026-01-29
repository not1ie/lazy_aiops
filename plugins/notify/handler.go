package notify

import (
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
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
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
	c.ShouldBindJSON(&req)

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
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
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
