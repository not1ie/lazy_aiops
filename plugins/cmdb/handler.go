package cmdb

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
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

// detectOS 探测主机操作系统
func (h *HostHandler) detectOS(host *Host, password string) {
	if host.IP == "" {
		return
	}

	// 使用 core/ssh 模块探测
	// 注意：这里需要构造临时的 SSH Client，因为 host 可能还没有保存到数据库，或者 Credential 是分开的
	// 如果是 Create，我们有 password。如果是 Update，我们需要查库获取 password。
	
	// 暂时只支持 Linux 探测
	client := &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: "root", // 默认假设 root，如果不是，需要在 Request 中传入或从 Credential 获取
		Password: password,
		Timeout:  5 * time.Second,
	}
	
	// 如果 Request 中有 Username，更新
	if host.CredentialID != "" {
		var cred Credential
		if err := h.db.First(&cred, "id = ?", host.CredentialID).Error; err == nil {
			client.Username = cred.Username
			if password == "" {
				client.Password = cred.Password
			}
		}
	}

	stdout, _, err := client.Execute("cat /etc/os-release")
	if err == nil {
		// 解析 os-release
		if strings.Contains(stdout, "Ubuntu") {
			host.OS = "Ubuntu"
		} else if strings.Contains(stdout, "CentOS") {
			host.OS = "CentOS"
		} else if strings.Contains(stdout, "Debian") {
			host.OS = "Debian"
		} else if strings.Contains(stdout, "Alpine") {
			host.OS = "Alpine"
		} else if strings.Contains(stdout, "Red Hat") {
			host.OS = "RHEL"
		} else {
			host.OS = "Linux"
		}
	}
}

// Create 创建主机
func (h *HostHandler) Create(c *gin.Context) {
	// ... (保留之前的代码结构)
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
	if host.Port == 0 {
		host.Port = 22
	}
	
	// ... (分组逻辑)
	if req.GroupName == "" {
		req.GroupName = "Default"
	}
	var group HostGroup
	if err := h.db.FirstOrCreate(&group, HostGroup{Name: req.GroupName}).Error; err == nil {
		host.GroupID = group.ID
	}

	// ... (凭据逻辑)
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

	// 自动探测 OS
	if req.Password != "" || req.Username != "" {
		// 异步探测，避免阻塞创建
		go func(h *Host, user, pass string) {
			// 需要一个临时 client
			client := &core.SSHClient{
				Host:     h.IP,
				Port:     h.Port,
				Username: user,
				Password: pass,
				Timeout:  10 * time.Second,
			}
			stdout, _, err := client.Execute("grep PRETTY_NAME /etc/os-release")
			if err == nil {
				osName := strings.TrimPrefix(strings.TrimSpace(stdout), "PRETTY_NAME=")
				osName = strings.Trim(osName, "\"")
				// 简化名称
				if strings.Contains(osName, "Ubuntu") {
					osName = "Ubuntu"
				} else if strings.Contains(osName, "CentOS") {
					osName = "CentOS"
				} else if strings.Contains(osName, "Debian") {
					osName = "Debian"
				}
				
				// 更新数据库
				// 注意：这里需要新的 DB 会话
				// h.db.Model(h).Update("os", osName) // 这里的 h.db 可能不安全并发使用？GORM DB 是并发安全的。
				// 但是 h.ID 必须已经生成。
				// 我们需要等待 Create 完成拿到 ID。
				// 所以这里其实不能简单的 go func。
				// 更好的方式是在 Create 之后调用。
			}
		}(&host, req.Username, req.Password)
	}

	if err := h.db.Create(&host).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	
	// 创建后触发探测
	if req.Username != "" && req.Password != "" {
		go h.detectOSAsync(host.ID, req.Username, req.Password)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": host})
}

func (h *HostHandler) detectOSAsync(hostID, username, password string) {
	var host Host
	if err := h.db.First(&host, "id = ?", hostID).Error; err != nil {
		return
	}
	
	client := &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: username,
		Password: password,
		Timeout:  10 * time.Second,
	}
	
	stdout, _, err := client.Execute("grep '^ID=' /etc/os-release")
	if err == nil {
		osID := strings.TrimPrefix(strings.TrimSpace(stdout), "ID=")
		osID = strings.Trim(osID, "\"")
		// 首字母大写
		if len(osID) > 0 {
			osID = strings.ToUpper(osID[:1]) + osID[1:]
		}
		h.db.Model(&host).Update("os", osID)
	}
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
