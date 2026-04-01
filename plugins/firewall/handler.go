package firewall

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosnmp/gosnmp"
	"gorm.io/gorm"
)

type FirewallHandler struct {
	db *gorm.DB
}

type firewallStatusSyncSummary struct {
	Total      int   `json:"total"`
	Healthy    int   `json:"healthy"`
	Unhealthy  int   `json:"unhealthy"`
	Failed     int   `json:"failed"`
	DurationMs int64 `json:"duration_ms"`
}

func NewFirewallHandler(db *gorm.DB) *FirewallHandler {
	return &FirewallHandler{db: db}
}

func firewallQueryTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func truncateFirewallStatusReason(raw string) string {
	if len(raw) <= 240 {
		return raw
	}
	return raw[:240]
}

func closeSNMPConn(snmp *gosnmp.GoSNMP) {
	if snmp == nil || snmp.Conn == nil {
		return
	}
	_ = snmp.Conn.Close()
}

func snmpPDUToFloat(v gosnmp.SnmpPDU) (float64, bool) {
	switch val := v.Value.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

func (h *FirewallHandler) syncAllDeviceStatus() (firewallStatusSyncSummary, error) {
	started := time.Now()
	summary := firewallStatusSyncSummary{}

	var devices []Firewall
	if err := h.db.Select("id").Find(&devices).Error; err != nil {
		return summary, err
	}
	summary.Total = len(devices)

	for i := range devices {
		if _, err := h.syncDeviceStatusByID(devices[i].ID); err != nil {
			summary.Unhealthy++
			summary.Failed++
			continue
		}
		summary.Healthy++
	}
	summary.DurationMs = time.Since(started).Milliseconds()
	return summary, nil
}

func (h *FirewallHandler) syncDeviceStatusByID(id string) ([]SNMPMetric, error) {
	var device Firewall
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return h.syncDeviceStatus(&device)
}

func (h *FirewallHandler) syncDeviceStatus(device *Firewall) ([]SNMPMetric, error) {
	if device == nil {
		return nil, errors.New("设备不存在")
	}
	now := time.Now()
	markOffline := func(err error) error {
		updates := map[string]interface{}{
			"status":        0,
			"last_check_at": &now,
			"status_reason": truncateFirewallStatusReason(err.Error()),
		}
		_ = h.db.Model(&Firewall{}).Where("id = ?", device.ID).Updates(updates).Error
		return err
	}

	snmp, err := h.createSNMPClient(device)
	if err != nil {
		return nil, markOffline(err)
	}
	defer closeSNMPConn(snmp)

	metrics, runtimeUpdates, err := h.collectSNMPMetrics(device, snmp, now)
	if err != nil {
		return nil, markOffline(err)
	}

	updates := map[string]interface{}{
		"status":         1,
		"last_check_at":  &now,
		"last_online_at": &now,
		"status_reason":  "",
	}
	for key, value := range runtimeUpdates {
		updates[key] = value
	}
	if err := h.db.Model(&Firewall{}).Where("id = ?", device.ID).Updates(updates).Error; err != nil {
		return nil, err
	}
	for i := range metrics {
		if err := h.db.Create(&metrics[i]).Error; err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

func (h *FirewallHandler) collectSNMPMetrics(device *Firewall, snmp *gosnmp.GoSNMP, now time.Time) ([]SNMPMetric, map[string]interface{}, error) {
	if device == nil {
		return nil, nil, errors.New("设备不存在")
	}
	metrics := make([]SNMPMetric, 0)
	runtime := make(map[string]interface{})

	oids := map[string]string{
		"sysUpTime": "1.3.6.1.2.1.1.3.0",
		"sysName":   "1.3.6.1.2.1.1.5.0",
		"ifNumber":  "1.3.6.1.2.1.2.1.0",
	}

	switch device.Vendor {
	case "fortinet":
		oids["cpuUsage"] = "1.3.6.1.4.1.12356.101.4.1.3.0"
		oids["memUsage"] = "1.3.6.1.4.1.12356.101.4.1.4.0"
		oids["sessionCount"] = "1.3.6.1.4.1.12356.101.4.1.8.0"
	case "huawei":
		oids["cpuUsage"] = "1.3.6.1.4.1.2011.6.3.4.1.2.0"
		oids["memUsage"] = "1.3.6.1.4.1.2011.6.3.5.1.1.2.0"
	case "cisco":
		oids["cpuUsage"] = "1.3.6.1.4.1.9.9.109.1.1.1.1.3.1"
		oids["memUsage"] = "1.3.6.1.4.1.9.9.48.1.1.1.5.1"
	}

	oidList := make([]string, 0, len(oids))
	for _, oid := range oids {
		oidList = append(oidList, oid)
	}

	result, err := snmp.Get(oidList)
	if err != nil {
		return nil, nil, fmt.Errorf("SNMP采集失败: %w", err)
	}

	for _, variable := range result.Variables {
		value, ok := snmpPDUToFloat(variable)
		if !ok {
			continue
		}
		metricType := "unknown"
		metricName := variable.Name
		unit := ""

		for name, oid := range oids {
			if variable.Name == "."+oid || variable.Name == oid {
				metricName = name
				switch name {
				case "cpuUsage":
					metricType = "cpu"
					unit = "%"
					runtime["cpu_usage"] = value
				case "memUsage":
					metricType = "memory"
					unit = "%"
					runtime["memory_usage"] = value
				case "sessionCount":
					metricType = "session"
					runtime["session_count"] = int64(value)
				case "sysUpTime":
					metricType = "uptime"
					unit = "ticks"
				}
				break
			}
		}

		metrics = append(metrics, SNMPMetric{
			FirewallID:  device.ID,
			MetricType:  metricType,
			MetricName:  metricName,
			Value:       value,
			Unit:        unit,
			CollectedAt: now,
		})
	}

	return metrics, runtime, nil
}

// ListDevices 设备列表
func (h *FirewallHandler) ListDevices(c *gin.Context) {
	if firewallQueryTruthy(c.Query("live")) {
		_, _ = h.syncAllDeviceStatus()
	}

	var devices []Firewall
	if err := h.db.Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": devices})
}

// CreateDevice 创建设备
func (h *FirewallHandler) CreateDevice(c *gin.Context) {
	var device Firewall
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
}

// GetDevice 获取设备详情
func (h *FirewallHandler) GetDevice(c *gin.Context) {
	id := c.Param("id")
	var device Firewall
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "设备不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
}

// UpdateDevice 更新设备
func (h *FirewallHandler) UpdateDevice(c *gin.Context) {
	id := c.Param("id")
	var device Firewall
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "设备不存在"})
		return
	}
	var req struct {
		Name          *string `json:"name"`
		Vendor        *string `json:"vendor"`
		Model         *string `json:"model"`
		IP            *string `json:"ip"`
		ManagePort    *int    `json:"manage_port"`
		SNMPVersion   *string `json:"snmp_version"`
		SNMPCommunity *string `json:"snmp_community"`
		SNMPPort      *int    `json:"snmp_port"`
		SNMPUser      *string `json:"snmp_user"`
		SNMPAuthProto *string `json:"snmp_auth_proto"`
		SNMPAuthPass  *string `json:"snmp_auth_pass"`
		SNMPPrivProto *string `json:"snmp_priv_proto"`
		SNMPPrivPass  *string `json:"snmp_priv_pass"`
		Status        *int    `json:"status"`
		Description   *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Vendor != nil {
		updates["vendor"] = *req.Vendor
	}
	if req.Model != nil {
		updates["model"] = *req.Model
	}
	if req.IP != nil {
		updates["ip"] = *req.IP
	}
	if req.ManagePort != nil {
		updates["manage_port"] = *req.ManagePort
	}
	if req.SNMPVersion != nil {
		updates["snmp_version"] = *req.SNMPVersion
	}
	if req.SNMPCommunity != nil {
		updates["snmp_community"] = *req.SNMPCommunity
	}
	if req.SNMPPort != nil {
		updates["snmp_port"] = *req.SNMPPort
	}
	if req.SNMPUser != nil {
		updates["snmp_user"] = *req.SNMPUser
	}
	if req.SNMPAuthProto != nil {
		updates["snmp_auth_proto"] = *req.SNMPAuthProto
	}
	if req.SNMPAuthPass != nil {
		updates["snmp_auth_pass"] = *req.SNMPAuthPass
	}
	if req.SNMPPrivProto != nil {
		updates["snmp_priv_proto"] = *req.SNMPPrivProto
	}
	if req.SNMPPrivPass != nil {
		updates["snmp_priv_pass"] = *req.SNMPPrivPass
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
		return
	}
	if err := h.db.Model(&device).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": device})
}

// DeleteDevice 删除设备
func (h *FirewallHandler) DeleteDevice(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Firewall{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestSNMP 测试SNMP连接
func (h *FirewallHandler) TestSNMP(c *gin.Context) {
	id := c.Param("id")
	var device Firewall
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "设备不存在"})
		return
	}

	snmp, err := h.createSNMPClient(&device)
	if err != nil {
		now := time.Now()
		_ = h.db.Model(&Firewall{}).Where("id = ?", device.ID).Updates(map[string]interface{}{
			"status":        0,
			"last_check_at": &now,
			"status_reason": truncateFirewallStatusReason(err.Error()),
		}).Error
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SNMP配置错误: " + err.Error()})
		return
	}
	defer closeSNMPConn(snmp)

	// 获取系统描述
	oids := []string{"1.3.6.1.2.1.1.1.0"} // sysDescr
	result, err := snmp.Get(oids)
	if err != nil {
		now := time.Now()
		_ = h.db.Model(&Firewall{}).Where("id = ?", device.ID).Updates(map[string]interface{}{
			"status":        0,
			"last_check_at": &now,
			"status_reason": truncateFirewallStatusReason(err.Error()),
		}).Error
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SNMP连接失败: " + err.Error()})
		return
	}

	sysDescr := ""
	for _, v := range result.Variables {
		if v.Type == gosnmp.OctetString {
			sysDescr = string(v.Value.([]byte))
		}
	}
	now := time.Now()
	_ = h.db.Model(&Firewall{}).Where("id = ?", device.ID).Updates(map[string]interface{}{
		"status":         1,
		"last_check_at":  &now,
		"last_online_at": &now,
		"status_reason":  "",
	}).Error

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"status":   "connected",
		"sys_desc": sysDescr,
	}})
}

// GetSNMPMetrics 获取SNMP指标
func (h *FirewallHandler) GetSNMPMetrics(c *gin.Context) {
	id := c.Param("id")
	metricType := c.DefaultQuery("type", "")

	var metrics []SNMPMetric
	query := h.db.Where("firewall_id = ?", id).Order("collected_at DESC")
	if metricType != "" {
		query = query.Where("metric_type = ?", metricType)
	}
	if err := query.Limit(100).Find(&metrics).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics})
}

// CollectSNMP 采集SNMP数据
func (h *FirewallHandler) CollectSNMP(c *gin.Context) {
	id := c.Param("id")
	metrics, err := h.syncDeviceStatusByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "设备不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "SNMP采集失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics})
}

// ListRules 规则列表
func (h *FirewallHandler) ListRules(c *gin.Context) {
	id := c.Param("id")
	var rules []FirewallRule
	if err := h.db.Where("firewall_id = ?", id).Order("priority").Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

// CreateRule 创建规则
func (h *FirewallHandler) CreateRule(c *gin.Context) {
	id := c.Param("id")
	var rule FirewallRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	rule.FirewallID = id
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// DeleteRule 删除规则
func (h *FirewallHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if err := h.db.Delete(&FirewallRule{}, "id = ?", ruleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *FirewallHandler) createSNMPClient(device *Firewall) (*gosnmp.GoSNMP, error) {
	snmp := &gosnmp.GoSNMP{
		Target:  device.IP,
		Port:    uint16(device.SNMPPort),
		Timeout: time.Duration(5) * time.Second,
		Retries: 2,
	}

	switch device.SNMPVersion {
	case "v1":
		snmp.Version = gosnmp.Version1
		snmp.Community = device.SNMPCommunity
	case "v2c":
		snmp.Version = gosnmp.Version2c
		snmp.Community = device.SNMPCommunity
	case "v3":
		snmp.Version = gosnmp.Version3
		snmp.SecurityModel = gosnmp.UserSecurityModel
		snmp.MsgFlags = gosnmp.AuthPriv
		snmp.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 device.SNMPUser,
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: device.SNMPAuthPass,
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        device.SNMPPrivPass,
		}
	default:
		return nil, fmt.Errorf("不支持的SNMP版本: %s", device.SNMPVersion)
	}

	if err := snmp.Connect(); err != nil {
		return nil, err
	}

	return snmp, nil
}
