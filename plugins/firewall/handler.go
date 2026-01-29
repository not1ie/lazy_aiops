package firewall

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gosnmp/gosnmp"
	"gorm.io/gorm"
)

type FirewallHandler struct {
	db *gorm.DB
}

func NewFirewallHandler(db *gorm.DB) *FirewallHandler {
	return &FirewallHandler{db: db}
}

// ListDevices 设备列表
func (h *FirewallHandler) ListDevices(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&device).Error; err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SNMP配置错误: " + err.Error()})
		return
	}
	defer snmp.Conn.Close()

	// 获取系统描述
	oids := []string{"1.3.6.1.2.1.1.1.0"} // sysDescr
	result, err := snmp.Get(oids)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SNMP连接失败: " + err.Error()})
		return
	}

	sysDescr := ""
	for _, v := range result.Variables {
		if v.Type == gosnmp.OctetString {
			sysDescr = string(v.Value.([]byte))
		}
	}

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
	var device Firewall
	if err := h.db.First(&device, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "设备不存在"})
		return
	}

	snmp, err := h.createSNMPClient(&device)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	defer snmp.Conn.Close()

	now := time.Now()
	metrics := make([]SNMPMetric, 0)

	// 通用OID
	oids := map[string]string{
		"sysUpTime":   "1.3.6.1.2.1.1.3.0",
		"sysName":     "1.3.6.1.2.1.1.5.0",
		"ifNumber":    "1.3.6.1.2.1.2.1.0",
	}

	// 根据厂商添加特定OID
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

	oidList := make([]string, 0)
	for _, oid := range oids {
		oidList = append(oidList, oid)
	}

	result, err := snmp.Get(oidList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "SNMP采集失败: " + err.Error()})
		return
	}

	for _, v := range result.Variables {
		var value float64
		switch v.Type {
		case gosnmp.Integer:
			value = float64(v.Value.(int))
		case gosnmp.Gauge32:
			value = float64(v.Value.(uint))
		case gosnmp.Counter32:
			value = float64(v.Value.(uint))
		case gosnmp.Counter64:
			value = float64(v.Value.(uint64))
		default:
			continue
		}

		metricType := "unknown"
		metricName := v.Name
		unit := ""

		// 识别指标类型
		for name, oid := range oids {
			if v.Name == "."+oid || v.Name == oid {
				metricName = name
				switch name {
				case "cpuUsage":
					metricType = "cpu"
					unit = "%"
				case "memUsage":
					metricType = "memory"
					unit = "%"
				case "sessionCount":
					metricType = "session"
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

	// 保存指标
	for _, m := range metrics {
		h.db.Create(&m)
	}

	// 更新设备状态
	updates := map[string]interface{}{"last_check_at": now, "status": 1}
	for _, m := range metrics {
		switch m.MetricType {
		case "cpu":
			updates["cpu_usage"] = m.Value
		case "memory":
			updates["memory_usage"] = m.Value
		case "session":
			updates["session_count"] = int64(m.Value)
		}
	}
	h.db.Model(&device).Updates(updates)

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
		Target:    device.IP,
		Port:      uint16(device.SNMPPort),
		Timeout:   time.Duration(5) * time.Second,
		Retries:   2,
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
