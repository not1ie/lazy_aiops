package monitor

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MonitorHandler struct {
	db        *gorm.DB
	collector *Collector
}

func NewMonitorHandler(db *gorm.DB, collector *Collector) *MonitorHandler {
	return &MonitorHandler{
		db:        db,
		collector: collector,
	}
}

// ListDomains 域名监控列表
func (h *MonitorHandler) ListDomains(c *gin.Context) {
	var domains []DomainMonitor
	if err := h.db.Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domains})
}

// CreateDomain 创建域名监控
func (h *MonitorHandler) CreateDomain(c *gin.Context) {
	var domain DomainMonitor
	if err := c.ShouldBindJSON(&domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&domain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domain})
}

// DeleteDomain 删除域名监控
func (h *MonitorHandler) DeleteDomain(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&DomainMonitor{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListRules 告警规则列表
func (h *MonitorHandler) ListRules(c *gin.Context) {
	var rules []AlertRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

// CreateRule 创建告警规则
func (h *MonitorHandler) CreateRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// ListAlerts 告警记录列表
func (h *MonitorHandler) ListAlerts(c *gin.Context) {
	var alerts []AlertRecord
	query := h.db.Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Limit(100).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": alerts})
}

// GetMetrics 获取监控指标数据
func (h *MonitorHandler) GetMetrics(c *gin.Context) {
	// 如果采集器可用，返回实时数据
	if h.collector != nil {
		metrics := h.collector.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"cpu":     metrics.CPU.Usage,
				"memory":  metrics.Memory.Usage,
				"disk":    metrics.Disk.Usage,
				"network": metrics.Network.InboundRate / (1024 * 1024), // 转换为MB/s
			},
		})
		return
	}
	
	// 否则返回模拟数据
	metrics := gin.H{
		"cpu":     65,
		"memory":  78,
		"disk":    45,
		"network": 125,
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics})
}

// GetRealtimeMetrics 获取实时监控指标（完整数据）
func (h *MonitorHandler) GetRealtimeMetrics(c *gin.Context) {
	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "监控采集器未启动",
		})
		return
	}
	
	metrics := h.collector.GetMetrics()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics})
}

// GetMetricsHistory 获取历史监控数据
func (h *MonitorHandler) GetMetricsHistory(c *gin.Context) {
	// 获取时间范围参数
	hours := 1
	if h := c.Query("hours"); h != "" {
		if v, err := strconv.Atoi(h); err == nil && v > 0 {
			hours = v
		}
	}
	
	// 查询历史数据
	var records []MetricRecord
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	if err := h.db.Where("timestamp >= ?", startTime).
		Order("timestamp ASC").
		Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// GetServerMetrics 获取服务器监控数据
func (h *MonitorHandler) GetServerMetrics(c *gin.Context) {
	// 如果采集器可用，返回实时主机数据
	if h.collector != nil {
		metrics := h.collector.GetMetrics()
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics.Hosts})
		return
	}
	
	// 否则返回模拟数据
	servers := []gin.H{
		{"hostname": "web-01", "ip": "192.168.1.10", "status": "online", "cpu": 45, "memory": 62, "uptime": "15天"},
		{"hostname": "web-02", "ip": "192.168.1.11", "status": "online", "cpu": 38, "memory": 55, "uptime": "15天"},
		{"hostname": "db-01", "ip": "192.168.1.20", "status": "online", "cpu": 78, "memory": 85, "uptime": "30天"},
		{"hostname": "cache-01", "ip": "192.168.1.30", "status": "offline", "cpu": 0, "memory": 0, "uptime": "-"},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": servers})
}

// GetChartData 获取图表数据
func (h *MonitorHandler) GetChartData(c *gin.Context) {
	metricType := c.Query("type")
	
	var data gin.H
	switch metricType {
	case "cpu_memory":
		data = gin.H{
			"labels": []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "24:00"},
			"cpu":    []int{30, 45, 35, 50, 65, 55, 40},
			"memory": []int{50, 60, 55, 70, 78, 72, 65},
		}
	case "network":
		data = gin.H{
			"labels": []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "24:00"},
			"inbound":  []int{80, 95, 85, 110, 125, 115, 100},
			"outbound": []int{60, 75, 65, 90, 105, 95, 80},
		}
	default:
		data = gin.H{}
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}
