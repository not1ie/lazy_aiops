package cmdb

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosnmp/gosnmp"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/security"
	"gorm.io/gorm"
)

type HostHandler struct {
	db        *gorm.DB
	secretKey string
}

type hostStatusSyncSummary struct {
	Total       int   `json:"total"`
	Online      int   `json:"online"`
	Offline     int   `json:"offline"`
	Maintenance int   `json:"maintenance"`
	Changed     int   `json:"changed"`
	Failed      int   `json:"failed"`
	DurationMs  int64 `json:"duration_ms"`
}

func NewHostHandler(db *gorm.DB, secretKey string) *HostHandler {
	return &HostHandler{db: db, secretKey: secretKey}
}

// List 主机列表
func (h *HostHandler) List(c *gin.Context) {
	if queryTruthy(c.Query("live")) {
		_, _ = h.syncHostStatuses(nil, 2*time.Second)
	}

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

// SyncHostStatuses 主机状态批量巡检
func (h *HostHandler) SyncHostStatuses(c *gin.Context) {
	var req struct {
		IDs       []string `json:"ids"`
		TimeoutMs int      `json:"timeout_ms"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.TimeoutMs <= 0 {
		if v := c.Query("timeout_ms"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil {
				req.TimeoutMs = parsed
			}
		}
	}

	timeout := clampDuration(req.TimeoutMs, 2*time.Second)
	summary, err := h.syncHostStatuses(req.IDs, timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "状态巡检失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "巡检完成", "data": summary})
}

func queryTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func clampDuration(raw int, fallback time.Duration) time.Duration {
	if raw <= 0 {
		return fallback
	}
	if raw < 200 {
		raw = 200
	}
	if raw > 10000 {
		raw = 10000
	}
	return time.Duration(raw) * time.Millisecond
}

func truncateReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) <= 240 {
		return reason
	}
	return reason[:240]
}

func (h *HostHandler) probeHostTCP(host Host, timeout time.Duration) (bool, string) {
	ip := strings.TrimSpace(host.IP)
	if ip == "" {
		return false, "IP 为空"
	}
	port := host.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, truncateReason(err.Error())
	}
	_ = conn.Close()
	return true, ""
}

func (h *HostHandler) syncHostStatuses(ids []string, timeout time.Duration) (hostStatusSyncSummary, error) {
	startedAt := time.Now()
	summary := hostStatusSyncSummary{}

	query := h.db.Model(&Host{})
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	var hosts []Host
	if err := query.Find(&hosts).Error; err != nil {
		return summary, err
	}
	summary.Total = len(hosts)
	if len(hosts) == 0 {
		return summary, nil
	}

	workerCount := 12
	if len(hosts) < workerCount {
		workerCount = len(hosts)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan Host)
	var wg sync.WaitGroup
	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for host := range jobs {
			now := time.Now()
			online, reason := h.probeHostTCP(host, timeout)
			updates := map[string]interface{}{
				"last_check_at": &now,
			}

			nextStatus := host.Status
			if host.Status != 2 {
				if online {
					nextStatus = 1
				} else {
					nextStatus = 0
				}
				updates["status"] = nextStatus
			}
			if online {
				updates["last_online_at"] = &now
				updates["status_reason"] = ""
			} else {
				updates["status_reason"] = reason
			}

			err := h.db.Model(&Host{}).Where("id = ?", host.ID).Updates(updates).Error

			mu.Lock()
			if err != nil {
				summary.Failed++
				mu.Unlock()
				continue
			}
			if nextStatus != host.Status {
				summary.Changed++
			}
			switch nextStatus {
			case 1:
				summary.Online++
			case 2:
				summary.Maintenance++
			default:
				summary.Offline++
			}
			mu.Unlock()
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	for _, host := range hosts {
		jobs <- host
	}
	close(jobs)
	wg.Wait()

	summary.DurationMs = time.Since(startedAt).Milliseconds()
	return summary, nil
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
	topCPUOut, topCPUErr, _ := client.Execute("ps -eo pid,comm,%cpu,%mem --sort=-%cpu | sed -n '1,8p'")
	topMemOut, topMemErr, _ := client.Execute("ps -eo pid,comm,%cpu,%mem --sort=-%mem | sed -n '1,8p'")
	tcpOut, tcpErr, _ := client.Execute(`sh -c 'if command -v ss >/dev/null 2>&1; then ss -tunap | sed -n "1,120p"; elif command -v netstat >/dev/null 2>&1; then netstat -tunap 2>/dev/null | sed -n "1,120p"; else echo "ss/netstat not found"; fi'`)

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
		"processes": gin.H{
			"top_cpu": parseProcessRows(topCPUOut),
			"top_mem": parseProcessRows(topMemOut),
			"errors": gin.H{
				"top_cpu": strings.TrimSpace(topCPUErr),
				"top_mem": strings.TrimSpace(topMemErr),
			},
		},
	}
	tcpRows, tcpSummary := parseTCPRows(tcpOut)
	result["tcp_connections"] = tcpRows
	result["tcp_summary"] = tcpSummary
	result["tcp_probe"] = gin.H{
		"error": strings.TrimSpace(tcpErr),
	}

	now := time.Now()
	isOnline := strings.TrimSpace(uname) != "" || strings.TrimSpace(osrel) != ""
	reason := strings.TrimSpace(unameErr)
	if reason == "" {
		reason = strings.TrimSpace(osErr)
	}
	if reason == "" && !isOnline {
		reason = "SSH 探测失败"
	}
	updates := map[string]interface{}{
		"last_check_at": &now,
	}
	if host.Status != 2 {
		if isOnline {
			updates["status"] = 1
		} else {
			updates["status"] = 0
		}
	}
	if isOnline {
		updates["last_online_at"] = &now
		updates["status_reason"] = ""
	} else {
		updates["status_reason"] = truncateReason(reason)
	}
	_ = h.db.Model(&Host{}).Where("id = ?", host.ID).Updates(updates).Error

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func parseProcessRows(raw string) []gin.H {
	lines := strings.Split(raw, "\n")
	rows := make([]gin.H, 0, 8)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "pid") || strings.Contains(lower, "%cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		rows = append(rows, gin.H{
			"pid":     fields[0],
			"command": fields[1],
			"cpu":     fields[2],
			"memory":  fields[3],
		})
		if len(rows) >= 6 {
			break
		}
	}
	return rows
}

func parseTCPRows(raw string) ([]gin.H, gin.H) {
	lines := strings.Split(raw, "\n")
	rows := make([]gin.H, 0, 80)
	stateCounter := map[string]int{}
	formatSS := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "netid ") {
			formatSS = true
			continue
		}
		if strings.HasPrefix(lower, "proto ") || strings.HasPrefix(lower, "active internet") {
			formatSS = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		proto := fields[0]
		state := ""
		localAddr := ""
		remoteAddr := ""
		process := ""

		if formatSS {
			if len(fields) < 6 {
				continue
			}
			state = fields[1]
			localAddr = fields[4]
			remoteAddr = fields[5]
			if len(fields) > 6 {
				process = strings.Join(fields[6:], " ")
			}
		} else {
			if len(fields) < 6 {
				continue
			}
			localAddr = fields[3]
			remoteAddr = fields[4]
			state = fields[5]
			if len(fields) > 6 {
				process = strings.Join(fields[6:], " ")
			}
		}

		if !strings.HasPrefix(strings.ToLower(proto), "tcp") {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(state))
		stateCounter[normalized]++
		rows = append(rows, gin.H{
			"proto":   proto,
			"state":   state,
			"local":   localAddr,
			"remote":  remoteAddr,
			"process": process,
		})
		if len(rows) >= 80 {
			break
		}
	}

	summary := gin.H{
		"total":       len(rows),
		"established": stateCounter["established"] + stateCounter["estab"],
		"listen":      stateCounter["listen"],
		"time_wait":   stateCounter["time-wait"] + stateCounter["time_wait"],
	}
	return rows, summary
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
		Name        *string `json:"name"`
		IP          *string `json:"ip"`
		Port        *int    `json:"port"`
		OS          *string `json:"os"`
		Status      *int    `json:"status"`
		GroupID     *string `json:"group_id"`
		GroupName   *string `json:"group_name"`
		CPU         *string `json:"cpu"`
		Memory      *string `json:"memory"`
		Disk        *string `json:"disk"`
		Tags        *string `json:"tags"`
		Description *string `json:"description"`
		Username    *string `json:"username"`
		Password    *string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IP != nil {
		updates["ip"] = *req.IP
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.OS != nil {
		updates["os"] = *req.OS
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.GroupID != nil {
		updates["group_id"] = *req.GroupID
	}
	if req.CPU != nil {
		updates["cpu"] = *req.CPU
	}
	if req.Memory != nil {
		updates["memory"] = *req.Memory
	}
	if req.Disk != nil {
		updates["disk"] = *req.Disk
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.GroupName != nil && strings.TrimSpace(*req.GroupName) != "" {
		var group HostGroup
		if err := h.db.FirstOrCreate(&group, HostGroup{Name: *req.GroupName}).Error; err == nil {
			updates["group_id"] = group.ID
		}
	}

	// 更新或创建凭据
	username := ""
	password := ""
	if req.Username != nil {
		username = strings.TrimSpace(*req.Username)
	}
	if req.Password != nil {
		password = *req.Password
	}
	if username != "" {
		if host.CredentialID != "" {
			// 更新现有凭据（空密码不覆盖）
			var cred Credential
			if err := h.db.First(&cred, "id = ?", host.CredentialID).Error; err == nil {
				credUpdates := map[string]interface{}{"username": username}
				if strings.TrimSpace(password) != "" {
					enc, encErr := security.Encrypt(h.secretKey, "cmdb.credential.password", password)
					if encErr != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + encErr.Error()})
						return
					}
					credUpdates["password"] = enc
				}
				_ = h.db.Model(&Credential{}).Where("id = ?", host.CredentialID).Updates(credUpdates).Error
			}
		} else {
			// 创建新凭据
			credName := host.Name
			if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
				credName = strings.TrimSpace(*req.Name)
			}
			cred := Credential{
				Name:     credName + "-cred",
				Type:     "password",
				Username: username,
				Password: password,
			}
			if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
				return
			}
			if err := h.db.Create(&cred).Error; err == nil {
				updates["credential_id"] = cred.ID
			}
		}
	}

	if len(updates) > 0 {
		if err := h.db.Model(&host).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}
	if err := h.db.First(&host, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// 如果凭据更新过，尝试重新识别 OS
	if username != "" {
		go h.detectOSAsync(host.ID, username, password)
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

type firewallDeviceSnapshot struct {
	Name          string `json:"name"`
	Vendor        string `json:"vendor"`
	Model         string `json:"model"`
	IP            string `json:"ip"`
	ManagePort    int    `json:"manage_port"`
	SNMPVersion   string `json:"snmp_version"`
	SNMPCommunity string `json:"snmp_community"`
	SNMPPort      int    `json:"snmp_port"`
	SNMPUser      string `json:"snmp_user"`
	SNMPAuthProto string `json:"snmp_auth_proto"`
	SNMPPrivProto string `json:"snmp_priv_proto"`
	Status        int    `json:"status"`
	Description   string `json:"description"`
}

func (h *HostHandler) buildNetworkDeviceListQuery(c *gin.Context) *gorm.DB {
	query := h.db.Preload("Credential").Order("updated_at DESC")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR vendor LIKE ? OR model LIKE ? OR serial_number LIKE ?", like, like, like, like, like)
	}
	if deviceType := strings.TrimSpace(c.Query("device_type")); deviceType != "" {
		query = query.Where("device_type = ?", strings.ToLower(deviceType))
	}
	if statusText := strings.TrimSpace(c.Query("status")); statusText != "" {
		if status, err := strconv.Atoi(statusText); err == nil {
			query = query.Where("status = ?", status)
		}
	}
	return query
}

func probeNetworkDeviceTCP(device NetworkDevice, timeout time.Duration) (bool, string) {
	ip := strings.TrimSpace(device.IP)
	if ip == "" {
		return false, "IP 为空"
	}
	port := device.ManagePort
	if port <= 0 {
		if strings.EqualFold(strings.TrimSpace(device.DeviceType), "firewall") {
			port = 443
		} else {
			port = 22
		}
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, truncateReason(err.Error())
	}
	_ = conn.Close()
	return true, ""
}

func (h *HostHandler) syncNetworkDeviceStatuses(devices []NetworkDevice, timeout time.Duration) {
	if len(devices) == 0 {
		return
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	workerCount := 10
	if len(devices) < workerCount {
		workerCount = len(devices)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	jobs := make(chan NetworkDevice)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for device := range jobs {
			now := time.Now()
			online, reason := probeNetworkDeviceTCP(device, timeout)
			nextStatus := device.Status
			if online {
				if nextStatus == 0 {
					nextStatus = 1
				}
			} else {
				nextStatus = 0
			}

			updates := map[string]interface{}{
				"status":        nextStatus,
				"last_check_at": &now,
			}
			if online {
				updates["last_online_at"] = &now
				if nextStatus != 2 {
					updates["status_reason"] = ""
				}
			} else {
				updates["status_reason"] = truncateReason(reason)
			}
			_ = h.db.Model(&NetworkDevice{}).Where("id = ?", device.ID).Updates(updates).Error
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	for i := range devices {
		jobs <- devices[i]
	}
	close(jobs)
	wg.Wait()
}

// ListNetworkDevices 网络设备列表（交换机/防火墙）
func (h *HostHandler) ListNetworkDevices(c *gin.Context) {
	var devices []NetworkDevice
	if err := h.buildNetworkDeviceListQuery(c).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if queryTruthy(c.Query("live")) {
		timeoutMs := 1500
		if raw := strings.TrimSpace(c.Query("timeout_ms")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				timeoutMs = parsed
			}
		}
		h.syncNetworkDeviceStatuses(devices, clampDuration(timeoutMs, 2*time.Second))
		devices = nil
		if err := h.buildNetworkDeviceListQuery(c).Find(&devices).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}
	for i := range devices {
		h.sanitizeNetworkDeviceForResponse(&devices[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": devices})
}

// CreateNetworkDevice 创建网络设备
func (h *HostHandler) CreateNetworkDevice(c *gin.Context) {
	var req struct {
		NetworkDevice
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	device := req.NetworkDevice
	device.DeviceType = strings.ToLower(strings.TrimSpace(device.DeviceType))
	if device.DeviceType == "" {
		device.DeviceType = "switch"
	}
	if device.DeviceType != "switch" && device.DeviceType != "firewall" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device_type 仅支持 switch/firewall"})
		return
	}
	device.IP = strings.TrimSpace(device.IP)
	if device.IP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "IP 不能为空"})
		return
	}
	if device.ManagePort == 0 {
		if device.DeviceType == "firewall" {
			device.ManagePort = 443
		} else {
			device.ManagePort = 22
		}
	}
	if device.SNMPPort == 0 {
		device.SNMPPort = 161
	}
	if device.SNMPVersion == "" {
		device.SNMPVersion = "v2c"
	}

	var exists NetworkDevice
	if err := h.db.Where("device_type = ? AND ip = ?", device.DeviceType, device.IP).First(&exists).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "该类型设备 IP 已存在"})
		return
	} else if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if username := strings.TrimSpace(req.Username); username != "" {
		cred := Credential{
			Name:     strings.TrimSpace(device.Name) + "-network-cred",
			Type:     "password",
			Username: username,
			Password: req.Password,
		}
		if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
			return
		}
		if err := h.db.Create(&cred).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		device.CredentialID = cred.ID
	}

	if err := h.db.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.sanitizeNetworkDeviceForResponse(&device)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
}

// GetNetworkDevice 获取网络设备详情（用于编辑）
func (h *HostHandler) GetNetworkDevice(c *gin.Context) {
	id := c.Param("id")
	var device NetworkDevice
	if err := h.db.Preload("Credential").First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "网络设备不存在"})
		return
	}
	if device.Credential != nil {
		if err := DecryptCredentialFields(h.secretKey, device.Credential); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "设备凭据解密失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
}

// UpdateNetworkDevice 更新网络设备
func (h *HostHandler) UpdateNetworkDevice(c *gin.Context) {
	id := c.Param("id")
	var device NetworkDevice
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "网络设备不存在"})
		return
	}

	var req struct {
		Name            *string `json:"name"`
		DeviceType      *string `json:"device_type"`
		Vendor          *string `json:"vendor"`
		Model           *string `json:"model"`
		IP              *string `json:"ip"`
		ManagePort      *int    `json:"manage_port"`
		SNMPVersion     *string `json:"snmp_version"`
		SNMPCommunity   *string `json:"snmp_community"`
		SNMPPort        *int    `json:"snmp_port"`
		SNMPUser        *string `json:"snmp_user"`
		SNMPAuthProto   *string `json:"snmp_auth_proto"`
		SNMPAuthPass    *string `json:"snmp_auth_pass"`
		SNMPPrivProto   *string `json:"snmp_priv_proto"`
		SNMPPrivPass    *string `json:"snmp_priv_pass"`
		Location        *string `json:"location"`
		Rack            *string `json:"rack"`
		SerialNumber    *string `json:"serial_number"`
		FirmwareVersion *string `json:"firmware_version"`
		Status          *int    `json:"status"`
		Tags            *string `json:"tags"`
		Description     *string `json:"description"`
		Username        *string `json:"username"`
		Password        *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.DeviceType != nil {
		deviceType := strings.ToLower(strings.TrimSpace(*req.DeviceType))
		if deviceType != "switch" && deviceType != "firewall" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "device_type 仅支持 switch/firewall"})
			return
		}
		updates["device_type"] = deviceType
	}
	if req.Vendor != nil {
		updates["vendor"] = strings.TrimSpace(*req.Vendor)
	}
	if req.Model != nil {
		updates["model"] = strings.TrimSpace(*req.Model)
	}
	if req.IP != nil {
		ip := strings.TrimSpace(*req.IP)
		if ip == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "IP 不能为空"})
			return
		}
		updates["ip"] = ip
	}
	if req.ManagePort != nil {
		updates["manage_port"] = *req.ManagePort
	}
	if req.SNMPVersion != nil {
		updates["snmp_version"] = strings.TrimSpace(*req.SNMPVersion)
	}
	if req.SNMPCommunity != nil {
		updates["snmp_community"] = strings.TrimSpace(*req.SNMPCommunity)
	}
	if req.SNMPPort != nil {
		updates["snmp_port"] = *req.SNMPPort
	}
	if req.SNMPUser != nil {
		updates["snmp_user"] = strings.TrimSpace(*req.SNMPUser)
	}
	if req.SNMPAuthProto != nil {
		updates["snmp_auth_proto"] = strings.TrimSpace(*req.SNMPAuthProto)
	}
	if req.SNMPAuthPass != nil {
		updates["snmp_auth_pass"] = *req.SNMPAuthPass
	}
	if req.SNMPPrivProto != nil {
		updates["snmp_priv_proto"] = strings.TrimSpace(*req.SNMPPrivProto)
	}
	if req.SNMPPrivPass != nil {
		updates["snmp_priv_pass"] = *req.SNMPPrivPass
	}
	if req.Location != nil {
		updates["location"] = strings.TrimSpace(*req.Location)
	}
	if req.Rack != nil {
		updates["rack"] = strings.TrimSpace(*req.Rack)
	}
	if req.SerialNumber != nil {
		updates["serial_number"] = strings.TrimSpace(*req.SerialNumber)
	}
	if req.FirmwareVersion != nil {
		updates["firmware_version"] = strings.TrimSpace(*req.FirmwareVersion)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Tags != nil {
		updates["tags"] = strings.TrimSpace(*req.Tags)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}

	if len(updates) > 0 {
		if err := h.db.Model(&device).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	username := ""
	password := ""
	if req.Username != nil {
		username = strings.TrimSpace(*req.Username)
	}
	if req.Password != nil {
		password = *req.Password
	}
	if username != "" {
		if device.CredentialID != "" {
			var cred Credential
			if err := h.db.First(&cred, "id = ?", device.CredentialID).Error; err == nil {
				credUpdates := map[string]interface{}{"username": username}
				if strings.TrimSpace(password) != "" {
					enc, err := security.Encrypt(h.secretKey, "cmdb.credential.password", password)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
						return
					}
					credUpdates["password"] = enc
				}
				if err := h.db.Model(&Credential{}).Where("id = ?", device.CredentialID).Updates(credUpdates).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
					return
				}
			}
		} else {
			credName := strings.TrimSpace(device.Name)
			if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
				credName = strings.TrimSpace(*req.Name)
			}
			cred := Credential{
				Name:     credName + "-network-cred",
				Type:     "password",
				Username: username,
				Password: password,
			}
			if err := EncryptCredentialFields(h.secretKey, &cred); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "凭据加密失败: " + err.Error()})
				return
			}
			if err := h.db.Create(&cred).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
			if err := h.db.Model(&device).Update("credential_id", cred.ID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
		}
	}

	var latest NetworkDevice
	if err := h.db.Preload("Credential").First(&latest, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.sanitizeNetworkDeviceForResponse(&latest)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": latest})
}

// DeleteNetworkDevice 删除网络设备
func (h *HostHandler) DeleteNetworkDevice(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&NetworkDevice{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestNetworkDevice 测试网络设备连通性（管理口/SSH/SNMP）
func (h *HostHandler) TestNetworkDevice(c *gin.Context) {
	id := c.Param("id")
	var device NetworkDevice
	if err := h.db.Preload("Credential").First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "网络设备不存在"})
		return
	}
	if device.Credential != nil {
		if err := DecryptCredentialFields(h.secretKey, device.Credential); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "设备凭据解密失败"})
			return
		}
	}

	result := gin.H{}
	tcpOK := false
	sshOK := false
	snmpOK := false
	reasons := make([]string, 0, 3)

	managePort := device.ManagePort
	if managePort == 0 {
		managePort = 22
	}
	addr := net.JoinHostPort(device.IP, fmt.Sprintf("%d", managePort))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		result["tcp"] = gin.H{"ok": false, "error": err.Error()}
		reasons = append(reasons, "TCP:"+err.Error())
	} else {
		_ = conn.Close()
		tcpOK = true
		result["tcp"] = gin.H{"ok": true, "latency_ms": time.Since(start).Milliseconds()}
	}

	if device.Credential != nil && strings.TrimSpace(device.Credential.Username) != "" {
		client := &core.SSHClient{
			Host:     device.IP,
			Port:     managePort,
			Username: device.Credential.Username,
			Password: device.Credential.Password,
			Key:      device.Credential.PrivateKey,
			Timeout:  8 * time.Second,
		}
		stdout, stderr, err := client.Execute("echo ok")
		if err != nil {
			reason := strings.TrimSpace(stderr)
			if reason == "" {
				reason = err.Error()
			}
			reasons = append(reasons, "SSH:"+reason)
			result["ssh"] = gin.H{"ok": false, "error": strings.TrimSpace(stderr)}
		} else {
			sshOK = true
			result["ssh"] = gin.H{"ok": true, "out": strings.TrimSpace(stdout)}
		}
	} else {
		result["ssh"] = gin.H{"ok": false, "message": "未配置 SSH 凭据"}
	}

	if strings.TrimSpace(device.SNMPCommunity) != "" || strings.EqualFold(strings.TrimSpace(device.SNMPVersion), "v3") {
		snmp, err := h.createSNMPClientForNetworkDevice(&device)
		if err != nil {
			reasons = append(reasons, "SNMP:"+err.Error())
			result["snmp"] = gin.H{"ok": false, "error": err.Error()}
		} else {
			defer snmp.Conn.Close()
			pdu, err := snmp.Get([]string{"1.3.6.1.2.1.1.1.0"})
			if err != nil {
				reasons = append(reasons, "SNMP:"+err.Error())
				result["snmp"] = gin.H{"ok": false, "error": err.Error()}
			} else {
				sysDesc := ""
				if len(pdu.Variables) > 0 && pdu.Variables[0].Type == gosnmp.OctetString {
					if v, ok := pdu.Variables[0].Value.([]byte); ok {
						sysDesc = string(v)
					}
				}
				snmpOK = true
				result["snmp"] = gin.H{"ok": true, "sys_desc": sysDesc}
			}
		}
	} else {
		result["snmp"] = gin.H{"ok": false, "message": "未配置 SNMP 参数"}
	}

	status := 0
	if tcpOK && (sshOK || device.Credential == nil) && (snmpOK || strings.TrimSpace(device.SNMPCommunity) == "") {
		status = 1
	} else if tcpOK || sshOK || snmpOK {
		status = 2
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":        status,
		"last_check_at": &now,
	}
	if status == 1 {
		updates["last_online_at"] = &now
		updates["status_reason"] = ""
	} else {
		updates["status_reason"] = truncateReason(strings.Join(reasons, "; "))
	}
	_ = h.db.Model(&device).Updates(updates).Error

	result["status"] = status
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// SyncNetworkDevicesFromFirewalls 从 firewall 模块同步防火墙资产到网络设备 CMDB
func (h *HostHandler) SyncNetworkDevicesFromFirewalls(c *gin.Context) {
	var source []firewallDeviceSnapshot
	if err := h.db.Table("firewalls").Select(
		"name", "vendor", "model", "ip", "manage_port", "snmp_version", "snmp_community",
		"snmp_port", "snmp_user", "snmp_auth_proto", "snmp_priv_proto", "status", "description",
	).Find(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取 firewall 设备失败: " + err.Error()})
		return
	}

	created := 0
	updated := 0
	skipped := 0
	for _, item := range source {
		ip := strings.TrimSpace(item.IP)
		if ip == "" {
			skipped++
			continue
		}
		managePort := item.ManagePort
		if managePort == 0 {
			managePort = 443
		}
		snmpPort := item.SNMPPort
		if snmpPort == 0 {
			snmpPort = 161
		}
		payload := map[string]interface{}{
			"name":            strings.TrimSpace(item.Name),
			"device_type":     "firewall",
			"vendor":          strings.TrimSpace(item.Vendor),
			"model":           strings.TrimSpace(item.Model),
			"ip":              ip,
			"manage_port":     managePort,
			"snmp_version":    strings.TrimSpace(item.SNMPVersion),
			"snmp_community":  strings.TrimSpace(item.SNMPCommunity),
			"snmp_port":       snmpPort,
			"snmp_user":       strings.TrimSpace(item.SNMPUser),
			"snmp_auth_proto": strings.TrimSpace(item.SNMPAuthProto),
			"snmp_priv_proto": strings.TrimSpace(item.SNMPPrivProto),
			"status":          item.Status,
			"description":     strings.TrimSpace(item.Description),
		}

		var existing NetworkDevice
		err := h.db.Where("device_type = ? AND ip = ?", "firewall", ip).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			device := NetworkDevice{
				Name:          payload["name"].(string),
				DeviceType:    "firewall",
				Vendor:        payload["vendor"].(string),
				Model:         payload["model"].(string),
				IP:            ip,
				ManagePort:    managePort,
				SNMPVersion:   payload["snmp_version"].(string),
				SNMPCommunity: payload["snmp_community"].(string),
				SNMPPort:      snmpPort,
				SNMPUser:      payload["snmp_user"].(string),
				SNMPAuthProto: payload["snmp_auth_proto"].(string),
				SNMPPrivProto: payload["snmp_priv_proto"].(string),
				Status:        item.Status,
				Description:   payload["description"].(string),
			}
			if err := h.db.Create(&device).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
			created++
			continue
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if err := h.db.Model(&existing).Updates(payload).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":   len(source),
			"created": created,
			"updated": updated,
			"skipped": skipped,
		},
		"message": "同步完成",
	})
}

func (h *HostHandler) createSNMPClientForNetworkDevice(device *NetworkDevice) (*gosnmp.GoSNMP, error) {
	if device == nil {
		return nil, fmt.Errorf("设备不能为空")
	}
	port := device.SNMPPort
	if port == 0 {
		port = 161
	}

	client := &gosnmp.GoSNMP{
		Target:  strings.TrimSpace(device.IP),
		Port:    uint16(port),
		Timeout: 5 * time.Second,
		Retries: 1,
	}
	version := strings.ToLower(strings.TrimSpace(device.SNMPVersion))
	switch version {
	case "", "v2", "v2c":
		client.Version = gosnmp.Version2c
		community := strings.TrimSpace(device.SNMPCommunity)
		if community == "" {
			community = "public"
		}
		client.Community = community
	case "v1":
		client.Version = gosnmp.Version1
		community := strings.TrimSpace(device.SNMPCommunity)
		if community == "" {
			community = "public"
		}
		client.Community = community
	case "v3":
		if strings.TrimSpace(device.SNMPUser) == "" {
			return nil, fmt.Errorf("SNMPv3 用户不能为空")
		}
		authProto := gosnmp.MD5
		if strings.EqualFold(strings.TrimSpace(device.SNMPAuthProto), "sha") {
			authProto = gosnmp.SHA
		}
		privProto := gosnmp.DES
		if strings.EqualFold(strings.TrimSpace(device.SNMPPrivProto), "aes") {
			privProto = gosnmp.AES
		}
		client.Version = gosnmp.Version3
		client.SecurityModel = gosnmp.UserSecurityModel
		client.MsgFlags = gosnmp.AuthPriv
		client.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 strings.TrimSpace(device.SNMPUser),
			AuthenticationProtocol:   authProto,
			AuthenticationPassphrase: device.SNMPAuthPass,
			PrivacyProtocol:          privProto,
			PrivacyPassphrase:        device.SNMPPrivPass,
		}
	default:
		return nil, fmt.Errorf("不支持的 SNMP 版本: %s", device.SNMPVersion)
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return client, nil
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
	var req HostGroup
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"parent_id":   req.ParentID,
	}
	if err := h.db.Model(&group).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&group, "id = ?", id)
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
	var req CloudResource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"account_id":  req.AccountID,
		"resource_id": req.ResourceID,
		"name":        req.Name,
		"type":        req.Type,
		"region":      req.Region,
		"zone":        req.Zone,
		"ip":          req.IP,
		"status":      req.Status,
		"spec":        req.Spec,
		"tags":        req.Tags,
	}
	if err := h.db.Model(&resource).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&resource, "id = ?", id)
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

func (h *HostHandler) sanitizeNetworkDeviceForResponse(device *NetworkDevice) {
	if device == nil {
		return
	}
	device.SNMPAuthPass = ""
	device.SNMPPrivPass = ""
	if device.Credential != nil {
		SanitizeCredentialFields(device.Credential)
	}
}

func coalesceString(newValue, oldValue string) string {
	if strings.TrimSpace(newValue) == "" {
		return oldValue
	}
	return newValue
}
