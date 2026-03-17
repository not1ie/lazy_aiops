package notify

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SendByTarget sends notification by notify group id or channel id.
func SendByTarget(db *gorm.DB, targetID, title, content, receiver, source, sourceID string) error {
	if db == nil {
		return fmt.Errorf("通知服务不可用")
	}
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return fmt.Errorf("通知目标不能为空")
	}
	if strings.TrimSpace(source) == "" {
		source = "system"
	}

	var group NotifyGroup
	if err := db.First(&group, "id = ?", targetID).Error; err == nil {
		return sendToGroup(db, &group, title, content, receiver, source, sourceID)
	}

	var channel NotifyChannel
	if err := db.First(&channel, "id = ? AND enabled = ?", targetID, true).Error; err == nil {
		return sendToChannel(db, &channel, title, content, receiver, source, sourceID)
	}

	return fmt.Errorf("通知目标不存在或未启用")
}

func sendToGroup(db *gorm.DB, group *NotifyGroup, title, content, receiver, source, sourceID string) error {
	if group == nil {
		return fmt.Errorf("通知组不存在")
	}
	var channelIDs []string
	if strings.TrimSpace(group.Channels) != "" {
		if err := json.Unmarshal([]byte(group.Channels), &channelIDs); err != nil {
			return fmt.Errorf("通知组渠道配置错误")
		}
	}
	if len(channelIDs) == 0 {
		return fmt.Errorf("通知组未配置渠道")
	}

	failed := make([]string, 0)
	for _, id := range channelIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		var channel NotifyChannel
		if err := db.First(&channel, "id = ? AND enabled = ?", id, true).Error; err != nil {
			failed = append(failed, id+":not_found")
			continue
		}
		if err := sendToChannel(db, &channel, title, content, receiver, source, sourceID); err != nil {
			failed = append(failed, id+":"+err.Error())
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("部分渠道发送失败: %s", strings.Join(failed, "; "))
	}
	return nil
}

func sendToChannel(db *gorm.DB, channel *NotifyChannel, title, content, receiver, source, sourceID string) error {
	if channel == nil {
		return fmt.Errorf("通知渠道不存在")
	}
	sender := NewSender()
	record := NotifyRecord{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		ChannelType: channel.Type,
		Title:       title,
		Content:     content,
		Receiver:    receiver,
		Status:      0,
		Source:      source,
		SourceID:    sourceID,
	}
	_ = db.Create(&record).Error

	if err := sender.Send(channel, title, content, receiver); err != nil {
		_ = db.Model(&record).Updates(map[string]interface{}{
			"status":    2,
			"error_msg": err.Error(),
		}).Error
		return err
	}

	now := time.Now()
	_ = db.Model(&record).Updates(map[string]interface{}{
		"status":  1,
		"sent_at": now,
	}).Error
	return nil
}
