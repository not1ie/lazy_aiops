package cmdb

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/security"
	"gorm.io/gorm"
)

type HostHandler struct {
	db        *gorm.DB
	secretKey string
}

func NewHostHandler(db *gorm.DB, secretKey string) *HostHandler {
	return &HostHandler{db: db, secretKey: secretKey}
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
	for i := range hosts {
		h.sanitizeHostForResponse(&hosts[i])
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
			_ = DecryptCredentialFields(h.secretKey, &cred)
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
		if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
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

	// Prefer PRETTY_NAME, fallback to ID, then uname
	stdout, _, err := client.Execute("cat /etc/os-release 2>/dev/null")
	if err == nil && strings.TrimSpace(stdout) != "" {
		pretty := ""
		id := ""
		for _, line := range strings.Split(stdout, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				pretty = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "ID=") {
				id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
		if pretty != "" {
			h.db.Model(&host).Update("os", pretty)
			return
		}
		if id != "" {
			id = strings.ToUpper(id[:1]) + id[1:]
			h.db.Model(&host).Update("os", id)
			return
		}
	}

	if uname, _, err := client.Execute("uname -srm 2>/dev/null"); err == nil {
		h.db.Model(&host).Update("os", strings.TrimSpace(uname))
	}
}

// TestHost 测试主机连通性并返回诊断信息
func (h *HostHandler) TestHost(c *gin.Context) {
	id := c.Param("id")
	var host Host
	if err := h.db.Preload("Credential").First(&host, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "主机不存在"})
		return
	}
	if host.Credential == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "主机未配置凭据"})
		return
	}
	if err := DecryptCredentialFields(h.secretKey, host.Credential); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据解密失败"})
		return
	}

	client := &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: host.Credential.Username,
		Password: host.Credential.Password,
		Key:      host.Credential.PrivateKey,
		Timeout:  8 * time.Second,
	}

	uname, unameErr, _ := client.Execute("uname -a")
	osrel, osErr, _ := client.Execute("cat /etc/os-release")

	// 若成功拿到系统信息，则回写 OS
	if strings.TrimSpace(osrel) != "" {
		pretty := ""
		idv := ""
		for _, line := range strings.Split(osrel, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				pretty = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "ID=") {
				idv = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
		if pretty != "" {
			h.db.Model(&host).Update("os", pretty)
		} else if idv != "" {
			idv = strings.ToUpper(idv[:1]) + idv[1:]
			h.db.Model(&host).Update("os", idv)
		}
	} else if strings.TrimSpace(uname) != "" {
		h.db.Model(&host).Update("os", strings.TrimSpace(uname))
	}

	result := gin.H{
		"uname":      gin.H{"out": uname, "err": unameErr},
		"os_release": gin.H{"out": osrel, "err": osErr},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// Get 获取主机详情
func (h *HostHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var host Host
	if err := h.db.Preload("Group").Preload("Credential").First(&host, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "主机不存在"})
		return
	}
	if host.Credential != nil {
		if err := DecryptCredentialFields(h.secretKey, host.Credential); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "主机凭据解密失败"})
			return
		}
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
		GroupName string `json:"group_name"`
		Username  string `json:"username"`
		Password  string `json:"password"`
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
	host.Status = req.Status
	if req.GroupID != "" {
		host.GroupID = req.GroupID
	}
	if req.GroupName != "" {
		var group HostGroup
		if err := h.db.FirstOrCreate(&group, HostGroup{Name: req.GroupName}).Error; err == nil {
			host.GroupID = group.ID
		}
	}
	// host.Description = req.Description // 如果需要支持更多字段

	// 更新或创建凭据
	if req.Username != "" {
		if host.CredentialID != "" {
			// 更新现有凭据（空密码不覆盖）
			var cred Credential
			if err := h.db.First(&cred, "id = ?", host.CredentialID).Error; err == nil {
				updates := map[string]interface{}{"username": req.Username}
				if strings.TrimSpace(req.Password) != "" {
					enc, encErr := security.Encrypt(h.secretKey, "cmdb.credential.password", req.Password)
					if encErr != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + encErr.Error()})
						return
					}
					updates["password"] = enc
				}
				_ = h.db.Model(&Credential{}).Where("id = ?", host.CredentialID).Updates(updates).Error
			}
		} else {
			// 创建新凭据
			cred := Credential{
				Name:     host.Name + "-cred",
				Type:     "password",
				Username: req.Username,
				Password: req.Password,
			}
			if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
				return
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
	// 如果凭据更新过，尝试重新识别 OS
	if req.Username != "" {
		go h.detectOSAsync(host.ID, req.Username, req.Password)
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

// UpdateGroup 更新分组
func (h *HostHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var group HostGroup
	if err := h.db.First(&group, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "分组不存在"})
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

// DeleteGroup 删除分组
func (h *HostHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&HostGroup{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListCredentials 凭据列表
func (h *HostHandler) ListCredentials(c *gin.Context) {
	var creds []Credential
	if err := h.db.Find(&creds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range creds {
		SanitizeCredentialFields(&creds[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": creds})
}

// GetCredential 获取凭据详情（用于编辑）
func (h *HostHandler) GetCredential(c *gin.Context) {
	id := c.Param("id")
	var cred Credential
	if err := h.db.First(&cred, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "凭据不存在"})
		return
	}
	if err := DecryptCredentialFields(h.secretKey, &cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据解密失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cred})
}

// CreateCredential 创建凭据
func (h *HostHandler) CreateCredential(c *gin.Context) {
	var cred Credential
	if err := c.ShouldBindJSON(&cred); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
		return
	}
	if err := h.db.Create(&cred).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	SanitizeCredentialFields(&cred)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cred})
}

// UpdateCredential 更新凭据
func (h *HostHandler) UpdateCredential(c *gin.Context) {
	id := c.Param("id")
	var current Credential
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "凭据不存在"})
		return
	}
	var req Credential
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := map[string]interface{}{
		"name":     coalesceString(req.Name, current.Name),
		"type":     coalesceString(req.Type, current.Type),
		"username": coalesceString(req.Username, current.Username),
		"notes":    coalesceString(req.Notes, current.Notes),
	}
	if strings.TrimSpace(req.Password) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.credential.password", req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		updates["password"] = enc
	}
	if strings.TrimSpace(req.PrivateKey) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.credential.private_key", req.PrivateKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		updates["private_key"] = enc
	}
	if strings.TrimSpace(req.Passphrase) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.credential.passphrase", req.Passphrase)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		updates["passphrase"] = enc
	}
	if strings.TrimSpace(req.AccessKey) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.credential.access_key", req.AccessKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		updates["access_key"] = enc
	}
	if strings.TrimSpace(req.SecretKey) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.credential.secret_key", req.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		updates["secret_key"] = enc
	}

	if err := h.db.Model(&Credential{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if err := h.db.First(&current, "id = ?", id).Error; err == nil {
		SanitizeCredentialFields(&current)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": current})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCredential 删除凭据
func (h *HostHandler) DeleteCredential(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Credential{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestCredential 测试凭据连通性
func (h *HostHandler) TestCredential(c *gin.Context) {
	id := c.Param("id")
	var cred Credential
	if err := h.db.First(&cred, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "凭据不存在"})
		return
	}
	if err := DecryptCredentialFields(h.secretKey, &cred); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据解密失败"})
		return
	}

	// API 类型只校验基本字段
	if cred.Type == "api" {
		if cred.AccessKey == "" || cred.SecretKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "AccessKey/SecretKey 不能为空"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "API 凭据格式正常"})
		return
	}

	var req struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请填写主机地址"})
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}

	password := cred.Password
	if password == "" && cred.Passphrase != "" {
		password = cred.Passphrase
	}
	client := &core.SSHClient{
		Host:     req.Host,
		Port:     req.Port,
		Username: cred.Username,
		Password: password,
		Key:      cred.PrivateKey,
		Timeout:  8 * time.Second,
	}
	_, stderr, err := client.Execute("echo ok")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "连接失败: " + stderr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "连接成功"})
}

// ListDatabases 数据库资产列表
func (h *HostHandler) ListDatabases(c *gin.Context) {
	var items []DatabaseAsset
	query := h.db.Order("updated_at DESC")
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR host LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if env := c.Query("environment"); env != "" {
		query = query.Where("environment = ?", env)
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range items {
		items[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// CreateDatabase 创建数据库资产
func (h *HostHandler) CreateDatabase(c *gin.Context) {
	var item DatabaseAsset
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if item.Port == 0 {
		item.Port = 3306
	}
	var err error
	item.Password, err = security.Encrypt(h.secretKey, "cmdb.database.password", item.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库密码加密失败: " + err.Error()})
		return
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	item.Password = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

// GetDatabase 获取数据库资产详情
func (h *HostHandler) GetDatabase(c *gin.Context) {
	id := c.Param("id")
	var item DatabaseAsset
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "数据库资产不存在"})
		return
	}
	if strings.TrimSpace(item.Password) != "" {
		plain, err := security.Decrypt(h.secretKey, "cmdb.database.password", item.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库密码解密失败"})
			return
		}
		item.Password = plain
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

// UpdateDatabase 更新数据库资产
func (h *HostHandler) UpdateDatabase(c *gin.Context) {
	id := c.Param("id")
	var current DatabaseAsset
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "数据库资产不存在"})
		return
	}
	var req DatabaseAsset
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Port == 0 {
		req.Port = 3306
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"type":        req.Type,
		"host":        req.Host,
		"port":        req.Port,
		"username":    req.Username,
		"database":    req.Database,
		"environment": req.Environment,
		"owner":       req.Owner,
		"tags":        req.Tags,
		"status":      req.Status,
		"description": req.Description,
	}
	if strings.TrimSpace(req.Password) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.database.password", req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库密码加密失败: " + err.Error()})
			return
		}
		updates["password"] = enc
	}
	if err := h.db.Model(&DatabaseAsset{}).Where("id = ?", id).Updates(updates).Error; err != nil {
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

// DeleteDatabase 删除数据库资产
func (h *HostHandler) DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&DatabaseAsset{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestDatabase 测试数据库端口连通性
func (h *HostHandler) TestDatabase(c *gin.Context) {
	id := c.Param("id")
	var item DatabaseAsset
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "数据库资产不存在"})
		return
	}
	addr := net.JoinHostPort(item.Host, func() string {
		if item.Port == 0 {
			return "3306"
		}
		return fmt.Sprintf("%d", item.Port)
	}())
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "连接失败: " + err.Error()})
		return
	}
	_ = conn.Close()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "连接成功", "latency_ms": time.Since(start).Milliseconds()})
}

// ListCloudAccounts 云账号列表
func (h *HostHandler) ListCloudAccounts(c *gin.Context) {
	var accounts []CloudAccount
	query := h.db.Order("updated_at DESC")
	if provider := c.Query("provider"); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range accounts {
		accounts[i].AccessKey = ""
		accounts[i].SecretKey = ""
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": accounts})
}

// CreateCloudAccount 创建云账号
func (h *HostHandler) CreateCloudAccount(c *gin.Context) {
	var account CloudAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	var err error
	account.AccessKey, err = security.Encrypt(h.secretKey, "cmdb.cloud.access_key", account.AccessKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥加密失败: " + err.Error()})
		return
	}
	account.SecretKey, err = security.Encrypt(h.secretKey, "cmdb.cloud.secret_key", account.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥加密失败: " + err.Error()})
		return
	}
	if err := h.db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	account.AccessKey = ""
	account.SecretKey = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": account})
}

// GetCloudAccount 获取云账号详情
func (h *HostHandler) GetCloudAccount(c *gin.Context) {
	id := c.Param("id")
	var account CloudAccount
	if err := h.db.First(&account, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "云账号不存在"})
		return
	}
	if strings.TrimSpace(account.AccessKey) != "" {
		plain, err := security.Decrypt(h.secretKey, "cmdb.cloud.access_key", account.AccessKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号 AccessKey 解密失败"})
			return
		}
		account.AccessKey = plain
	}
	if strings.TrimSpace(account.SecretKey) != "" {
		plain, err := security.Decrypt(h.secretKey, "cmdb.cloud.secret_key", account.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号 SecretKey 解密失败"})
			return
		}
		account.SecretKey = plain
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": account})
}

// UpdateCloudAccount 更新云账号
func (h *HostHandler) UpdateCloudAccount(c *gin.Context) {
	id := c.Param("id")
	var current CloudAccount
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "云账号不存在"})
		return
	}
	var req CloudAccount
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"provider":    req.Provider,
		"region":      req.Region,
		"status":      req.Status,
		"description": req.Description,
	}
	if strings.TrimSpace(req.AccessKey) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.cloud.access_key", req.AccessKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥加密失败: " + err.Error()})
			return
		}
		updates["access_key"] = enc
	}
	if strings.TrimSpace(req.SecretKey) != "" {
		enc, err := security.Encrypt(h.secretKey, "cmdb.cloud.secret_key", req.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥加密失败: " + err.Error()})
			return
		}
		updates["secret_key"] = enc
	}
	if err := h.db.Model(&CloudAccount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&current, "id = ?", id).Error; err == nil {
		current.AccessKey = ""
		current.SecretKey = ""
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": current})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCloudAccount 删除云账号
func (h *HostHandler) DeleteCloudAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CloudAccount{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestCloudAccount 测试云账号（仅校验字段）
func (h *HostHandler) TestCloudAccount(c *gin.Context) {
	id := c.Param("id")
	var account CloudAccount
	if err := h.db.First(&account, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "云账号不存在"})
		return
	}
	accessKey, err := security.Decrypt(h.secretKey, "cmdb.cloud.access_key", account.AccessKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥解密失败"})
		return
	}
	secretKey, err := security.Decrypt(h.secretKey, "cmdb.cloud.secret_key", account.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "云账号密钥解密失败"})
		return
	}
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "AccessKey/SecretKey 不能为空"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "账号格式正常（未实际调用云 API）"})
}

// ListCloudResources 云资源列表
func (h *HostHandler) ListCloudResources(c *gin.Context) {
	var resources []CloudResource
	query := h.db.Preload("Account").Order("updated_at DESC")
	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR resource_id LIKE ? OR ip LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := query.Find(&resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resources})
}

// CreateCloudResource 创建云资源
func (h *HostHandler) CreateCloudResource(c *gin.Context) {
	var resource CloudResource
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resource})
}

// GetCloudResource 获取云资源详情
func (h *HostHandler) GetCloudResource(c *gin.Context) {
	id := c.Param("id")
	var resource CloudResource
	if err := h.db.Preload("Account").First(&resource, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "云资源不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resource})
}

// UpdateCloudResource 更新云资源
func (h *HostHandler) UpdateCloudResource(c *gin.Context) {
	id := c.Param("id")
	var resource CloudResource
	if err := h.db.First(&resource, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "云资源不存在"})
		return
	}
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resource})
}

// DeleteCloudResource 删除云资源
func (h *HostHandler) DeleteCloudResource(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CloudResource{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *HostHandler) sanitizeHostForResponse(host *Host) {
	if host == nil {
		return
	}
	if host.Credential != nil {
		SanitizeCredentialFields(host.Credential)
	}
}

func coalesceString(newValue, oldValue string) string {
	if strings.TrimSpace(newValue) == "" {
		return oldValue
	}
	return newValue
}
