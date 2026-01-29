package cmdb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HostHandler struct {
	db *gorm.DB
}

func NewHostHandler(db *gorm.DB) *HostHandler {
	return &HostHandler{db: db}
}

// List 主机列表
func (h *HostHandler) List(c *gin.Context) {
	var hosts []Host
	query := h.db.Preload("Group").Preload("Credential")

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if groupID := c.Query("group_id"); groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}

	if err := query.Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": hosts})
}

// Create 创建主机
func (h *HostHandler) Create(c *gin.Context) {
	// 定义请求结构，包含 group_name 和 凭据信息
	var req struct {
		Host
		GroupName string `json:"group_name"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	host := req.Host
	
	// 1. 处理分组 (默认分组)
	if req.GroupName == "" {
		req.GroupName = "Default"
	}
	var group HostGroup
	if err := h.db.FirstOrCreate(&group, HostGroup{Name: req.GroupName}).Error; err == nil {
		host.GroupID = group.ID
	}

	// 2. 处理凭据 (自动创建)
	if req.Username != "" {
		cred := Credential{
			Name:     host.Name + "-cred",
			Type:     "password",
			Username: req.Username,
			Password: req.Password,
		}
		if err := h.db.Create(&cred).Error; err == nil {
			host.CredentialID = cred.ID
		}
	}

	if err := h.db.Create(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

// Get 获取主机详情
func (h *HostHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var host Host
	if err := h.db.Preload("Group").Preload("Credential").First(&host, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "主机不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

// Update 更新主机
func (h *HostHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var host Host
	if err := h.db.First(&host, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "主机不存在"})
		return
	}

	var req struct {
		Host
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 更新基本字段
	host.Name = req.Name
	host.IP = req.IP
	host.Port = req.Port
	host.OS = req.OS
	// host.Description = req.Description // 如果需要支持更多字段

	// 更新或创建凭据
	if req.Username != "" {
		if host.CredentialID != "" {
			// 更新现有凭据
			h.db.Model(&Credential{}).Where("id = ?", host.CredentialID).Updates(map[string]interface{}{
				"username": req.Username,
				"password": req.Password,
			})
		} else {
			// 创建新凭据
			cred := Credential{
				Name:     host.Name + "-cred",
				Type:     "password",
				Username: req.Username,
				Password: req.Password,
			}
			if err := h.db.Create(&cred).Error; err == nil {
				host.CredentialID = cred.ID
			}
		}
	}

	if err := h.db.Save(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

// Delete 删除主机
func (h *HostHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Host{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListGroups 分组列表
func (h *HostHandler) ListGroups(c *gin.Context) {
	var groups []HostGroup
	if err := h.db.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateGroup 创建分组
func (h *HostHandler) CreateGroup(c *gin.Context) {
	var group HostGroup
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

// ListCredentials 凭据列表
func (h *HostHandler) ListCredentials(c *gin.Context) {
	var creds []Credential
	if err := h.db.Find(&creds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": creds})
}

// CreateCredential 创建凭据
func (h *HostHandler) CreateCredential(c *gin.Context) {
	var cred Credential
	if err := c.ShouldBindJSON(&cred); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&cred).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cred})
}
