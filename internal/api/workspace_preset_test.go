package api

import (
	"testing"
	"time"
)

func TestNormalizeWorkspaceScope(t *testing.T) {
	if got := normalizeWorkspaceScope("team"); got != workspacePresetScopeTeam {
		t.Fatalf("expected team scope, got %s", got)
	}
	if got := normalizeWorkspaceScope("TEAM"); got != workspacePresetScopeTeam {
		t.Fatalf("expected case-insensitive team scope, got %s", got)
	}
	if got := normalizeWorkspaceScope("private"); got != workspacePresetScopePrivate {
		t.Fatalf("expected private scope, got %s", got)
	}
	if got := normalizeWorkspaceScope(""); got != workspacePresetScopePrivate {
		t.Fatalf("expected empty scope fallback to private, got %s", got)
	}
}

func TestNormalizeWorkspaceTabs(t *testing.T) {
	tabs, err := normalizeWorkspaceTabs([]workspaceTab{
		{Path: "  "},
		{Path: "host"},
		{Path: "/host", Pinned: true},
		{Path: "/host", Pinned: false},
		{Path: "/k8s/workloads"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tabs) != 2 {
		t.Fatalf("expected 2 unique tabs, got %d", len(tabs))
	}
	if tabs[0].Path != "/host" || !tabs[0].Pinned {
		t.Fatalf("unexpected first tab: %+v", tabs[0])
	}
	if tabs[1].Path != "/k8s/workloads" {
		t.Fatalf("unexpected second tab: %+v", tabs[1])
	}
}

func TestNormalizeWorkspaceTabsEmpty(t *testing.T) {
	_, err := normalizeWorkspaceTabs([]workspaceTab{
		{Path: "foo"},
		{Path: ""},
	})
	if err == nil {
		t.Fatal("expected error for invalid tabs")
	}
}

func TestToWorkspacePresetResponse(t *testing.T) {
	now := time.Now()
	row := &workspacePresetRecord{
		ID:             "preset-1",
		Name:           "值班模板",
		Scope:          workspacePresetScopeTeam,
		OwnerID:        "u1",
		OwnerName:      "admin",
		Recommended:    true,
		SortOrder:      10,
		UseCount:       7,
		LastUsedByID:   "u2",
		LastUsedByName: "alice",
		LastUsedAt:     &now,
		Tabs:           `[{"path":"/monitor/center","pinned":true},{"path":"/alert/events","pinned":false}]`,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	resp := toWorkspacePresetResponse(row, true)
	if !resp.Editable {
		t.Fatal("expected editable=true")
	}
	if len(resp.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(resp.Tabs))
	}
	if resp.Tabs[0].Path != "/monitor/center" || !resp.Tabs[0].Pinned {
		t.Fatalf("unexpected first tab: %+v", resp.Tabs[0])
	}
	if resp.OwnerName != "admin" {
		t.Fatalf("unexpected owner name: %s", resp.OwnerName)
	}
	if !resp.Recommended || resp.SortOrder != 10 || resp.UseCount != 7 {
		t.Fatalf("unexpected stats fields: recommended=%v sort=%d use=%d", resp.Recommended, resp.SortOrder, resp.UseCount)
	}
	if resp.LastUsedByName != "alice" || resp.LastUsedByID != "u2" || resp.LastUsedAt == nil {
		t.Fatalf("unexpected last-used fields: %+v", resp)
	}
}

func TestDefaultTeamWorkspacePresets(t *testing.T) {
	presets := defaultTeamWorkspacePresets()
	if len(presets) < 5 {
		t.Fatalf("expected at least 5 default presets, got %d", len(presets))
	}
	for _, item := range presets {
		if item.Name == "" {
			t.Fatalf("default preset name should not be empty: %+v", item)
		}
		if _, err := normalizeWorkspaceTabs(item.Tabs); err != nil {
			t.Fatalf("default preset %q has invalid tabs: %v", item.Name, err)
		}
	}
}

func TestSanitizeWorkspacePresetName(t *testing.T) {
	valid, err := sanitizeWorkspacePresetName("  值班工作台  ")
	if err != nil {
		t.Fatalf("expected valid name, got error: %v", err)
	}
	if valid != "值班工作台" {
		t.Fatalf("unexpected sanitized name: %s", valid)
	}
	if _, err := sanitizeWorkspacePresetName(" "); err == nil {
		t.Fatal("expected empty name to be rejected")
	}
}
