package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	"github.com/lazyautoops/lazy-auto-ops/plugins/jump"
	"github.com/lazyautoops/lazy-auto-ops/plugins/monitor"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&cmdb.Host{},
		&docker.DockerHost{},
		&monitor.MetricRecord{},
		&monitor.AgentHeartbeat{},
		&jump.JumpIntegrationConfig{},
	); err != nil {
		t.Fatalf("migrate schema failed: %v", err)
	}
	return db
}

func TestGetOverviewReturnsContractAndSummary(t *testing.T) {
	db := newDashboardTestDB(t)

	now := time.Now()
	lastCheckOld := now.Add(-10 * time.Minute)
	lastSeenOffline := now.Add(-5 * time.Minute)
	metricAt := now.Add(-1 * time.Minute)
	syncAt := now.Add(-2 * time.Minute)

	host := cmdb.Host{
		Name:        "host-offline",
		IP:          "192.168.1.10",
		LastCheckAt: &lastCheckOld,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("create host failed: %v", err)
	}
	if err := db.Model(&cmdb.Host{}).Where("id = ?", host.ID).Update("status", 0).Error; err != nil {
		t.Fatalf("set host status failed: %v", err)
	}
	dh := docker.DockerHost{
		Name:        "docker-a",
		HostID:      host.ID,
		Status:      "online",
		LastCheckAt: &lastCheckOld,
	}
	if err := db.Create(&dh).Error; err != nil {
		t.Fatalf("create docker host failed: %v", err)
	}
	agent := monitor.AgentHeartbeat{
		AgentID:  "agent-1",
		Hostname: "node-a",
		IP:       "10.0.0.2",
		LastSeen: lastSeenOffline,
		Status:   "online",
		CPU:      11.2,
		Memory:   20.1,
		Disk:     31.4,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent failed: %v", err)
	}
	metric := monitor.MetricRecord{
		Timestamp:   metricAt,
		CPUUsage:    12.3,
		MemoryUsage: 34.5,
		DiskUsage:   56.7,
		NetworkIn:   1024 * 1024,
		NetworkOut:  1024 * 1024,
	}
	if err := db.Create(&metric).Error; err != nil {
		t.Fatalf("create metric failed: %v", err)
	}
	cfg := jump.JumpIntegrationConfig{
		Enabled:        true,
		LastSyncStatus: "failed",
		LastSyncMsg:    "permission denied",
		LastSyncAt:     &syncAt,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create jump config failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDashboardHandler(db, 90*time.Second)
	r.GET("/api/v1/dashboard/overview", h.GetOverview)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/overview?hours=24", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ContractVersion string `json:"contract_version"`
			Summary         struct {
				HostTotal    int               `json:"host_total"`
				HostOffline  int               `json:"host_offline"`
				DockerTotal  int               `json:"docker_total"`
				AgentTotal   int               `json:"agent_total"`
				AgentOnline  int               `json:"agent_online"`
				ModuleStatus map[string]string `json:"module_status"`
			} `json:"summary"`
			Snapshots struct {
				Hosts []map[string]interface{} `json:"hosts"`
			} `json:"snapshots"`
			Quality struct {
				TrustScore int    `json:"trust_score"`
				TrustGrade string `json:"trust_grade"`
				Dimensions []struct {
					Key   string `json:"key"`
					Score int    `json:"score"`
				} `json:"dimensions"`
				ActionHints []struct {
					Key      string `json:"key"`
					Priority int    `json:"priority"`
				} `json:"action_hints"`
			} `json:"quality"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d", resp.Code)
	}
	if resp.Data.ContractVersion == "" {
		t.Fatalf("contract version should not be empty")
	}
	if resp.Data.Summary.HostTotal != 1 || resp.Data.Summary.HostOffline != 1 {
		t.Fatalf("unexpected host summary: %+v", resp.Data.Summary)
	}
	if resp.Data.Summary.DockerTotal != 1 {
		t.Fatalf("unexpected docker summary: %+v", resp.Data.Summary)
	}
	if resp.Data.Summary.AgentTotal != 1 || resp.Data.Summary.AgentOnline != 0 {
		t.Fatalf("unexpected agent summary: %+v", resp.Data.Summary)
	}
	if len(resp.Data.Snapshots.Hosts) != 1 {
		t.Fatalf("expected one host snapshot, got %d", len(resp.Data.Snapshots.Hosts))
	}
	if resp.Data.Summary.ModuleStatus["cmdb"] != "error" {
		t.Fatalf("expected cmdb module error, got %q", resp.Data.Summary.ModuleStatus["cmdb"])
	}
	if resp.Data.Summary.ModuleStatus["jump"] != "warning" && resp.Data.Summary.ModuleStatus["jump"] != "error" {
		t.Fatalf("expected jump warning/error, got %q", resp.Data.Summary.ModuleStatus["jump"])
	}
	if resp.Data.Quality.TrustScore <= 0 || resp.Data.Quality.TrustScore > 100 {
		t.Fatalf("unexpected trust score: %d", resp.Data.Quality.TrustScore)
	}
	if resp.Data.Quality.TrustGrade == "" {
		t.Fatalf("expected trust grade")
	}
	if len(resp.Data.Quality.Dimensions) < 4 {
		t.Fatalf("expected at least 4 quality dimensions, got %d", len(resp.Data.Quality.Dimensions))
	}
	if len(resp.Data.Quality.ActionHints) == 0 {
		t.Fatalf("expected non-empty action hints")
	}
	hasJumpHint := false
	for _, hint := range resp.Data.Quality.ActionHints {
		if hint.Key == "jump-sync" {
			hasJumpHint = true
			break
		}
	}
	if !hasJumpHint {
		t.Fatalf("expected jump-sync action hint, got %+v", resp.Data.Quality.ActionHints)
	}
}
