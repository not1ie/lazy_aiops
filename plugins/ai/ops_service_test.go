package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/plugins/knowledge"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestAIService(t *testing.T) *AIService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIOpsIncident{}, &AIOpsTimelineEvent{}, &knowledge.Document{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	c := &core.Core{
		DB: db,
		AI: core.NewAIService("openai", "", "", "gpt-4o-mini"),
	}
	return NewAIService(db, c)
}

func TestPreflightRiskEscalation(t *testing.T) {
	svc := newTestAIService(t)
	got, err := svc.PreflightRisk(&AIOpsPreflightRequest{
		Command: "kubectl rollout restart deploy/payment",
		Context: "prod",
	})
	if err != nil {
		t.Fatalf("preflight error: %v", err)
	}
	if got.RiskScore < 70 {
		t.Fatalf("expected high risk score, got=%d", got.RiskScore)
	}
	if !got.EscalateApproval {
		t.Fatalf("expected escalate approval=true")
	}
}

func TestBuildTimelineMermaid(t *testing.T) {
	svc := newTestAIService(t)
	now := time.Now().Add(-2 * time.Minute)
	incident := &AIOpsIncident{
		IncidentID:       "CHG-TEST-1",
		Title:            "test incident",
		Query:            "why timeout",
		Status:           "resolved",
		RootCauseAt:      &now,
		FirstFixActionAt: ptrTime(now.Add(30 * time.Second)),
		MTTDSeconds:      30,
		MTTRSeconds:      90,
	}
	if err := svc.db.Create(incident).Error; err != nil {
		t.Fatalf("create incident: %v", err)
	}
	events := []AIOpsTimelineEvent{
		{IncidentID: incident.IncidentID, Stage: "precheck", Status: "success", Detail: "ok"},
		{IncidentID: incident.IncidentID, Stage: "llm_response", Status: "success", Detail: "root cause found"},
		{IncidentID: incident.IncidentID, Stage: "apply", Status: "success", Detail: "restart service"},
		{IncidentID: incident.IncidentID, Stage: "verify", Status: "success", Detail: "latency back"},
	}
	for i := range events {
		if err := svc.db.Create(&events[i]).Error; err != nil {
			t.Fatalf("create event[%d]: %v", i, err)
		}
	}

	got, err := svc.BuildTimeline(&AIOpsTimelineQuery{
		IncidentID: incident.IncidentID,
		Format:     "mermaid",
	})
	if err != nil {
		t.Fatalf("build timeline: %v", err)
	}
	if got["timeline"] == nil {
		t.Fatalf("expected timeline output")
	}
	timeline, ok := got["timeline"].(string)
	if !ok {
		t.Fatalf("timeline type mismatch")
	}
	if timeline == "" || !contains(timeline, "sequenceDiagram") {
		t.Fatalf("unexpected timeline: %s", timeline)
	}
}

func TestGenerateRunbookFromIncident(t *testing.T) {
	svc := newTestAIService(t)
	incident := &AIOpsIncident{
		IncidentID:       "CHG-RUNBOOK-1",
		Title:            "payment timeout",
		Query:            "支付超时",
		Status:           "resolved",
		PlanJSON:         `{"need_approval":true,"risk_level":"medium","steps":[{"title":"重启","action":"restart deploy","risk":"短时抖动","command_hint":"kubectl rollout restart deploy/payment -n payment"}],"validation_steps":["检查 P99 延迟"],"rollback_steps":["回滚 deployment"]}`,
		RootCauseSummary: "连接池耗尽",
	}
	if err := svc.db.Create(incident).Error; err != nil {
		t.Fatalf("create incident: %v", err)
	}
	if err := svc.db.Create(&AIOpsTimelineEvent{IncidentID: incident.IncidentID, Stage: "tool_call", Status: "success", Detail: "fetch metrics"}).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}

	doc, err := svc.GenerateRunbookFromIncident(&AIOpsRunbookGenerateRequest{
		IncidentID: incident.IncidentID,
		Title:      "swarm-oom",
	}, "tester")
	if err != nil {
		t.Fatalf("generate runbook: %v", err)
	}
	if doc.ID == 0 {
		t.Fatalf("expected persisted document")
	}
	if !strings.Contains(doc.Content, "## Remediation Steps") {
		t.Fatalf("unexpected content: %s", doc.Content)
	}
}

func ptrTime(v time.Time) *time.Time { return &v }

func contains(raw, sub string) bool {
	return strings.Contains(raw, sub)
}
