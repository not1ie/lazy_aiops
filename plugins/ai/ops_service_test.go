package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestAIService(t *testing.T) *AIService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIOpsIncident{}, &AIOpsTimelineEvent{}); err != nil {
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

func ptrTime(v time.Time) *time.Time { return &v }

func contains(raw, sub string) bool {
	return strings.Contains(raw, sub)
}
