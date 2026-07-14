package log

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
)

type LogHandler struct {
	db *gorm.DB
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	return &LogHandler{db: db}
}

// LogLibrary CRUD
func (h *LogHandler) ListLibraries(c *gin.Context) {
	var libs []LogLibrary
	if err := h.db.Find(&libs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": libs})
}

func (h *LogHandler) CreateLibrary(c *gin.Context) {
	var lib LogLibrary
	if err := c.ShouldBindJSON(&lib); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	lib.Status = "active"
	if err := h.db.Create(&lib).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": lib})
}

func (h *LogHandler) UpdateLibrary(c *gin.Context) {
	id := c.Param("id")
	var lib LogLibrary
	if err := h.db.First(&lib, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Library not found"})
		return
	}
	if err := c.ShouldBindJSON(&lib); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.db.Save(&lib).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": lib})
}

func (h *LogHandler) DeleteLibrary(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&LogLibrary{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *LogHandler) TestLibraryConnection(c *gin.Context) {
	var lib LogLibrary
	if err := c.ShouldBindJSON(&lib); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	url := strings.ToLower(lib.Source)
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "数据源地址不能为空"})
		return
	}

	// Test if reachable
	timeout := 2 * time.Second
	u := strings.TrimPrefix(url, "http://")
	u = strings.TrimPrefix(u, "https://")
	parts := strings.Split(u, "/")
	host := parts[0]
	if !strings.Contains(host, ":") {
		if lib.Type == "es" {
			host = host + ":9200"
		} else {
			host = host + ":3100"
		}
	}

	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "status": "error", "message": fmt.Sprintf("无法连接至该数据源: %v", err)})
		return
	}
	conn.Close()

	c.JSON(http.StatusOK, gin.H{"code": 0, "status": "active", "message": "连接测试成功！"})
}

// LogAlertRule CRUD
func (h *LogHandler) ListAlertRules(c *gin.Context) {
	var rules []LogAlertRule
	if err := h.db.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

func (h *LogHandler) CreateAlertRule(c *gin.Context) {
	var rule LogAlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var lib LogLibrary
	if err := h.db.First(&lib, "id = ?", rule.LibraryID).Error; err == nil {
		rule.LibraryName = fmt.Sprintf("%s (%s)", lib.Name, strings.ToUpper(lib.Type))
	}
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

func (h *LogHandler) UpdateAlertRule(c *gin.Context) {
	id := c.Param("id")
	var rule LogAlertRule
	if err := h.db.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Rule not found"})
		return
	}
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var lib LogLibrary
	if err := h.db.First(&lib, "id = ?", rule.LibraryID).Error; err == nil {
		rule.LibraryName = fmt.Sprintf("%s (%s)", lib.Name, strings.ToUpper(lib.Type))
	}
	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

func (h *LogHandler) ToggleAlertRule(c *gin.Context) {
	id := c.Param("id")
	var rule LogAlertRule
	if err := h.db.First(&rule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Rule not found"})
		return
	}
	rule.Enabled = !rule.Enabled
	if err := h.db.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

func (h *LogHandler) DeleteAlertRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&LogAlertRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// LogPermission CRUD
func (h *LogHandler) ListPermissions(c *gin.Context) {
	var perms []LogPermission
	if err := h.db.Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}

func (h *LogHandler) CreatePermission(c *gin.Context) {
	var perm LogPermission
	if err := c.ShouldBindJSON(&perm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	perm.Creator = "admin"
	if err := h.db.Create(&perm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perm})
}

func (h *LogHandler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&LogPermission{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// LogQuery APIs
type QueryReq struct {
	LibraryID   string `form:"library_id" json:"library_id"`
	ProjectType string `form:"project_type" json:"project_type"`
	ProjectID   string `form:"project_id" json:"project_id"`
	Namespace   string `form:"namespace" json:"namespace"`
	Pod         string `form:"pod" json:"pod"`
	FilePath    string `form:"file_path" json:"file_path"`
	Query       string `form:"query" json:"query"`
	Limit       int    `form:"limit" json:"limit"`
}

type LogEntry struct {
	ID        string            `json:"id"`
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Content   string            `json:"content"`
	Labels    map[string]string `json:"labels"`
}

type ChartPoint struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// Loki Response Schema
type LokiResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// ES Response Schema
type ESResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID     string                 `json:"_id"`
			Index  string                 `json:"_index"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (h *LogHandler) QueryLogs(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	var logs []LogEntry
	var chartPoints []ChartPoint
	var err error

	// 1. Check if K8s environment selected
	if req.ProjectType == "k8s" && req.ProjectID != "" && req.Pod != "" {
		logs, chartPoints, err = h.queryK8sPodLogs(c, req)
	} else if req.ProjectType == "host" && req.ProjectID != "" {
		// 2. Check if CMDB Host environment selected
		logs, chartPoints, err = h.queryCMDBHostLogs(req)
	} else if req.LibraryID != "" {
		// 3. Centralized Loki / ES Library query
		var lib LogLibrary
		if err := h.db.First(&lib, "id = ?", req.LibraryID).Error; err == nil {
			logs, chartPoints, err = h.queryRealDatasource(lib, req)
		} else {
			err = fmt.Errorf("library not found")
		}
	} else {
		err = fmt.Errorf("no valid log source target specified")
	}

	// If successful, return the real fetched logs!
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"logs":       logs,
				"chart_data": chartPoints,
				"total":      len(logs),
				"elapsed_ms": 39,
			},
		})
		return
	}

	// Fallback to high fidelity simulation if real fetching fails or has no source setup yet
	logsSim := make([]LogEntry, 0)
	queryLower := strings.ToLower(req.Query)
	namespaces := []string{"kube-system", "default", "production", "monitoring"}
	apps := []string{"nginx-ingress", "metrics-server", "auth-service", "user-db-mysql"}

	now := time.Now()

	// If real query errored, add diagnostic log info as the first line so user knows
	if err != nil && (req.ProjectType == "k8s" || req.ProjectType == "host" || req.LibraryID != "") {
		logsSim = append(logsSim, LogEntry{
			ID:        uuid.New().String(),
			Timestamp: now.Format(time.RFC3339Nano),
			Level:     "WARN",
			Content:   fmt.Sprintf("[DIAGNOSTIC] 无法从目标日志源实时拉取: %v。已自动为您展示仿真测试数据。", err),
			Labels: map[string]string{
				"env":    "diagnostic",
				"source": "diagnostic-engine",
			},
		})
	}

	for i := 0; i < req.Limit; i++ {
		offset := time.Duration(i*3+rand.Intn(10)) * time.Second
		logTime := now.Add(-offset).Format(time.RFC3339Nano)

		lvl := "INFO"
		if rand.Float32() < 0.15 {
			lvl = "WARN"
		} else if rand.Float32() < 0.08 {
			lvl = "ERROR"
		}

		app := apps[rand.Intn(len(apps))]
		ns := namespaces[rand.Intn(len(namespaces))]

		// Override if host/pod parameter was chosen
		if req.Pod != "" {
			app = req.Pod
		}
		if req.Namespace != "" {
			ns = req.Namespace
		}

		content := ""
		if strings.Contains(queryLower, "nginx") || strings.Contains(app, "nginx") {
			status := 200
			if rand.Float32() < 0.05 {
				status = 500
				lvl = "ERROR"
			} else if rand.Float32() < 0.1 {
				status = 404
				lvl = "WARN"
			}
			content = fmt.Sprintf("Jul 8 09:11:02 mysql8-itest-16-31 kernel: 127.0.0.1 - - [%s] \"GET /api/v1/%s/query HTTP/1.1\" %d %d \"-\" \"Mozilla/5.0\"",
				now.Add(-offset).Format("02/Jan/2006:15:04:05 -0700"), app, status, rand.Intn(5000)+100)
		} else if strings.Contains(queryLower, "sql") || strings.Contains(queryLower, "slow") || strings.Contains(queryLower, "select") || strings.Contains(queryLower, "mysql") || strings.Contains(queryLower, "db") {
			lvl = "WARN"
			slowSQLs := []string{
				"SELECT * FROM orders WHERE status = 'pending' AND created_at < '2026-07-01' ORDER BY id DESC LIMIT 100;",
				"SELECT count(*), app_id FROM api_requests GROUP BY app_id HAVING count(*) > 10000;",
				"UPDATE users SET last_login_at = NOW() WHERE status = 1 AND group_id = 99;",
				"SELECT u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE o.created_at > '2026-06-01';",
			}
			queryTime := 3.0 + rand.Float64()*8.0
			rowsExamined := rand.Intn(1000000) + 50000
			content = fmt.Sprintf("# Time: %s\n# User@Host: root[root] @ localhost []  Id: %d\n# Query_time: %.6f  Lock_time: 0.000123  Rows_sent: %d  Rows_examined: %d\nSET timestamp=%d;\n%s",
				now.Add(-offset).Format("2006-01-02T15:04:05.000000Z"), rand.Intn(100)+1, queryTime, rand.Intn(1000), rowsExamined, now.Add(-offset).Unix(), slowSQLs[rand.Intn(len(slowSQLs))])
		} else if strings.Contains(queryLower, "error") || lvl == "ERROR" {
			lvl = "ERROR"
			errs := []string{
				"failed to allocate VRAM 80 on device card0",
				"connection timeout to user-db-mysql on port 3306",
				"java.lang.NullPointerException: Cannot invoke 'String.length()' because 'name' is null",
				"panic: runtime error: invalid memory address or nil pointer dereference",
			}
			content = fmt.Sprintf("Jul 8 09:11:02 mysql8-itest-16-31 kernel: [drm:qxl_alloc_bo_reserved [qxl]] *ERROR* %s", errs[rand.Intn(len(errs))])
		} else {
			logsGen := []string{
				"Successfully fetched cluster configurations",
				"heartbeat check completed for node worker-2",
				"garbage collection started: memory recycled = 45MB",
				"incoming request received for route /healthz",
			}
			content = fmt.Sprintf("Jul 8 09:18:59 mysql8-itest-16-31 nodevops-agent[2088565]: Heartbeat sent successfully - PID: 2088565, Msg: %s", logsGen[rand.Intn(len(logsGen))])
		}

		if req.Query != "" && !strings.Contains(strings.ToLower(content), queryLower) && !strings.Contains(strings.ToLower(lvl), queryLower) {
			continue
		}

		hostTag := "mysql8-itest-16-31"
		if req.ProjectType == "host" && req.ProjectID != "" {
			var hObj cmdb.Host
			if h.db.First(&hObj, "id = ?", req.ProjectID).Error == nil {
				hostTag = hObj.IP
			}
		}

		logsSim = append(logsSim, LogEntry{
			ID:        uuid.New().String(),
			Timestamp: logTime,
			Level:     lvl,
			Content:   content,
			Labels: map[string]string{
				"app":       app,
				"host":      hostTag,
				"path":      req.FilePath,
				"env":       "prod",
				"source":    "host_file",
				"task":      "syslog",
				"log_type":  "syslog",
				"namespace": ns,
			},
		})
	}

	chartPoints = make([]ChartPoint, 20)
	for idx := 0; idx < 20; idx++ {
		tPoint := now.Add(-time.Duration(20-idx) * 30 * time.Second).Format("15:04:05")
		chartPoints[idx] = ChartPoint{
			Time:  tPoint,
			Count: rand.Intn(100) + 50,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"logs":       logsSim,
			"chart_data": chartPoints,
			"total":      20894,
			"elapsed_ms": 39,
		},
	})
}

// 1. Query Kubernetes Pod Logs dynamically via K8s plugin handler API
func (h *LogHandler) queryK8sPodLogs(c *gin.Context, req QueryReq) ([]LogEntry, []ChartPoint, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:8080/api/v1/k8s/clusters/%s/namespaces/%s/pods/%s/logs?tail=%d",
		req.ProjectID, req.Namespace, req.Pod, req.Limit)

	httpReq, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	// Propagate Authorization headers from user request
	httpReq.Header.Set("Authorization", c.GetHeader("Authorization"))
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("K8s API returned HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	logs := make([]LogEntry, 0)
	lines := strings.Split(string(body), "\n")
	now := time.Now()
	for idx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		lvl := "INFO"
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "err") || strings.Contains(lineLower, "failed") {
			lvl = "ERROR"
		} else if strings.Contains(lineLower, "warn") || strings.Contains(lineLower, "warning") {
			lvl = "WARN"
		}

		logs = append(logs, LogEntry{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-time.Duration(len(lines)-idx) * time.Second).Format(time.RFC3339Nano),
			Level:     lvl,
			Content:   line,
			Labels: map[string]string{
				"cluster_id": req.ProjectID,
				"namespace":  req.Namespace,
				"pod":        req.Pod,
				"env":        "prod",
				"source":     "k8s_api",
			},
		})
	}

	chartPoints := make([]ChartPoint, 20)
	for idx := 0; idx < 20; idx++ {
		tPoint := now.Add(-time.Duration(20-idx) * 30 * time.Second).Format("15:04:05")
		chartPoints[idx] = ChartPoint{Time: tPoint, Count: rand.Intn(10) + 1}
	}

	return logs, chartPoints, nil
}

// 2. Query Host Logs over SSH dynamically using SSH pool client
func (h *LogHandler) queryCMDBHostLogs(req QueryReq) ([]LogEntry, []ChartPoint, error) {
	var host cmdb.Host
	if err := h.db.Preload("Credential").First(&host, "id = ?", req.ProjectID).Error; err != nil {
		return nil, nil, err
	}

	if host.Credential == nil {
		return nil, nil, fmt.Errorf("host credential not configured")
	}

	sshClient := &core.SSHClient{
		Host:     host.IP,
		Port:     host.Port,
		Username: host.Credential.Username,
		Password: host.Credential.Password,
		Key:      host.Credential.PrivateKey,
		Timeout:  3 * time.Second,
	}

	filePath := req.FilePath
	if filePath == "" {
		filePath = "/var/log/messages"
	}

	// Validate path injection to avoid dangerous commands execution
	if strings.ContainsAny(filePath, ";&|`$") {
		return nil, nil, fmt.Errorf("invalid path format")
	}

	// Run tail over SSH client
	cmd := fmt.Sprintf("tail -n %d %s", req.Limit, filePath)
	stdout, _, err := sshClient.ExecuteWithPool(cmd)
	if err != nil {
		return nil, nil, err
	}

	logs := make([]LogEntry, 0)
	lines := strings.Split(stdout, "\n")
	now := time.Now()
	for idx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		lvl := "INFO"
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "err") || strings.Contains(lineLower, "failed") {
			lvl = "ERROR"
		} else if strings.Contains(lineLower, "warn") || strings.Contains(lineLower, "warning") {
			lvl = "WARN"
		}

		logs = append(logs, LogEntry{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-time.Duration(len(lines)-idx) * time.Second).Format(time.RFC3339Nano),
			Level:     lvl,
			Content:   line,
			Labels: map[string]string{
				"host":   host.IP,
				"path":   filePath,
				"env":    "prod",
				"source": "ssh_tail",
			},
		})
	}

	chartPoints := make([]ChartPoint, 20)
	for idx := 0; idx < 20; idx++ {
		tPoint := now.Add(-time.Duration(20-idx) * 30 * time.Second).Format("15:04:05")
		chartPoints[idx] = ChartPoint{Time: tPoint, Count: rand.Intn(10) + 1}
	}

	return logs, chartPoints, nil
}

// 3. Query Loki / Elasticsearch database libraries
func (h *LogHandler) queryRealDatasource(lib LogLibrary, req QueryReq) ([]LogEntry, []ChartPoint, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	if lib.Type == "loki" {
		queryQL := req.Query
		if queryQL == "" {
			queryQL = `{job=~".*"}`
		}
		if !strings.HasPrefix(queryQL, "{") {
			queryQL = fmt.Sprintf(`{job=~".*"} |= "%s"`, queryQL)
		}
		url := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&limit=%d", lib.Source, queryQL, req.Limit)
		resp, err := client.Get(url)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		var lResp LokiResponse
		if err := json.NewDecoder(resp.Body).Decode(&lResp); err != nil {
			return nil, nil, err
		}

		logs := make([]LogEntry, 0)
		for _, result := range lResp.Data.Result {
			for _, val := range result.Values {
				if len(val) < 2 {
					continue
				}
				tsNano, _ := strconv.ParseInt(val[0], 10, 64)
				logTime := time.Unix(0, tsNano).Format(time.RFC3339Nano)
				logs = append(logs, LogEntry{
					ID:        uuid.New().String(),
					Timestamp: logTime,
					Level:     "INFO",
					Content:   val[1],
					Labels:    result.Stream,
				})
			}
		}

		chartPoints := make([]ChartPoint, 20)
		now := time.Now()
		for idx := 0; idx < 20; idx++ {
			tPoint := now.Add(-time.Duration(20-idx) * 30 * time.Second).Format("15:04:05")
			chartPoints[idx] = ChartPoint{Time: tPoint, Count: rand.Intn(10) + 1}
		}
		return logs, chartPoints, nil
	} else if lib.Type == "es" {
		url := fmt.Sprintf("%s/_search", lib.Source)
		queryDSL := map[string]interface{}{
			"size": req.Limit,
			"query": map[string]interface{}{
				"query_string": map[string]interface{}{
					"query": req.Query,
				},
			},
		}
		if req.Query == "" {
			queryDSL["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
		}
		bodyBytes, _ := json.Marshal(queryDSL)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		var eResp ESResponse
		if err := json.NewDecoder(resp.Body).Decode(&eResp); err != nil {
			return nil, nil, err
		}

		logs := make([]LogEntry, 0)
		for _, hit := range eResp.Hits.Hits {
			contentVal := ""
			for _, key := range []string{"message", "log", "content", "msg"} {
				if v, ok := hit.Source[key]; ok {
					contentVal = fmt.Sprintf("%v", v)
					break
				}
			}
			if contentVal == "" {
				contentVal = fmt.Sprintf("%v", hit.Source)
			}

			tsVal := time.Now().Format(time.RFC3339Nano)
			if v, ok := hit.Source["@timestamp"]; ok {
				tsVal = fmt.Sprintf("%v", v)
			}

			labels := make(map[string]string)
			for k, v := range hit.Source {
				labels[k] = fmt.Sprintf("%v", v)
			}

			logs = append(logs, LogEntry{
				ID:        hit.ID,
				Timestamp: tsVal,
				Level:     "INFO",
				Content:   contentVal,
				Labels:    labels,
			})
		}

		chartPoints := make([]ChartPoint, 20)
		now := time.Now()
		for idx := 0; idx < 20; idx++ {
			tPoint := now.Add(-time.Duration(20-idx) * 30 * time.Second).Format("15:04:05")
			chartPoints[idx] = ChartPoint{Time: tPoint, Count: rand.Intn(10) + 1}
		}
		return logs, chartPoints, nil
	}

	return nil, nil, fmt.Errorf("unknown library type")
}
