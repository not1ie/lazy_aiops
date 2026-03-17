package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	workspacePresetScopePrivate = "private"
	workspacePresetScopeTeam    = "team"
)

type workspaceTab struct {
	Path   string `json:"path"`
	Pinned bool   `json:"pinned"`
}

type workspacePresetRecord struct {
	ID             string `gorm:"primaryKey;size:36"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Name           string         `gorm:"size:64;not null;index;index:uk_workspace_preset_owner_name,unique,priority:3"`
	Scope          string         `gorm:"size:16;not null;index;index:uk_workspace_preset_owner_name,unique,priority:1"`
	OwnerID        string         `gorm:"size:36;not null;index;index:uk_workspace_preset_owner_name,unique,priority:2"`
	OwnerName      string         `gorm:"size:64"`
	Tabs           string         `gorm:"type:text"`
	Version        int            `gorm:"default:1"`
	Recommended    bool           `gorm:"default:false;index"`
	SortOrder      int            `gorm:"default:0;index"`
	UseCount       int            `gorm:"default:0"`
	LastUsedByID   string         `gorm:"size:36"`
	LastUsedByName string         `gorm:"size:64"`
	LastUsedAt     *time.Time
}

type workspacePresetUpsertRequest struct {
	Name        string         `json:"name" binding:"required,max=24"`
	Scope       string         `json:"scope"`
	Tabs        []workspaceTab `json:"tabs" binding:"required"`
	Recommended *bool          `json:"recommended,omitempty"`
	SortOrder   *int           `json:"sort_order,omitempty"`
}

type workspacePresetReorderItem struct {
	ID string `json:"id"`
}

type workspacePresetReorderRequest struct {
	Items []workspacePresetReorderItem `json:"items" binding:"required"`
}

type workspacePresetResponse struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Scope          string         `json:"scope"`
	OwnerID        string         `json:"owner_id"`
	OwnerName      string         `json:"owner_name"`
	Editable       bool           `json:"editable"`
	Recommended    bool           `json:"recommended"`
	SortOrder      int            `json:"sort_order"`
	UseCount       int            `json:"use_count"`
	LastUsedByID   string         `json:"last_used_by_id"`
	LastUsedByName string         `json:"last_used_by_name"`
	LastUsedAt     *time.Time     `json:"last_used_at"`
	Tabs           []workspaceTab `json:"tabs"`
	UpdatedAt      time.Time      `json:"updated_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

type defaultTeamWorkspacePreset struct {
	Name        string
	SortOrder   int
	Recommended bool
	Tabs        []workspaceTab
}

func (w *workspacePresetRecord) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}

func (workspacePresetRecord) TableName() string {
	return "workspace_presets"
}

func (s *Server) setupWorkspacePresetRoutes(g *gin.RouterGroup) {
	g.GET("/user/workspace-presets", s.listWorkspacePresets)
	g.POST("/user/workspace-presets", s.createWorkspacePreset)
	g.PUT("/user/workspace-presets/:id", s.updateWorkspacePreset)
	g.DELETE("/user/workspace-presets/:id", s.deleteWorkspacePreset)
	g.POST("/user/workspace-presets/reorder", s.reorderWorkspacePresets)
	g.POST("/user/workspace-presets/:id/use", s.markWorkspacePresetUsed)
}

func (s *Server) currentAuthUser(c *gin.Context) (*core.User, bool, error) {
	userID := c.GetString("user_id")
	user, err := s.core.Auth.GetUserByID(userID)
	if err != nil {
		return nil, false, err
	}
	if user == nil {
		return nil, false, errors.New("user not found")
	}
	isAdmin := user != nil && user.Role != nil && strings.EqualFold(user.Role.Code, "admin")
	return user, isAdmin, nil
}

func (s *Server) listWorkspacePresets(c *gin.Context) {
	user, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}

	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	db := s.core.DB.Model(&workspacePresetRecord{})
	switch scope {
	case "mine":
		db = db.Where("owner_id = ?", user.ID)
	case "team":
		db = db.Where("scope = ?", workspacePresetScopeTeam)
	default:
		db = db.Where("scope = ? OR owner_id = ?", workspacePresetScopeTeam, user.ID)
	}

	var rows []workspacePresetRecord
	if err := db.Order("recommended desc, sort_order asc, updated_at desc").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取模板失败"})
		return
	}

	editableOnly := strings.EqualFold(strings.TrimSpace(c.Query("editable")), "true")
	resp := make([]workspacePresetResponse, 0, len(rows))
	for _, row := range rows {
		editable := isAdmin || row.OwnerID == user.ID
		if editableOnly && !editable {
			continue
		}
		resp = append(resp, toWorkspacePresetResponse(&row, editable))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

func defaultTeamWorkspacePresets() []defaultTeamWorkspacePreset {
	return []defaultTeamWorkspacePreset{
		{
			Name:        "系统·资产排障工作台",
			SortOrder:   10,
			Recommended: true,
			Tabs: []workspaceTab{
				{Path: "/asset/overview", Pinned: true},
				{Path: "/asset/ops", Pinned: true},
				{Path: "/host"},
				{Path: "/cmdb/network-devices"},
				{Path: "/firewall"},
				{Path: "/jump/sessions"},
			},
		},
		{
			Name:        "系统·容器值班工作台",
			SortOrder:   20,
			Recommended: true,
			Tabs: []workspaceTab{
				{Path: "/k8s/overview", Pinned: true},
				{Path: "/k8s/workloads", Pinned: true},
				{Path: "/k8s/deployments"},
				{Path: "/k8s/pods"},
				{Path: "/k8s/events"},
				{Path: "/k8s/services"},
			},
		},
		{
			Name:        "系统·监控告警工作台",
			SortOrder:   30,
			Recommended: true,
			Tabs: []workspaceTab{
				{Path: "/monitor/center", Pinned: true},
				{Path: "/domain/center", Pinned: true},
				{Path: "/alert/events"},
				{Path: "/alert/rules"},
				{Path: "/notify/channels"},
				{Path: "/domain/ssl"},
			},
		},
		{
			Name:        "系统·交付发布工作台",
			SortOrder:   40,
			Recommended: true,
			Tabs: []workspaceTab{
				{Path: "/delivery/center", Pinned: true},
				{Path: "/cicd/pipelines", Pinned: true},
				{Path: "/cicd/executions"},
				{Path: "/cicd/releases"},
				{Path: "/workorder/tickets"},
				{Path: "/application"},
			},
		},
		{
			Name:        "系统·协同处置工作台",
			SortOrder:   50,
			Recommended: true,
			Tabs: []workspaceTab{
				{Path: "/collab/center", Pinned: true},
				{Path: "/ai", Pinned: true},
				{Path: "/terminal"},
				{Path: "/oncall/schedule"},
				{Path: "/workflow/designer"},
				{Path: "/workorder/tickets"},
			},
		},
	}
}

func (s *Server) ensureDefaultTeamWorkspacePresets() error {
	if s == nil || s.core == nil || s.core.DB == nil {
		return nil
	}
	s.workspacePresetInitMu.Lock()
	defer s.workspacePresetInitMu.Unlock()
	if s.workspacePresetInitDone {
		return nil
	}

	defaults := defaultTeamWorkspacePresets()
	for _, item := range defaults {
		tabs, err := normalizeWorkspaceTabs(item.Tabs)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(tabs)
		row := workspacePresetRecord{
			Name:        item.Name,
			Scope:       workspacePresetScopeTeam,
			OwnerID:     "system",
			OwnerName:   "系统预置",
			Tabs:        string(payload),
			Version:     1,
			Recommended: item.Recommended,
			SortOrder:   item.SortOrder,
		}
		if err := s.core.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "scope"},
				{Name: "owner_id"},
				{Name: "name"},
			},
			DoNothing: true,
		}).Create(&row).Error; err != nil {
			return err
		}
	}
	s.workspacePresetInitDone = true
	return nil
}

func (s *Server) ensureWorkspacePresetUniqueConstraint() error {
	if s == nil || s.core == nil || s.core.DB == nil {
		return nil
	}

	// purge historically soft-deleted rows so uniqueness can be enforced on active rows
	if err := s.core.DB.Unscoped().
		Where("deleted_at IS NOT NULL").
		Delete(&workspacePresetRecord{}).Error; err != nil {
		return err
	}

	var rows []workspacePresetRecord
	if err := s.core.DB.
		Order("updated_at desc").
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(rows))
	duplicateIDs := make([]string, 0)
	for _, row := range rows {
		key := row.Scope + "\x00" + row.OwnerID + "\x00" + row.Name
		if _, ok := seen[key]; ok {
			duplicateIDs = append(duplicateIDs, row.ID)
			continue
		}
		seen[key] = struct{}{}
	}
	if len(duplicateIDs) > 0 {
		if err := s.core.DB.Unscoped().
			Where("id IN ?", duplicateIDs).
			Delete(&workspacePresetRecord{}).Error; err != nil {
			return err
		}
	}

	if !s.core.DB.Migrator().HasIndex(&workspacePresetRecord{}, "uk_workspace_preset_owner_name") {
		if err := s.core.DB.Migrator().CreateIndex(&workspacePresetRecord{}, "uk_workspace_preset_owner_name"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) createWorkspacePreset(c *gin.Context) {
	var req workspacePresetUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	user, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}
	name, err := sanitizeWorkspacePresetName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	scope := normalizeWorkspaceScope(req.Scope)
	if scope == workspacePresetScopeTeam && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可创建团队模板"})
		return
	}

	tabs, err := normalizeWorkspaceTabs(req.Tabs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	payload, _ := json.Marshal(tabs)

	row := workspacePresetRecord{
		Name:      name,
		Scope:     scope,
		OwnerID:   user.ID,
		OwnerName: firstNonEmpty(user.Nickname, user.Username, "unknown"),
		Tabs:      string(payload),
		Version:   1,
		SortOrder: int(time.Now().Unix()),
	}
	if isAdmin && scope == workspacePresetScopeTeam && req.Recommended != nil {
		row.Recommended = *req.Recommended
	}
	if isAdmin && scope == workspacePresetScopeTeam && req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}

	if err := s.core.DB.Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建模板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toWorkspacePresetResponse(&row, true)})
}

func (s *Server) updateWorkspacePreset(c *gin.Context) {
	id := c.Param("id")
	var req workspacePresetUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	user, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}
	name, err := sanitizeWorkspacePresetName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	var row workspacePresetRecord
	if err := s.core.DB.First(&row, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}
	if !isAdmin && row.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限修改该模板"})
		return
	}

	scope := normalizeWorkspaceScope(req.Scope)
	if scope == workspacePresetScopeTeam && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可设置团队模板"})
		return
	}

	tabs, err := normalizeWorkspaceTabs(req.Tabs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	payload, _ := json.Marshal(tabs)
	updates := map[string]interface{}{
		"name":    name,
		"scope":   scope,
		"tabs":    string(payload),
		"version": row.Version + 1,
	}
	if scope == workspacePresetScopeTeam && isAdmin {
		if req.Recommended != nil {
			updates["recommended"] = *req.Recommended
		}
		if req.SortOrder != nil {
			updates["sort_order"] = *req.SortOrder
		}
	}
	if err := s.core.DB.Model(&row).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新模板失败"})
		return
	}
	if err := s.core.DB.First(&row, "id = ?", id).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": toWorkspacePresetResponse(&row, true)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (s *Server) deleteWorkspacePreset(c *gin.Context) {
	id := c.Param("id")
	user, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}
	var row workspacePresetRecord
	if err := s.core.DB.First(&row, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}
	if !isAdmin && row.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限删除该模板"})
		return
	}
	if err := s.core.DB.Unscoped().Delete(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除模板失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (s *Server) reorderWorkspacePresets(c *gin.Context) {
	var req workspacePresetReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	_, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可排序团队模板"})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "模板列表不能为空"})
		return
	}
	tx := s.core.DB.Begin()
	for idx, item := range req.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "存在无效模板ID"})
			return
		}
		if err := tx.Model(&workspacePresetRecord{}).
			Where("id = ? AND scope = ?", id, workspacePresetScopeTeam).
			Updates(map[string]interface{}{
				"sort_order": (idx + 1) * 10,
				"updated_at": time.Now(),
			}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "排序保存失败"})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "排序保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "排序已更新"})
}

func (s *Server) markWorkspacePresetUsed(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "模板ID不能为空"})
		return
	}
	user, isAdmin, err := s.currentAuthUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户失败"})
		return
	}
	var row workspacePresetRecord
	if err := s.core.DB.First(&row, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}
	if !isAdmin && row.Scope != workspacePresetScopeTeam && row.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问该模板"})
		return
	}
	now := time.Now()
	if err := s.core.DB.Model(&row).Updates(map[string]interface{}{
		"use_count":         gorm.Expr("use_count + ?", 1),
		"last_used_by_id":   user.ID,
		"last_used_by_name": firstNonEmpty(user.Nickname, user.Username, "unknown"),
		"last_used_at":      &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "记录模板使用失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录成功"})
}

func toWorkspacePresetResponse(row *workspacePresetRecord, editable bool) workspacePresetResponse {
	tabs := make([]workspaceTab, 0)
	if row != nil && strings.TrimSpace(row.Tabs) != "" {
		_ = json.Unmarshal([]byte(row.Tabs), &tabs)
	}
	return workspacePresetResponse{
		ID:             row.ID,
		Name:           row.Name,
		Scope:          row.Scope,
		OwnerID:        row.OwnerID,
		OwnerName:      row.OwnerName,
		Editable:       editable,
		Recommended:    row.Recommended,
		SortOrder:      row.SortOrder,
		UseCount:       row.UseCount,
		LastUsedByID:   row.LastUsedByID,
		LastUsedByName: row.LastUsedByName,
		LastUsedAt:     row.LastUsedAt,
		Tabs:           tabs,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func normalizeWorkspaceScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case workspacePresetScopeTeam:
		return workspacePresetScopeTeam
	default:
		return workspacePresetScopePrivate
	}
}

func normalizeWorkspaceTabs(tabs []workspaceTab) ([]workspaceTab, error) {
	if len(tabs) == 0 {
		return nil, errors.New("页签不能为空")
	}
	seen := make(map[string]struct{})
	result := make([]workspaceTab, 0, len(tabs))
	for _, tab := range tabs {
		path := strings.TrimSpace(tab.Path)
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, workspaceTab{Path: path, Pinned: tab.Pinned})
		if len(result) >= 18 {
			break
		}
	}
	if len(result) == 0 {
		return nil, errors.New("页签无效")
	}
	return result, nil
}

func sanitizeWorkspacePresetName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(name)
	if length < 2 || length > 24 {
		return "", errors.New("模板名称长度需为2-24个字符")
	}
	return name, nil
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
