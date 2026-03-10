package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/security"
	"gorm.io/gorm"
)

type MonitorHandler struct {
	db           *gorm.DB
	collector    *Collector
	promURL      string
	pushURL      string
	agentSecret  string
	secretKey    string
	agentTimeout time.Duration
	httpClient   *http.Client
}

func NewMonitorHandler(db *gorm.DB, collector *Collector, promURL, pushURL, agentSecret string, agentTimeout time.Duration, secretKey string) *MonitorHandler {
	return &MonitorHandler{
		db:           db,
		collector:    collector,
		promURL:      promURL,
		pushURL:      pushURL,
		agentSecret:  agentSecret,
		secretKey:    secretKey,
		agentTimeout: agentTimeout,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// ListSettings 获取监控配置列表
func (h *MonitorHandler) ListSettings(c *gin.Context) {
	var list []MonitorSetting
	h.db.Order("updated_at desc").Find(&list)
	for i := range list {
		h.sanitizeSettingSecrets(&list[i])
	}
	if len(list) == 0 && h.promURL != "" {
		list = append(list, MonitorSetting{
			Name:           "default",
			PrometheusURL:  h.promURL,
			PushgatewayURL: h.pushURL,
			AuthType:       "none",
			Active:         true,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// CreateSetting 新增监控配置
func (h *MonitorHandler) CreateSetting(c *gin.Context) {
	var req MonitorSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.PrometheusURL = strings.TrimSpace(req.PrometheusURL)
	req.PushgatewayURL = strings.TrimSpace(req.PushgatewayURL)
	req.AuthType = strings.TrimSpace(req.AuthType)
	if req.AuthType == "" {
		req.AuthType = "none"
	}
	if req.AuthType != "none" && req.AuthType != "basic" && req.AuthType != "bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "auth_type 仅支持 none/basic/bearer"})
		return
	}
	if req.Name == "" {
		req.Name = "prometheus"
	}
	// 如果没有任何 active，则默认激活该条
	var activeCount int64
	h.db.Model(&MonitorSetting{}).Where("active = ?", true).Count(&activeCount)
	if activeCount == 0 {
		req.Active = true
	}
	var err error
	req.Password, err = security.Encrypt(h.secretKey, "monitor.setting.password", req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置密码加密失败"})
		return
	}
	req.Token, err = security.Encrypt(h.secretKey, "monitor.setting.token", req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置令牌加密失败"})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.sanitizeSettingSecrets(&req)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": req})
}

// GetSetting 获取监控配置详情（用于编辑）
func (h *MonitorHandler) GetSetting(c *gin.Context) {
	id := c.Param("id")
	var setting MonitorSetting
	if err := h.db.First(&setting, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	if err := h.decryptSettingSecrets(&setting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置解密失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": setting})
}

// UpdateSetting 更新监控配置
func (h *MonitorHandler) UpdateSetting(c *gin.Context) {
	id := c.Param("id")
	var current MonitorSetting
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}
	var req MonitorSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.PrometheusURL = strings.TrimSpace(req.PrometheusURL)
	req.PushgatewayURL = strings.TrimSpace(req.PushgatewayURL)
	req.AuthType = strings.TrimSpace(req.AuthType)
	if req.AuthType == "" {
		req.AuthType = "none"
	}
	if req.AuthType != "none" && req.AuthType != "basic" && req.AuthType != "bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "auth_type 仅支持 none/basic/bearer"})
		return
	}
	updates := map[string]interface{}{
		"name":            req.Name,
		"prometheus_url":  req.PrometheusURL,
		"pushgateway_url": req.PushgatewayURL,
		"auth_type":       req.AuthType,
		"username":        req.Username,
	}
	if strings.TrimSpace(req.Password) != "" {
		enc, err := security.Encrypt(h.secretKey, "monitor.setting.password", req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置密码加密失败"})
			return
		}
		updates["password"] = enc
	}
	if strings.TrimSpace(req.Token) != "" {
		enc, err := security.Encrypt(h.secretKey, "monitor.setting.token", req.Token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置令牌加密失败"})
			return
		}
		updates["token"] = enc
	}
	if err := h.db.Model(&MonitorSetting{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.First(&current, "id = ?", id).Error; err == nil {
		h.sanitizeSettingSecrets(&current)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": current})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteSetting 删除监控配置
func (h *MonitorHandler) DeleteSetting(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&MonitorSetting{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ActivateSetting 激活某个配置
func (h *MonitorHandler) ActivateSetting(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&MonitorSetting{}).Where("active = ?", true).Update("active", false)
	if err := h.db.Model(&MonitorSetting{}).Where("id = ?", id).Update("active", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已激活"})
}

// TestSetting 测试某个配置
func (h *MonitorHandler) TestSetting(c *gin.Context) {
	id := c.Param("id")
	var setting MonitorSetting
	if err := h.db.First(&setting, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "配置不存在"})
		return
	}
	if setting.PrometheusURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Prometheus 地址为空"})
		return
	}
	if err := h.decryptSettingSecrets(&setting); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "配置解密失败"})
		return
	}
	reqURL, _ := url.Parse(setting.PrometheusURL + "/api/v1/query")
	params := reqURL.Query()
	params.Set("query", "up")
	reqURL.RawQuery = params.Encode()
	h.proxyPromGet(c, reqURL.String(), setting.AuthType, setting.Username, setting.Password, setting.Token)
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
	data := gin.H{}

	// 优先读取本地采集器，低质量数据会由 Prometheus 覆盖
	if h.collector != nil {
		metrics := h.collector.GetMetrics()
		data = gin.H{
			"cpu":     roundMetric(metrics.CPU.Usage),
			"memory":  roundMetric(metrics.Memory.Usage),
			"disk":    roundMetric(metrics.Disk.Usage),
			"network": roundMetric(float64(metrics.Network.InboundRate+metrics.Network.OutboundRate) / (1024 * 1024)),
		}
	}

	// 使用 Prometheus 填补/覆盖采集器中的异常零值
	promMetrics, err := h.queryPromSystemMetrics()
	if err == nil {
		data = mergeMetricPayload(data, promMetrics)
	}

	if len(data) == 0 {
		data = gin.H{"cpu": 0, "memory": 0, "disk": 0, "network": 0}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sanitizeMetricPayload(data)})
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

	if metricHistoryWeak(records) {
		if promRecords, err := h.queryPromSystemHistory(hours); err == nil && len(promRecords) > 0 {
			records = promRecords
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// GetServerMetrics 获取服务器监控数据
func (h *MonitorHandler) GetServerMetrics(c *gin.Context) {
	// 本地采集器若包含可用指标优先返回
	if h.collector != nil {
		metrics := h.collector.GetMetrics()
		if hostsMetricUsable(metrics.Hosts) {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": metrics.Hosts})
			return
		}
	}

	// Collector 不可用或数据为空时，回退到 Prometheus 聚合主机指标
	hosts, err := h.queryPromHostsMetrics()
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": hosts})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": []gin.H{}})
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
			"labels":   []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00", "24:00"},
			"inbound":  []int{80, 95, 85, 110, 125, 115, 100},
			"outbound": []int{60, 75, 65, 90, 105, 95, 80},
		}
	default:
		data = gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}

// ProxyPromQuery Prometheus 即时查询
func (h *MonitorHandler) ProxyPromQuery(c *gin.Context) {
	promURL, authType, username, password, token := h.getPromConfig()
	if promURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Prometheus 未配置"})
		return
	}
	q := c.Query("query")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "query不能为空"})
		return
	}
	reqURL, _ := url.Parse(promURL + "/api/v1/query")
	params := reqURL.Query()
	params.Set("query", q)
	if t := c.Query("time"); t != "" {
		params.Set("time", t)
	}
	reqURL.RawQuery = params.Encode()
	h.proxyPromGet(c, reqURL.String(), authType, username, password, token)
}

// ProxyPromQueryRange Prometheus 范围查询
func (h *MonitorHandler) ProxyPromQueryRange(c *gin.Context) {
	promURL, authType, username, password, token := h.getPromConfig()
	if promURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Prometheus 未配置"})
		return
	}
	q := c.Query("query")
	start := c.Query("start")
	end := c.Query("end")
	step := c.Query("step")
	if q == "" || start == "" || end == "" || step == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "query/start/end/step不能为空"})
		return
	}
	reqURL, _ := url.Parse(promURL + "/api/v1/query_range")
	params := reqURL.Query()
	params.Set("query", q)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)
	reqURL.RawQuery = params.Encode()
	h.proxyPromGet(c, reqURL.String(), authType, username, password, token)
}

// ProxyPromBuildInfo 获取 Prometheus 构建信息
func (h *MonitorHandler) ProxyPromBuildInfo(c *gin.Context) {
	promURL, authType, username, password, token := h.getPromConfig()
	if promURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Prometheus 未配置"})
		return
	}
	reqURL := promURL + "/api/v1/status/buildinfo"
	h.proxyPromGet(c, reqURL, authType, username, password, token)
}

// ProxyPromRuntimeInfo 获取 Prometheus 运行信息
func (h *MonitorHandler) ProxyPromRuntimeInfo(c *gin.Context) {
	promURL, authType, username, password, token := h.getPromConfig()
	if promURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Prometheus 未配置"})
		return
	}
	reqURL := promURL + "/api/v1/status/runtimeinfo"
	h.proxyPromGet(c, reqURL, authType, username, password, token)
}

// ProxyPushgatewayMetrics 获取 Pushgateway 指标
func (h *MonitorHandler) ProxyPushgatewayMetrics(c *gin.Context) {
	if h.pushURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Pushgateway 未配置"})
		return
	}
	reqURL := h.pushURL + "/metrics"
	h.proxyGetRaw(c, reqURL)
}

// ListPromHistory 查询历史
func (h *MonitorHandler) ListPromHistory(c *gin.Context) {
	var items []PromQueryHistory
	query := h.db.Order("created_at DESC")
	if mode := c.Query("mode"); mode != "" {
		query = query.Where("mode = ?", mode)
	}
	if fav := c.Query("favorite"); fav != "" {
		if fav == "1" || fav == "true" {
			query = query.Where("favorite = ?", true)
		}
	}
	if err := query.Limit(50).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// CreatePromHistory 创建查询历史
func (h *MonitorHandler) CreatePromHistory(c *gin.Context) {
	var req struct {
		Mode  string `json:"mode" binding:"required"`
		Query string `json:"query" binding:"required"`
		Start string `json:"start"`
		End   string `json:"end"`
		Step  string `json:"step"`
		Name  string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	item := PromQueryHistory{
		Mode:      req.Mode,
		Query:     req.Query,
		Start:     req.Start,
		End:       req.End,
		Step:      req.Step,
		Name:      req.Name,
		CreatedBy: c.GetString("username"),
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

// UpdatePromHistory 更新名称/收藏
func (h *MonitorHandler) UpdatePromHistory(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name     string `json:"name"`
		Favorite *bool  `json:"favorite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Favorite != nil {
		updates["favorite"] = *req.Favorite
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无可更新字段"})
		return
	}
	if err := h.db.Model(&PromQueryHistory{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// ListDashboards 监控大盘模板
func (h *MonitorHandler) ListDashboards(c *gin.Context) {
	scope := c.Query("scope")
	query := h.db.Order("updated_at DESC")
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	var items []DashboardTemplate
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// SaveDashboard 创建/覆盖模板
func (h *MonitorHandler) SaveDashboard(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Scope   string `json:"scope" binding:"required"`
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	user := c.GetString("username")
	var item DashboardTemplate
	err := h.db.Where("name = ? AND scope = ? AND created_by = ?", req.Name, req.Scope, user).First(&item).Error
	if err == nil {
		updates := map[string]interface{}{
			"payload": req.Payload,
		}
		if err := h.db.Model(&DashboardTemplate{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
		return
	}
	item = DashboardTemplate{
		Name:      req.Name,
		Scope:     req.Scope,
		Payload:   req.Payload,
		CreatedBy: user,
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

// UpdateDashboard 更新模板
func (h *MonitorHandler) UpdateDashboard(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Payload != "" {
		updates["payload"] = req.Payload
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无可更新字段"})
		return
	}
	if err := h.db.Model(&DashboardTemplate{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteDashboard 删除模板
func (h *MonitorHandler) DeleteDashboard(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&DashboardTemplate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *MonitorHandler) proxyGet(c *gin.Context, url string) {
	resp, err := h.httpClient.Get(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

func (h *MonitorHandler) proxyPromGet(c *gin.Context, url, authType, username, password, token string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	switch authType {
	case "basic":
		req.SetBasicAuth(username, password)
	case "bearer":
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

func (h *MonitorHandler) proxyGetRaw(c *gin.Context, url string) {
	resp, err := h.httpClient.Get(url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func (h *MonitorHandler) getPromConfig() (string, string, string, string, string) {
	var setting MonitorSetting
	if err := h.db.Where("active = ?", true).Order("updated_at desc").First(&setting).Error; err != nil || setting.PrometheusURL == "" {
		return h.promURL, "none", "", "", ""
	}
	if err := h.decryptSettingSecrets(&setting); err != nil {
		return setting.PrometheusURL, setting.AuthType, setting.Username, "", ""
	}
	return setting.PrometheusURL, setting.AuthType, setting.Username, setting.Password, setting.Token
}

type promQueryAPIResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"`
		Result     []promVectorResult `json:"result"`
	} `json:"data"`
	Error string `json:"error"`
}

type promRangeAPIResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string            `json:"resultType"`
		Result     []promRangeResult `json:"result"`
	} `json:"data"`
	Error string `json:"error"`
}

type promVectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type promRangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

func (h *MonitorHandler) queryPromVector(query string) ([]promVectorResult, error) {
	promURL, authType, username, password, token := h.getPromConfig()
	if strings.TrimSpace(promURL) == "" {
		return nil, fmt.Errorf("Prometheus 未配置")
	}
	reqURL, _ := url.Parse(strings.TrimRight(promURL, "/") + "/api/v1/query")
	params := reqURL.Query()
	params.Set("query", strings.TrimSpace(query))
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	switch authType {
	case "basic":
		req.SetBasicAuth(username, password)
	case "bearer":
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Prometheus 查询失败: %s", strings.TrimSpace(string(body)))
	}

	var parsed promQueryAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "success" {
		if parsed.Error != "" {
			return nil, fmt.Errorf(parsed.Error)
		}
		return nil, fmt.Errorf("Prometheus 查询失败")
	}
	return parsed.Data.Result, nil
}

func (h *MonitorHandler) queryPromRange(query string, start, end int64, step int) ([]promRangeResult, error) {
	promURL, authType, username, password, token := h.getPromConfig()
	if strings.TrimSpace(promURL) == "" {
		return nil, fmt.Errorf("Prometheus 未配置")
	}
	reqURL, _ := url.Parse(strings.TrimRight(promURL, "/") + "/api/v1/query_range")
	params := reqURL.Query()
	params.Set("query", strings.TrimSpace(query))
	params.Set("start", strconv.FormatInt(start, 10))
	params.Set("end", strconv.FormatInt(end, 10))
	params.Set("step", strconv.Itoa(step))
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	switch authType {
	case "basic":
		req.SetBasicAuth(username, password)
	case "bearer":
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Prometheus 范围查询失败: %s", strings.TrimSpace(string(body)))
	}

	var parsed promRangeAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Status != "success" {
		if parsed.Error != "" {
			return nil, fmt.Errorf(parsed.Error)
		}
		return nil, fmt.Errorf("Prometheus 范围查询失败")
	}
	return parsed.Data.Result, nil
}

func parsePromFlexibleNumber(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func parsePromMatrixPoint(raw []interface{}) (time.Time, float64, bool) {
	if len(raw) < 2 {
		return time.Time{}, 0, false
	}
	tsFloat := parsePromFlexibleNumber(raw[0])
	if tsFloat <= 0 {
		return time.Time{}, 0, false
	}
	val := parsePromFlexibleNumber(raw[1])
	return time.Unix(int64(tsFloat), 0), val, true
}

func parsePromNumber(raw []interface{}) float64 {
	if len(raw) < 2 {
		return 0
	}
	switch v := raw[1].(type) {
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	case float64:
		return v
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
}

func (h *MonitorHandler) queryPromSingleValue(query string) (float64, bool, error) {
	result, err := h.queryPromVector(query)
	if err != nil {
		return 0, false, err
	}
	if len(result) == 0 {
		return 0, false, nil
	}
	return parsePromNumber(result[0].Value), true, nil
}

func (h *MonitorHandler) queryPromSystemMetrics() (gin.H, error) {
	metrics := gin.H{
		"cpu":     0.0,
		"memory":  0.0,
		"disk":    0.0,
		"network": 0.0,
	}
	hasData := false

	if v, ok, err := h.queryPromSingleValue(`100 - (avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`); err == nil && ok {
		metrics["cpu"] = roundMetric(v)
		hasData = true
	}
	if v, ok, err := h.queryPromSingleValue(`100 * (1 - (sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes)))`); err == nil && ok {
		metrics["memory"] = roundMetric(v)
		hasData = true
	}
	if v, ok, err := h.queryPromSingleValue(`100 - (sum(node_filesystem_free_bytes{fstype!="tmpfs",mountpoint="/"}) / sum(node_filesystem_size_bytes{fstype!="tmpfs",mountpoint="/"}) * 100)`); err == nil && ok {
		metrics["disk"] = roundMetric(v)
		hasData = true
	}
	if v, ok, err := h.queryPromSingleValue(`sum(rate(node_network_receive_bytes_total[5m]) + rate(node_network_transmit_bytes_total[5m])) / 1024 / 1024`); err == nil && ok {
		metrics["network"] = roundMetric(v)
		hasData = true
	}

	if !hasData {
		return nil, fmt.Errorf("未获取到 Prometheus 系统指标")
	}
	return metrics, nil
}

func promHistoryStepSeconds(hours int) int {
	switch {
	case hours <= 1:
		return 30
	case hours <= 6:
		return 60
	case hours <= 24:
		return 120
	default:
		return 300
	}
}

func (h *MonitorHandler) queryPromSystemHistory(hours int) ([]MetricRecord, error) {
	end := time.Now().Unix()
	start := time.Now().Add(-time.Duration(hours) * time.Hour).Unix()
	step := promHistoryStepSeconds(hours)

	cpuRange, err := h.queryPromRange(`100 - (avg(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`, start, end, step)
	if err != nil {
		return nil, err
	}
	memRange, _ := h.queryPromRange(`100 * (1 - (sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes)))`, start, end, step)
	diskRange, _ := h.queryPromRange(`100 - (sum(node_filesystem_free_bytes{fstype!="tmpfs",mountpoint="/"}) / sum(node_filesystem_size_bytes{fstype!="tmpfs",mountpoint="/"}) * 100)`, start, end, step)
	netInRange, _ := h.queryPromRange(`sum(rate(node_network_receive_bytes_total[5m]))`, start, end, step)
	netOutRange, _ := h.queryPromRange(`sum(rate(node_network_transmit_bytes_total[5m]))`, start, end, step)

	recordsMap := map[int64]*MetricRecord{}
	getRecord := func(ts time.Time) *MetricRecord {
		key := ts.Unix()
		if item, ok := recordsMap[key]; ok {
			return item
		}
		item := &MetricRecord{Timestamp: ts}
		recordsMap[key] = item
		return item
	}
	applySeries := func(result []promRangeResult, setter func(*MetricRecord, float64)) {
		if len(result) == 0 {
			return
		}
		for _, point := range result[0].Values {
			ts, value, ok := parsePromMatrixPoint(point)
			if !ok {
				continue
			}
			setter(getRecord(ts), roundMetric(value))
		}
	}

	applySeries(cpuRange, func(item *MetricRecord, value float64) { item.CPUUsage = value })
	applySeries(memRange, func(item *MetricRecord, value float64) { item.MemoryUsage = value })
	applySeries(diskRange, func(item *MetricRecord, value float64) { item.DiskUsage = value })
	applySeries(netInRange, func(item *MetricRecord, value float64) { item.NetworkIn = uint64(value) })
	applySeries(netOutRange, func(item *MetricRecord, value float64) { item.NetworkOut = uint64(value) })

	if len(recordsMap) == 0 {
		return nil, fmt.Errorf("未获取到 Prometheus 历史指标")
	}
	keys := make([]int64, 0, len(recordsMap))
	for k := range recordsMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	records := make([]MetricRecord, 0, len(keys))
	for _, k := range keys {
		records = append(records, *recordsMap[k])
	}
	return records, nil
}

func (h *MonitorHandler) queryPromHostsMetrics() ([]gin.H, error) {
	cpuResult, err := h.queryPromVector(`100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`)
	if err != nil {
		return nil, err
	}
	memResult, _ := h.queryPromVector(`100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))`)
	diskResult, _ := h.queryPromVector(`max by(instance) (100 - (node_filesystem_free_bytes{fstype!="tmpfs",mountpoint="/"} / node_filesystem_size_bytes{fstype!="tmpfs",mountpoint="/"} * 100))`)
	loadResult, _ := h.queryPromVector(`node_load1`)
	uptimeResult, _ := h.queryPromVector(`time() - node_boot_time_seconds`)

	type hostItem struct {
		Hostname string
		IP       string
		Status   string
		CPU      float64
		Memory   float64
		Disk     float64
		Load1    float64
		Uptime   string
	}
	hostMap := map[string]*hostItem{}
	getHost := func(instance string, metric map[string]string) *hostItem {
		item, exists := hostMap[instance]
		if exists {
			return item
		}
		hostOrIP := extractHostFromInstance(instance)
		hostname := metric["nodename"]
		if hostname == "" {
			hostname = metric["hostname"]
		}
		if hostname == "" {
			hostname = hostOrIP
		}
		item = &hostItem{
			Hostname: hostname,
			IP:       hostOrIP,
			Status:   "online",
		}
		hostMap[instance] = item
		return item
	}

	for _, item := range cpuResult {
		inst := item.Metric["instance"]
		if inst == "" {
			continue
		}
		host := getHost(inst, item.Metric)
		host.CPU = roundMetric(parsePromNumber(item.Value))
	}
	for _, item := range memResult {
		inst := item.Metric["instance"]
		if inst == "" {
			continue
		}
		host := getHost(inst, item.Metric)
		host.Memory = roundMetric(parsePromNumber(item.Value))
	}
	for _, item := range diskResult {
		inst := item.Metric["instance"]
		if inst == "" {
			continue
		}
		host := getHost(inst, item.Metric)
		host.Disk = roundMetric(parsePromNumber(item.Value))
	}
	for _, item := range loadResult {
		inst := item.Metric["instance"]
		if inst == "" {
			continue
		}
		host := getHost(inst, item.Metric)
		host.Load1 = roundMetric(parsePromNumber(item.Value))
	}
	for _, item := range uptimeResult {
		inst := item.Metric["instance"]
		if inst == "" {
			continue
		}
		host := getHost(inst, item.Metric)
		host.Uptime = formatUptime(parsePromNumber(item.Value))
	}

	if len(hostMap) == 0 {
		return nil, fmt.Errorf("未获取到主机指标")
	}
	instances := make([]string, 0, len(hostMap))
	for k := range hostMap {
		instances = append(instances, k)
	}
	sort.Strings(instances)

	hosts := make([]gin.H, 0, len(instances))
	for _, instance := range instances {
		item := hostMap[instance]
		hosts = append(hosts, gin.H{
			"instance": instance,
			"hostname": item.Hostname,
			"ip":       item.IP,
			"status":   item.Status,
			"cpu":      item.CPU,
			"memory":   item.Memory,
			"disk":     item.Disk,
			"load1":    item.Load1,
			"uptime":   item.Uptime,
		})
	}
	return hosts, nil
}

func extractHostFromInstance(instance string) string {
	v := strings.TrimSpace(instance)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "[") {
		if idx := strings.Index(v, "]"); idx > 1 {
			return v[1:idx]
		}
	}
	if idx := strings.LastIndex(v, ":"); idx > 0 {
		return v[:idx]
	}
	return v
}

func formatUptime(seconds float64) string {
	if seconds <= 0 {
		return "-"
	}
	hours := seconds / 3600
	if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	}
	days := hours / 24
	if days < 30 {
		return fmt.Sprintf("%.1fd", days)
	}
	months := days / 30
	return fmt.Sprintf("%.1fmo", months)
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f
	default:
		return 0
	}
}

func sanitizeMetricPayload(payload gin.H) gin.H {
	out := gin.H{}
	for _, key := range []string{"cpu", "memory", "disk", "network"} {
		value := toFloat(payload[key])
		if value < 0 {
			value = 0
		}
		out[key] = roundMetric(value)
	}
	return out
}

func mergeMetricPayload(base gin.H, fallback gin.H) gin.H {
	base = sanitizeMetricPayload(base)
	fallback = sanitizeMetricPayload(fallback)
	merged := gin.H{}
	for _, key := range []string{"cpu", "memory", "disk", "network"} {
		baseVal := toFloat(base[key])
		fallbackVal := toFloat(fallback[key])
		if baseVal <= 0 && fallbackVal > 0 {
			merged[key] = fallbackVal
		} else {
			merged[key] = baseVal
		}
	}
	return sanitizeMetricPayload(merged)
}

func metricHistoryWeak(records []MetricRecord) bool {
	if len(records) == 0 {
		return true
	}
	healthyPoints := 0
	for _, item := range records {
		if item.CPUUsage > 0 || item.MemoryUsage > 0 || item.NetworkIn > 0 || item.NetworkOut > 0 {
			healthyPoints++
			if healthyPoints >= 2 {
				return false
			}
		}
	}
	return true
}

func hostsMetricUsable(hosts []HostMetrics) bool {
	if len(hosts) == 0 {
		return false
	}
	for _, host := range hosts {
		if host.CPU > 0 || host.Memory > 0 || host.Disk > 0 {
			return true
		}
	}
	return false
}

func roundMetric(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// AgentHeartbeat 采集器心跳
func (h *MonitorHandler) AgentHeartbeat(c *gin.Context) {
	if h.agentSecret != "" {
		token := c.GetHeader("X-Agent-Token")
		if token != h.agentSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
			return
		}
	}

	var req struct {
		AgentID  string                 `json:"agent_id" binding:"required"`
		Hostname string                 `json:"hostname"`
		IP       string                 `json:"ip"`
		Version  string                 `json:"version"`
		OS       string                 `json:"os"`
		Labels   map[string]string      `json:"labels"`
		Meta     map[string]interface{} `json:"meta"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AgentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	now := time.Now()
	var agent AgentHeartbeat
	if err := h.db.First(&agent, "agent_id = ?", req.AgentID).Error; err != nil {
		agent = AgentHeartbeat{
			AgentID:  req.AgentID,
			Hostname: req.Hostname,
			IP:       req.IP,
			Version:  req.Version,
			OS:       req.OS,
			Status:   "online",
			LastSeen: now,
		}
	} else {
		agent.Hostname = req.Hostname
		agent.IP = req.IP
		agent.Version = req.Version
		agent.OS = req.OS
		agent.Status = "online"
		agent.LastSeen = now
	}

	if req.Labels != nil {
		if b, err := json.Marshal(req.Labels); err == nil {
			agent.Labels = string(b)
		}
	}
	if req.Meta != nil {
		if b, err := json.Marshal(req.Meta); err == nil {
			agent.Meta = string(b)
		}
	}
	if req.Meta != nil {
		if v, ok := req.Meta["cpu"].(float64); ok {
			agent.CPU = v
		}
		if v, ok := req.Meta["memory"].(float64); ok {
			agent.Memory = v
		}
		if v, ok := req.Meta["disk"].(float64); ok {
			agent.Disk = v
		}
		if v, ok := req.Meta["net_in"].(float64); ok {
			agent.NetIn = v
		}
		if v, ok := req.Meta["net_out"].(float64); ok {
			agent.NetOut = v
		}
	}

	record := AgentHeartbeatRecord{
		AgentID:   req.AgentID,
		Timestamp: now,
		Labels:    agent.Labels,
		Meta:      agent.Meta,
	}

	if agent.ID == "" {
		if err := h.db.Create(&agent).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	} else {
		if err := h.db.Save(&agent).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}
	_ = h.db.Create(&record).Error

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// ListAgents 采集器列表
func (h *MonitorHandler) ListAgents(c *gin.Context) {
	var agents []AgentHeartbeat
	if err := h.db.Order("last_seen DESC").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	now := time.Now()
	for i := range agents {
		if now.Sub(agents[i].LastSeen) > h.agentTimeout {
			agents[i].Status = "offline"
		} else {
			agents[i].Status = "online"
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": agents})
}

// GetAgent 获取单个采集器详情
func (h *MonitorHandler) GetAgent(c *gin.Context) {
	id := c.Param("id")
	var agent AgentHeartbeat
	if err := h.db.First(&agent, "agent_id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Agent不存在"})
		return
	}
	now := time.Now()
	if now.Sub(agent.LastSeen) > h.agentTimeout {
		agent.Status = "offline"
	} else {
		agent.Status = "online"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": agent})
}

// GetAgentHistory 采集器心跳历史
func (h *MonitorHandler) GetAgentHistory(c *gin.Context) {
	id := c.Param("id")
	hours := 24
	if v := c.Query("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hours = n
		}
	}
	start := time.Now().Add(-time.Duration(hours) * time.Hour)
	var records []AgentHeartbeatRecord
	if err := h.db.Where("agent_id = ? AND timestamp >= ?", id, start).
		Order("timestamp ASC").
		Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

func (h *MonitorHandler) sanitizeSettingSecrets(setting *MonitorSetting) {
	if setting == nil {
		return
	}
	setting.Password = ""
	setting.Token = ""
}

func (h *MonitorHandler) decryptSettingSecrets(setting *MonitorSetting) error {
	if setting == nil {
		return nil
	}
	var err error
	setting.Password, err = security.Decrypt(h.secretKey, "monitor.setting.password", setting.Password)
	if err != nil {
		return err
	}
	setting.Token, err = security.Decrypt(h.secretKey, "monitor.setting.token", setting.Token)
	if err != nil {
		return err
	}
	return nil
}
