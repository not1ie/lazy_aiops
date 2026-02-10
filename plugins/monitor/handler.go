package monitor

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MonitorHandler struct {
	db           *gorm.DB
	collector    *Collector
	promURL      string
	pushURL      string
	agentSecret  string
	agentTimeout time.Duration
	httpClient   *http.Client
}

func NewMonitorHandler(db *gorm.DB, collector *Collector, promURL, pushURL, agentSecret string, agentTimeout time.Duration) *MonitorHandler {
	return &MonitorHandler{
		db:           db,
		collector:    collector,
		promURL:      promURL,
		pushURL:      pushURL,
		agentSecret:  agentSecret,
		agentTimeout: agentTimeout,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// GetSettings 获取监控配置
func (h *MonitorHandler) GetSettings(c *gin.Context) {
	setting, _ := h.loadSetting()
	if setting == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"prometheus_url":  h.promURL,
			"pushgateway_url": h.pushURL,
			"auth_type":       "none",
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": setting})
}

// UpdateSettings 更新监控配置
func (h *MonitorHandler) UpdateSettings(c *gin.Context) {
	var req MonitorSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
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

	setting, _ := h.loadSetting()
	if setting == nil {
		setting = &MonitorSetting{}
	}
	setting.PrometheusURL = req.PrometheusURL
	setting.PushgatewayURL = req.PushgatewayURL
	setting.AuthType = req.AuthType
	setting.Username = req.Username
	setting.Password = req.Password
	setting.Token = req.Token

	if setting.ID == "" {
		if err := h.db.Create(setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	} else {
		if err := h.db.Save(setting).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
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

func (h *MonitorHandler) loadSetting() (*MonitorSetting, error) {
	var setting MonitorSetting
	if err := h.db.Order("updated_at desc").First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (h *MonitorHandler) getPromConfig() (string, string, string, string, string) {
	setting, err := h.loadSetting()
	if err != nil || setting == nil || setting.PrometheusURL == "" {
		return h.promURL, "none", "", "", ""
	}
	return setting.PrometheusURL, setting.AuthType, setting.Username, setting.Password, setting.Token
}

// AgentHeartbeat 采集器心跳
func (h *MonitorHandler) AgentHeartbeat(c *gin.Context) {
	if h.agentSecret != "" {
		token := c.GetHeader("X-Agent-Token")
		if token == "" {
			token = c.Query("token")
		}
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
		AgentID:  req.AgentID,
		Timestamp: now,
		Labels:   agent.Labels,
		Meta:     agent.Meta,
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
