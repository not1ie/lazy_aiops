package jump

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	"github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	"github.com/lazyautoops/lazy-auto-ops/plugins/terminal"
	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

type JumpHandler struct {
	db        *gorm.DB
	secretKey string
}

type syncStat struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

type JumpPolicyView struct {
	JumpPermissionPolicy
	AssetName   string `json:"asset_name"`
	AccountName string `json:"account_name"`
}

type commandRiskDecision struct {
	Matched      bool
	WhitelistHit bool
	RuleID       string
	RuleName     string
	Severity     string
	Action       string
	Reason       string
	RuleIDs      []string
	RuleNames    []string
	MatchedRules []commandRuleHit
}

type commandRuleHit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RuleKind string `json:"rule_kind"`
	Action   string `json:"action"`
	Severity string `json:"severity"`
	Priority int    `json:"priority"`
	Pattern  string `json:"pattern"`
}

func NewJumpHandler(db *gorm.DB, secretKey string) *JumpHandler {
	return &JumpHandler{
		db:        db,
		secretKey: secretKey,
	}
}

func (h *JumpHandler) ListAssets(c *gin.Context) {
	var items []JumpAsset
	query := h.db.Model(&JumpAsset{}).Order("created_at DESC")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR address LIKE ? OR source_ref LIKE ?", like, like, like)
	}
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("source = ?", source)
	}
	if assetType := strings.TrimSpace(c.Query("asset_type")); assetType != "" {
		query = query.Where("asset_type = ?", assetType)
	}
	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		if b, err := strconv.ParseBool(enabled); err == nil {
			query = query.Where("enabled = ?", b)
		}
	}

	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

func (h *JumpHandler) CreateAsset(c *gin.Context) {
	var req JumpAsset
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Protocol) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "名称和协议必填"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Protocol = strings.TrimSpace(req.Protocol)
	if req.AssetType == "" {
		req.AssetType = inferAssetType(req.Protocol)
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	if req.Port == 0 {
		req.Port = inferPort(req.Protocol)
	}
	if req.SourceRef != "" {
		var count int64
		h.db.Model(&JumpAsset{}).Where("source = ? AND source_ref = ?", req.Source, req.SourceRef).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该来源资产已存在，请勿重复创建"})
			return
		}
	}

	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": req})
}

func (h *JumpHandler) GetAsset(c *gin.Context) {
	id := c.Param("id")
	var item JumpAsset
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "资产不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (h *JumpHandler) UpdateAsset(c *gin.Context) {
	id := c.Param("id")
	var item JumpAsset
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "资产不存在"})
		return
	}

	updates := map[string]interface{}{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "deleted_at")
	if protocol, ok := updates["protocol"].(string); ok && strings.TrimSpace(protocol) != "" {
		if _, exists := updates["port"]; !exists {
			updates["port"] = inferPort(strings.TrimSpace(protocol))
		}
		if _, exists := updates["asset_type"]; !exists {
			updates["asset_type"] = inferAssetType(strings.TrimSpace(protocol))
		}
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无有效更新字段"})
		return
	}

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.First(&item, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (h *JumpHandler) DeleteAsset(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&JumpAsset{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.Where("asset_id = ?", id).Delete(&JumpPermissionPolicy{}).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *JumpHandler) ListAccounts(c *gin.Context) {
	var items []JumpAccount
	query := h.db.Model(&JumpAccount{}).Order("created_at DESC")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR username LIKE ?", like, like)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		if b, err := strconv.ParseBool(enabled); err == nil {
			query = query.Where("enabled = ?", b)
		}
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]SafeJumpAccount, 0, len(items))
	for i := range items {
		result = append(result, toSafeAccount(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *JumpHandler) CreateAccount(c *gin.Context) {
	var req JumpAccount
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Username) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "账号名称和登录名必填"})
		return
	}
	if req.AuthType == "" {
		req.AuthType = "password"
	}
	if strings.TrimSpace(req.Secret) != "" {
		enc, err := encryptJumpSecret(h.secretKey, strings.TrimSpace(req.Secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "账号密钥加密失败"})
			return
		}
		req.Secret = enc
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toSafeAccount(&req)})
}

func (h *JumpHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	var item JumpAccount
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toSafeAccount(&item)})
}

func (h *JumpHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var item JumpAccount
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}

	updates := map[string]interface{}{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "deleted_at")
	if secret, ok := updates["secret"].(string); ok && strings.TrimSpace(secret) == "" {
		delete(updates, "secret")
	}
	if secret, ok := updates["secret"].(string); ok && strings.TrimSpace(secret) != "" {
		enc, err := encryptJumpSecret(h.secretKey, strings.TrimSpace(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "账号密钥加密失败"})
			return
		}
		updates["secret"] = enc
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无有效更新字段"})
		return
	}

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.First(&item, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toSafeAccount(&item)})
}

func (h *JumpHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&JumpAccount{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.Where("account_id = ?", id).Delete(&JumpPermissionPolicy{}).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *JumpHandler) ListPolicies(c *gin.Context) {
	var items []JumpPermissionPolicy
	query := h.db.Model(&JumpPermissionPolicy{}).Order("created_at DESC")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if v, err := strconv.Atoi(status); err == nil {
			query = query.Where("status = ?", v)
		}
	}
	if assetID := strings.TrimSpace(c.Query("asset_id")); assetID != "" {
		query = query.Where("asset_id = ?", assetID)
	}
	if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if roleCode := strings.TrimSpace(c.Query("role_code")); roleCode != "" {
		query = query.Where("role_code = ?", roleCode)
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": h.attachPolicyNames(items)})
}

func (h *JumpHandler) CreatePolicy(c *gin.Context) {
	var req JumpPermissionPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Name == "" || req.AssetID == "" || req.AccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "策略名、资产和账号必填"})
		return
	}
	if req.UserID == "" && req.RoleCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户ID和角色编码至少填写一个"})
		return
	}
	if req.Protocol == "" {
		var asset JumpAsset
		if err := h.db.First(&asset, "id = ?", req.AssetID).Error; err == nil {
			req.Protocol = asset.Protocol
		}
	}
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	if req.Status == 0 {
		req.Status = 1
	}
	if msg := validatePolicyPayload(req.TimeWindowStart, req.TimeWindowEnd, req.MaxDurationSec, req.ConcurrentLimit); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": msg})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": req})
}

func (h *JumpHandler) GetPolicy(c *gin.Context) {
	id := c.Param("id")
	var item JumpPermissionPolicy
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "策略不存在"})
		return
	}
	views := h.attachPolicyNames([]JumpPermissionPolicy{item})
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": views[0]})
}

func (h *JumpHandler) UpdatePolicy(c *gin.Context) {
	id := c.Param("id")
	var item JumpPermissionPolicy
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "策略不存在"})
		return
	}

	updates := map[string]interface{}{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "deleted_at")
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无有效更新字段"})
		return
	}
	if v, ok := updates["protocol"].(string); ok {
		updates["protocol"] = strings.ToLower(strings.TrimSpace(v))
	}
	timeStart := item.TimeWindowStart
	if v, ok := updates["time_window_start"].(string); ok {
		timeStart = strings.TrimSpace(v)
		updates["time_window_start"] = timeStart
	}
	timeEnd := item.TimeWindowEnd
	if v, ok := updates["time_window_end"].(string); ok {
		timeEnd = strings.TrimSpace(v)
		updates["time_window_end"] = timeEnd
	}
	maxDuration := item.MaxDurationSec
	if _, exists := updates["max_duration_sec"]; exists {
		v, ok := intFromAny(updates["max_duration_sec"])
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "max_duration_sec 必须是数字"})
			return
		}
		maxDuration = v
		updates["max_duration_sec"] = v
	}
	concurrentLimit := item.ConcurrentLimit
	if _, exists := updates["concurrent_limit"]; exists {
		v, ok := intFromAny(updates["concurrent_limit"])
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "concurrent_limit 必须是数字"})
			return
		}
		concurrentLimit = v
		updates["concurrent_limit"] = v
	}
	if msg := validatePolicyPayload(timeStart, timeEnd, maxDuration, concurrentLimit); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": msg})
		return
	}

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.First(&item, "id = ?", id)
	views := h.attachPolicyNames([]JumpPermissionPolicy{item})
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": views[0]})
}

func (h *JumpHandler) DeletePolicy(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&JumpPermissionPolicy{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *JumpHandler) ListCommandRules(c *gin.Context) {
	var items []JumpCommandRule
	query := h.db.Model(&JumpCommandRule{}).Order("priority DESC, created_at DESC")
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		if b, err := strconv.ParseBool(enabled); err == nil {
			query = query.Where("enabled = ?", b)
		}
	}
	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		query = query.Where("protocol = ? OR protocol = ''", protocol)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR pattern LIKE ?", like, like)
	}
	if ruleKind := strings.TrimSpace(c.Query("rule_kind")); ruleKind != "" {
		query = query.Where("rule_kind = ?", normalizeRuleKind(ruleKind))
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

func (h *JumpHandler) GetCommandRuleStats(c *gin.Context) {
	days := 7
	if d, err := strconv.Atoi(strings.TrimSpace(c.Query("days"))); err == nil && d >= 1 && d <= 60 {
		days = d
	}
	startAt := time.Now().Add(-time.Duration(days-1) * 24 * time.Hour).Truncate(24 * time.Hour)

	var totalRules int64
	var enabledRules int64
	var allowRules int64
	_ = h.db.Model(&JumpCommandRule{}).Count(&totalRules).Error
	_ = h.db.Model(&JumpCommandRule{}).Where("enabled = ?", true).Count(&enabledRules).Error
	_ = h.db.Model(&JumpCommandRule{}).Where("enabled = ? AND rule_kind = ?", true, "allow").Count(&allowRules).Error

	var totalCommands int64
	var riskyCommands int64
	var blockedCommands int64
	var whitelistPass int64
	var alertsLinked int64
	var windowCommands int64
	_ = h.db.Model(&JumpCommand{}).Count(&totalCommands).Error
	_ = h.db.Model(&JumpCommand{}).Where("risk_level <> ''").Count(&riskyCommands).Error
	_ = h.db.Model(&JumpCommand{}).Where("blocked = ?", true).Count(&blockedCommands).Error
	_ = h.db.Model(&JumpCommand{}).Where("whitelist_hit = ?", true).Count(&whitelistPass).Error
	_ = h.db.Model(&JumpCommand{}).Where("alert_id <> ''").Count(&alertsLinked).Error
	_ = h.db.Model(&JumpCommand{}).Where("executed_at >= ?", startAt).Count(&windowCommands).Error

	type dayRow struct {
		Day       string `json:"day"`
		Total     int    `json:"total"`
		Risky     int    `json:"risky"`
		Blocked   int    `json:"blocked"`
		AllowPass int    `json:"allow_pass"`
	}
	rawRows := make([]dayRow, 0)
	_ = h.db.Raw(
		`SELECT DATE(executed_at) AS day,
		        COUNT(1) AS total,
		        SUM(CASE WHEN risk_level <> '' THEN 1 ELSE 0 END) AS risky,
		        SUM(CASE WHEN blocked = 1 THEN 1 ELSE 0 END) AS blocked,
		        SUM(CASE WHEN whitelist_hit = 1 THEN 1 ELSE 0 END) AS allow_pass
		   FROM jump_commands
		  WHERE executed_at >= ?
		  GROUP BY DATE(executed_at)
		  ORDER BY day ASC`,
		startAt,
	).Scan(&rawRows).Error
	byDayMap := map[string]dayRow{}
	for i := range rawRows {
		byDayMap[rawRows[i].Day] = rawRows[i]
	}
	byDay := make([]dayRow, 0, days)
	for i := 0; i < days; i++ {
		d := startAt.Add(time.Duration(i) * 24 * time.Hour)
		key := d.Format("2006-01-02")
		if row, ok := byDayMap[key]; ok {
			byDay = append(byDay, row)
		} else {
			byDay = append(byDay, dayRow{Day: key})
		}
	}

	type topItem struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	topUsers := make([]topItem, 0)
	topAssets := make([]topItem, 0)
	_ = h.db.Raw(
		`SELECT username AS name, COUNT(1) AS count
		   FROM jump_commands
		  WHERE executed_at >= ? AND risk_level <> ''
		  GROUP BY username
		  ORDER BY count DESC
		  LIMIT 10`,
		startAt,
	).Scan(&topUsers).Error
	_ = h.db.Raw(
		`SELECT COALESCE(s.asset_name, '-') AS name, COUNT(1) AS count
		   FROM jump_commands c
		   LEFT JOIN jump_sessions s ON s.id = c.session_id
		  WHERE c.executed_at >= ? AND c.risk_level <> ''
		  GROUP BY COALESCE(s.asset_name, '-')
		  ORDER BY count DESC
		  LIMIT 10`,
		startAt,
	).Scan(&topAssets).Error

	var rows []JumpCommand
	_ = h.db.Select("matched_rules").Where("matched_rules <> ''").Where("executed_at >= ?", startAt).Find(&rows).Error
	hits := map[string]int{}
	for i := range rows {
		raw := strings.TrimSpace(rows[i].MatchedRules)
		if raw == "" {
			continue
		}
		var parsed []commandRuleHit
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			continue
		}
		for _, one := range parsed {
			name := strings.TrimSpace(one.Name)
			if name == "" {
				continue
			}
			hits[name]++
		}
	}
	type hitItem struct {
		Rule  string `json:"rule"`
		Count int    `json:"count"`
	}
	top := make([]hitItem, 0, len(hits))
	for name, count := range hits {
		top = append(top, hitItem{Rule: name, Count: count})
	}
	for i := 0; i < len(top); i++ {
		for j := i + 1; j < len(top); j++ {
			if top[j].Count > top[i].Count {
				top[i], top[j] = top[j], top[i]
			}
		}
	}
	if len(top) > 10 {
		top = top[:10]
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"rules_total":      totalRules,
		"rules_enabled":    enabledRules,
		"rules_allow":      allowRules,
		"commands_total":   totalCommands,
		"commands_window":  windowCommands,
		"window_days":      days,
		"commands_risky":   riskyCommands,
		"commands_blocked": blockedCommands,
		"commands_allow":   whitelistPass,
		"alerts_linked":    alertsLinked,
		"top_rules":        top,
		"top_users":        topUsers,
		"top_assets":       topAssets,
		"trend_by_day":     byDay,
	}})
}

func (h *JumpHandler) BatchCommandRules(c *gin.Context) {
	var req struct {
		Action string   `json:"action"`
		IDs    []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请至少选择一条规则"})
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "enable":
		if err := h.db.Model(&JumpCommandRule{}).Where("id IN ?", req.IDs).Update("enabled", true).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "disable":
		if err := h.db.Model(&JumpCommandRule{}).Where("id IN ?", req.IDs).Update("enabled", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "delete":
		if err := h.db.Where("id IN ?", req.IDs).Delete(&JumpCommandRule{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的批量动作"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "批量操作成功"})
}

func (h *JumpHandler) CreateCommandRule(c *gin.Context) {
	var req JumpCommandRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Pattern = strings.TrimSpace(req.Pattern)
	req.Protocol = strings.ToLower(strings.TrimSpace(req.Protocol))
	req.RuleKind = normalizeRuleKind(req.RuleKind)
	req.MatchType = normalizeRuleMatchType(req.MatchType)
	req.Severity = normalizeRuleSeverity(req.Severity)
	req.Action = normalizeRuleAction(req.Action)
	if req.Name == "" || req.Pattern == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则名称和匹配表达式必填"})
		return
	}
	if req.Priority == 0 {
		req.Priority = 100
	}
	if req.RuleKind == "allow" {
		req.Severity = "info"
		req.Action = "alert"
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": req})
}

func (h *JumpHandler) GetCommandRule(c *gin.Context) {
	id := c.Param("id")
	var item JumpCommandRule
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (h *JumpHandler) UpdateCommandRule(c *gin.Context) {
	id := c.Param("id")
	var item JumpCommandRule
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	updates := map[string]interface{}{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "deleted_at")

	if v, ok := updates["name"].(string); ok {
		updates["name"] = strings.TrimSpace(v)
	}
	if v, ok := updates["pattern"].(string); ok {
		updates["pattern"] = strings.TrimSpace(v)
	}
	if v, ok := updates["protocol"].(string); ok {
		updates["protocol"] = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := updates["match_type"].(string); ok {
		updates["match_type"] = normalizeRuleMatchType(v)
	}
	if v, ok := updates["rule_kind"].(string); ok {
		updates["rule_kind"] = normalizeRuleKind(v)
	}
	if v, ok := updates["severity"].(string); ok {
		updates["severity"] = normalizeRuleSeverity(v)
	}
	if v, ok := updates["action"].(string); ok {
		updates["action"] = normalizeRuleAction(v)
	}
	if kind, ok := updates["rule_kind"].(string); ok && kind == "allow" {
		updates["severity"] = "info"
		updates["action"] = "alert"
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无有效更新字段"})
		return
	}

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.First(&item, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (h *JumpHandler) DeleteCommandRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&JumpCommandRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *JumpHandler) ListSessions(c *gin.Context) {
	var items []JumpSession
	query := h.db.Model(&JumpSession{}).Order("started_at DESC")

	if roleCode := c.GetString("role_code"); roleCode != "admin" {
		query = query.Where("user_id = ?", c.GetString("user_id"))
	}
	if assetID := strings.TrimSpace(c.Query("asset_id")); assetID != "" {
		query = query.Where("asset_id = ?", assetID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

func (h *JumpHandler) StartSession(c *gin.Context) {
	var req struct {
		AssetID   string `json:"asset_id" binding:"required"`
		AccountID string `json:"account_id" binding:"required"`
		Protocol  string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var asset JumpAsset
	if err := h.db.First(&asset, "id = ?", req.AssetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "资产不存在"})
		return
	}
	if !asset.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "资产已禁用"})
		return
	}

	var account JumpAccount
	if err := h.db.First(&account, "id = ?", req.AccountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}
	if !account.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "账号已禁用"})
		return
	}

	userID := c.GetString("user_id")
	roleCode := c.GetString("role_code")
	protocol := req.Protocol
	if protocol == "" {
		protocol = asset.Protocol
	}
	policy, err := h.resolveAccessPolicy(userID, roleCode, asset.ID, account.ID, protocol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if policy == nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前用户无该资产访问授权"})
		return
	}

	now := time.Now()
	if msg := validatePolicyTimeWindow(policy, now); msg != "" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": msg})
		return
	}
	if policy.ConcurrentLimit > 0 {
		var activeCount int64
		if err := h.db.Model(&JumpSession{}).
			Where("user_id = ?", userID).
			Where("policy_id = ?", policy.ID).
			Where("status IN ?", []string{"pending_approval", "active"}).
			Count(&activeCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if int(activeCount) >= policy.ConcurrentLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "已达到策略并发会话上限"})
			return
		}
	}

	status := "active"
	needApprove := false
	if policy.RequireApprove && roleCode != "admin" {
		status = "pending_approval"
		needApprove = true
	}
	session := JumpSession{
		SessionNo:    fmt.Sprintf("JMP-%s-%s", now.Format("20060102150405"), strings.ToUpper(uuid.NewString()[:8])),
		UserID:       userID,
		Username:     c.GetString("username"),
		RoleCode:     roleCode,
		AssetID:      asset.ID,
		AssetName:    asset.Name,
		AccountID:    account.ID,
		AccountName:  account.Name,
		PolicyID:     policy.ID,
		Protocol:     protocol,
		SourceIP:     c.ClientIP(),
		Status:       status,
		StartedAt:    now,
		CommandCount: 0,
	}
	if session.Status == "pending_approval" {
		session.CloseReason = "waiting_approval"
	}

	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"session":      session,
		"need_approve": needApprove,
	}})
}

func (h *JumpHandler) ConnectSession(c *gin.Context) {
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && session.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限连接该会话"})
		return
	}
	if session.Status != "active" {
		if session.Status == "pending_approval" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话待审批，暂不可连接"})
			return
		}
		if session.Status == "rejected" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已被拒绝，无法连接"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已关闭，无法连接"})
		return
	}

	var asset JumpAsset
	if err := h.db.First(&asset, "id = ?", session.AssetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话关联资产不存在"})
		return
	}
	var account JumpAccount
	if err := h.db.First(&account, "id = ?", session.AccountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话关联账号不存在"})
		return
	}
	policy := h.loadSessionPolicy(&session, &asset, &account)
	now := time.Now()
	if msg := validatePolicyTimeWindow(policy, now); msg != "" {
		_ = h.closeSessionWithStatus(&session, "blocked", msg, c.GetString("username"), now)
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": msg})
		return
	}
	if policy != nil && policy.MaxDurationSec > 0 && !session.StartedAt.IsZero() {
		if int(now.Sub(session.StartedAt).Seconds()) >= policy.MaxDurationSec {
			_ = h.closeSessionWithStatus(&session, "closed", "会话已达到策略最大时长，需重新发起", c.GetString("username"), now)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已达到策略最大时长，需重新发起"})
			return
		}
	}

	protocol := strings.ToLower(strings.TrimSpace(session.Protocol))
	if protocol == "" {
		protocol = strings.ToLower(strings.TrimSpace(asset.Protocol))
	}
	if protocol == "" {
		protocol = "ssh"
	}

	switch protocol {
	case "ssh", "docker":
		termSession, err := h.createTerminalRelaySession(&session, &asset, &account, c.GetString("username"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		openURL := "/terminal?session_id=" + url.QueryEscape(termSession.ID)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"mode":             "terminal",
			"relay_session_id": termSession.ID,
			"relay_plugin":     "terminal",
			"open_url":         openURL,
			"terminal_session": termSession,
			"jump_session_id":  session.ID,
			"jump_session_no":  session.SessionNo,
			"protocol":         protocol,
		}})
		return
	case "k8s":
		clusterID := h.resolveClusterIDForK8sAsset(&asset)
		openURL := "/k8s/terminal"
		if clusterID != "" {
			openURL += "?clusterId=" + url.QueryEscape(clusterID)
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"mode":            "k8s",
			"relay_plugin":    "k8s",
			"open_url":        openURL,
			"jump_session_id": session.ID,
			"jump_session_no": session.SessionNo,
			"protocol":        protocol,
		}})
		return
	case "mysql", "postgres":
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"mode":            "sql",
			"relay_plugin":    "jump",
			"open_url":        "/jump/sessions",
			"jump_session_id": session.ID,
			"jump_session_no": session.SessionNo,
			"protocol":        protocol,
			"execute_api":     fmt.Sprintf("/api/v1/jump/sessions/%s/sql/execute", session.ID),
			"note":            "可在会话审计页进入 SQL 控制台执行并审计 SQL 语句",
		}})
		return
	default:
		c.JSON(http.StatusNotImplemented, gin.H{"code": 501, "message": "当前协议尚未接入在线代理，请先使用命令审计模式"})
		return
	}
}

func (h *JumpHandler) ApproveSession(c *gin.Context) {
	if c.GetString("role_code") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可审批"})
		return
	}
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.Status != "pending_approval" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "当前会话不是待审批状态"})
		return
	}
	now := time.Now()
	if err := h.db.Model(&session).Updates(map[string]interface{}{
		"status":       "active",
		"approved_by":  c.GetString("username"),
		"approved_at":  now,
		"close_reason": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&session, "id = ?", sessionID).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func (h *JumpHandler) RejectSession(c *gin.Context) {
	if c.GetString("role_code") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可审批"})
		return
	}
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.Status != "pending_approval" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "当前会话不是待审批状态"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "审批拒绝"
	}
	now := time.Now()
	if err := h.closeSessionWithStatus(&session, "rejected", fmt.Sprintf("rejected_by:%s %s", c.GetString("username"), reason), c.GetString("username"), now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&session, "id = ?", sessionID).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func (h *JumpHandler) GetSession(c *gin.Context) {
	id := c.Param("id")
	var item JumpSession
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && item.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限查看该会话"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": item})
}

func (h *JumpHandler) RecordCommand(c *gin.Context) {
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && session.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限写入该会话"})
		return
	}
	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已关闭，无法记录命令"})
		return
	}

	var req struct {
		Command       string `json:"command" binding:"required"`
		ResultCode    int    `json:"result_code"`
		OutputSnippet string `json:"output_snippet"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	policy := h.loadSessionPolicy(&session, nil, nil)
	now := time.Now()
	if policy != nil && policy.MaxDurationSec > 0 && !session.StartedAt.IsZero() {
		if int(now.Sub(session.StartedAt).Seconds()) >= policy.MaxDurationSec {
			_ = h.closeSessionWithStatus(&session, "closed", "会话已达到策略最大时长，需重新发起", c.GetString("username"), now)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已达到策略最大时长，需重新发起"})
			return
		}
	}

	decision := h.evaluateCommandRisk(session.Protocol, req.Command)
	alertID := ""
	if decision.Matched && !decision.WhitelistHit && (decision.Action == "alert" || decision.Action == "block") {
		alertID = h.createCommandRiskAlert(&session, decision, req.Command, now)
	}
	matchedRules, _ := json.Marshal(decision.MatchedRules)
	cmd := JumpCommand{
		SessionID:     session.ID,
		Username:      session.Username,
		CommandType:   "shell",
		Command:       req.Command,
		ResultCode:    req.ResultCode,
		OutputSnippet: req.OutputSnippet,
		RuleID:        strings.Join(decision.RuleIDs, ","),
		RuleName:      strings.Join(decision.RuleNames, ","),
		MatchedRules:  string(matchedRules),
		WhitelistHit:  decision.WhitelistHit,
		RiskLevel:     decision.Severity,
		RiskAction:    decision.Action,
		RiskReason:    decision.Reason,
		Blocked:       decision.Matched && !decision.WhitelistHit && decision.Action == "block",
		AlertID:       alertID,
		ExecutedAt:    now,
	}
	if err := h.db.Create(&cmd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if decision.Matched && !decision.WhitelistHit {
		_ = h.createRiskEvent(&session, &cmd, decision, now)
	}

	_ = h.db.Model(&JumpSession{}).Where("id = ?", session.ID).Updates(map[string]interface{}{
		"last_command_at": now,
		"command_count":   gorm.Expr("command_count + 1"),
	}).Error
	if cmd.Blocked {
		_ = h.closeSessionWithStatus(&session, "blocked", decision.Reason, c.GetString("username"), now)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cmd})
}

func (h *JumpHandler) ExecuteSQL(c *gin.Context) {
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && session.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限执行该会话SQL"})
		return
	}
	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话不可执行SQL"})
		return
	}

	var req struct {
		SQL      string `json:"sql" binding:"required"`
		Database string `json:"database"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	sqlText := strings.TrimSpace(req.SQL)
	if sqlText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SQL不能为空"})
		return
	}
	if strings.Count(sqlText, ";") > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "一次只允许执行一条SQL"})
		return
	}

	var asset JumpAsset
	if err := h.db.First(&asset, "id = ?", session.AssetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "关联资产不存在"})
		return
	}
	var account JumpAccount
	if err := h.db.First(&account, "id = ?", session.AccountID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "关联账号不存在"})
		return
	}

	now := time.Now()
	policy := h.loadSessionPolicy(&session, &asset, &account)
	if policy != nil && policy.MaxDurationSec > 0 && !session.StartedAt.IsZero() {
		if int(now.Sub(session.StartedAt).Seconds()) >= policy.MaxDurationSec {
			_ = h.closeSessionWithStatus(&session, "closed", "会话已达到策略最大时长，需重新发起", c.GetString("username"), now)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话已达到策略最大时长，需重新发起"})
			return
		}
	}

	decision := h.evaluateCommandRisk(session.Protocol, sqlText)
	alertID := ""
	if decision.Matched && !decision.WhitelistHit && (decision.Action == "alert" || decision.Action == "block") {
		alertID = h.createCommandRiskAlert(&session, decision, sqlText, now)
	}

	cmd := JumpCommand{
		SessionID:    session.ID,
		Username:     session.Username,
		CommandType:  "sql",
		Command:      sqlText,
		ResultCode:   0,
		RuleID:       strings.Join(decision.RuleIDs, ","),
		RuleName:     strings.Join(decision.RuleNames, ","),
		WhitelistHit: decision.WhitelistHit,
		RiskLevel:    decision.Severity,
		RiskAction:   decision.Action,
		RiskReason:   decision.Reason,
		Blocked:      decision.Matched && !decision.WhitelistHit && decision.Action == "block",
		AlertID:      alertID,
		ExecutedAt:   now,
	}
	matchedRules, _ := json.Marshal(decision.MatchedRules)
	cmd.MatchedRules = string(matchedRules)

	if cmd.Blocked {
		cmd.ResultCode = 1
		cmd.OutputSnippet = "命中阻断策略，SQL已拒绝执行"
		if err := h.db.Create(&cmd).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		_ = h.createRiskEvent(&session, &cmd, decision, now)
		_ = h.db.Model(&JumpSession{}).Where("id = ?", session.ID).Updates(map[string]interface{}{
			"last_command_at": now,
			"command_count":   gorm.Expr("command_count + 1"),
		}).Error
		_ = h.closeSessionWithStatus(&session, "blocked", decision.Reason, c.GetString("username"), now)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": cmd, "message": "命中阻断策略，SQL已拒绝执行"})
		return
	}

	startAt := time.Now()
	output, affectedRows, resultCode, execErr := h.executeSQLOnAsset(&session, &asset, &account, req.Database, sqlText)
	durationMS := int(time.Since(startAt).Milliseconds())
	cmd.ResultCode = resultCode
	cmd.OutputSnippet = output
	if execErr != nil {
		cmd.ResultCode = 1
		if cmd.RiskLevel == "" {
			cmd.RiskLevel = "warning"
		}
		if cmd.RiskAction == "" {
			cmd.RiskAction = "alert"
		}
		cmd.RiskReason = truncateJumpText(execErr.Error(), 512)
	}

	if err := h.db.Create(&cmd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if decision.Matched && !decision.WhitelistHit {
		_ = h.createRiskEvent(&session, &cmd, decision, now)
	}
	_ = h.db.Model(&JumpSession{}).Where("id = ?", session.ID).Updates(map[string]interface{}{
		"last_command_at": now,
		"command_count":   gorm.Expr("command_count + 1"),
	}).Error

	if execErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": execErr.Error(), "data": gin.H{
			"session_id":    session.ID,
			"command_id":    cmd.ID,
			"duration_ms":   durationMS,
			"affected_rows": affectedRows,
			"output":        output,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"session_id":    session.ID,
		"command_id":    cmd.ID,
		"duration_ms":   durationMS,
		"affected_rows": affectedRows,
		"output":        output,
	}})
}

func (h *JumpHandler) ListSessionCommands(c *gin.Context) {
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && session.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限查看该会话命令"})
		return
	}

	var cmds []JumpCommand
	query := h.db.Where("session_id = ?", sessionID).Order("executed_at DESC")
	if commandType := strings.TrimSpace(c.Query("command_type")); commandType != "" {
		query = query.Where("command_type = ?", strings.ToLower(commandType))
	}
	if limitText := strings.TrimSpace(c.Query("limit")); limitText != "" {
		if limit, err := strconv.Atoi(limitText); err == nil && limit > 0 && limit <= 1000 {
			query = query.Limit(limit)
		}
	}
	if err := query.Find(&cmds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cmds})
}

func (h *JumpHandler) DisconnectSession(c *gin.Context) {
	if c.GetString("role_code") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅管理员可断开会话"})
		return
	}
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.Status != "active" && session.Status != "pending_approval" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": session, "message": "会话已结束"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "管理员强制断开"
	}
	now := time.Now()
	status := "closed"
	if session.Status == "pending_approval" {
		status = "rejected"
	}
	if err := h.closeSessionWithStatus(&session, status, reason, c.GetString("username"), now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&session, "id = ?", sessionID).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func (h *JumpHandler) ListRiskEvents(c *gin.Context) {
	var items []JumpRiskEvent
	query := h.db.Model(&JumpRiskEvent{}).Order("fired_at DESC")
	if roleCode := c.GetString("role_code"); roleCode != "admin" {
		query = query.Where("username = ?", c.GetString("username"))
	}
	if sessionID := strings.TrimSpace(c.Query("session_id")); sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if severity := strings.TrimSpace(c.Query("severity")); severity != "" {
		query = query.Where("severity = ?", normalizeRuleSeverity(severity))
	}
	if eventType := strings.TrimSpace(c.Query("event_type")); eventType != "" {
		query = query.Where("event_type = ?", strings.ToLower(eventType))
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

func (h *JumpHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("id")
	var session JumpSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if c.GetString("role_code") != "admin" && session.UserID != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限关闭该会话"})
		return
	}
	if session.Status != "active" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": session, "message": "会话已关闭"})
		return
	}

	var req struct {
		CloseReason string `json:"close_reason"`
		Status      string `json:"status"`
	}
	_ = c.ShouldBindJSON(&req)

	now := time.Now()
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "closed"
	}
	if err := h.closeSessionWithStatus(&session, status, strings.TrimSpace(req.CloseReason), c.GetString("username"), now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.db.First(&session, "id = ?", sessionID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

func (h *JumpHandler) closeSessionWithStatus(session *JumpSession, status, reason, operator string, now time.Time) error {
	if session == nil {
		return nil
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "closed"
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "normal_close"
	}
	duration := 0
	if !session.StartedAt.IsZero() {
		duration = int(now.Sub(session.StartedAt).Seconds())
		if duration < 0 {
			duration = 0
		}
	}
	updates := map[string]interface{}{
		"status":          status,
		"ended_at":        now,
		"duration_sec":    duration,
		"close_reason":    reason,
		"disconnected_by": strings.TrimSpace(operator),
		"disconnected_at": now,
	}
	if err := h.db.Model(session).Where("status IN ?", []string{"pending_approval", "active"}).Updates(updates).Error; err != nil {
		return err
	}
	if strings.TrimSpace(session.RelaySessionID) != "" {
		_ = h.db.Model(&terminal.TerminalSession{}).Where("id = ?", session.RelaySessionID).Updates(map[string]interface{}{
			"status":   2,
			"ended_at": now,
		}).Error
	}
	return nil
}

func (h *JumpHandler) createRiskEvent(session *JumpSession, cmd *JumpCommand, decision commandRiskDecision, firedAt time.Time) error {
	if session == nil || cmd == nil || !decision.Matched || decision.WhitelistHit {
		return nil
	}
	eventType := "alert"
	if decision.Action == "block" {
		eventType = "blocked"
	}
	risk := JumpRiskEvent{
		SessionID:   session.ID,
		CommandID:   cmd.ID,
		AssetID:     session.AssetID,
		AssetName:   session.AssetName,
		Username:    session.Username,
		EventType:   eventType,
		Severity:    normalizeRuleSeverity(decision.Severity),
		Action:      normalizeRuleAction(decision.Action),
		RuleID:      decision.RuleID,
		RuleName:    decision.RuleName,
		Command:     cmd.Command,
		Description: truncateJumpText(decision.Reason, 512),
		FiredAt:     firedAt,
	}
	return h.db.Create(&risk).Error
}

func (h *JumpHandler) loadSessionPolicy(session *JumpSession, asset *JumpAsset, account *JumpAccount) *JumpPermissionPolicy {
	if session == nil {
		return nil
	}
	if strings.TrimSpace(session.PolicyID) != "" {
		var policy JumpPermissionPolicy
		if err := h.db.First(&policy, "id = ? AND status = ?", session.PolicyID, 1).Error; err == nil {
			return &policy
		}
	}
	if session.RoleCode == "admin" {
		return nil
	}
	assetID := session.AssetID
	accountID := session.AccountID
	protocol := session.Protocol
	if asset != nil && strings.TrimSpace(protocol) == "" {
		protocol = asset.Protocol
	}
	if account != nil {
		accountID = account.ID
	}
	policy, err := h.resolveAccessPolicy(session.UserID, session.RoleCode, assetID, accountID, protocol)
	if err != nil {
		return nil
	}
	if policy != nil && strings.TrimSpace(policy.ID) != "" && session.PolicyID == "" {
		_ = h.db.Model(&JumpSession{}).Where("id = ?", session.ID).Update("policy_id", policy.ID).Error
	}
	return policy
}

func validatePolicyTimeWindow(policy *JumpPermissionPolicy, now time.Time) string {
	if policy == nil {
		return ""
	}
	start := strings.TrimSpace(policy.TimeWindowStart)
	end := strings.TrimSpace(policy.TimeWindowEnd)
	if start == "" && end == "" {
		return ""
	}
	startMin, okStart := parseHHMM(start)
	endMin, okEnd := parseHHMM(end)
	if !okStart || !okEnd {
		return ""
	}
	currentMin := now.Hour()*60 + now.Minute()
	allowed := false
	if startMin <= endMin {
		allowed = currentMin >= startMin && currentMin <= endMin
	} else {
		allowed = currentMin >= startMin || currentMin <= endMin
	}
	if allowed {
		return ""
	}
	return fmt.Sprintf("当前时间不在授权时段（%s-%s）", start, end)
}

func parseHHMM(v string) (int, bool) {
	v = strings.TrimSpace(v)
	if len(v) != 5 {
		return 0, false
	}
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func (h *JumpHandler) executeSQLOnAsset(session *JumpSession, asset *JumpAsset, account *JumpAccount, databaseName, sqlText string) (string, int64, int, error) {
	if session == nil || asset == nil || account == nil {
		return "", 0, 1, errors.New("会话上下文不完整")
	}
	protocol := strings.ToLower(strings.TrimSpace(session.Protocol))
	if protocol == "" {
		protocol = strings.ToLower(strings.TrimSpace(asset.Protocol))
	}
	switch protocol {
	case "mysql":
		return h.executeMySQL(asset, account, databaseName, sqlText)
	case "postgres":
		return h.executePostgres(asset, account, databaseName, sqlText)
	default:
		return "", 0, 1, fmt.Errorf("协议 %s 不支持在线SQL执行", protocol)
	}
}

func (h *JumpHandler) executeMySQL(asset *JumpAsset, account *JumpAccount, databaseName, sqlText string) (string, int64, int, error) {
	host := strings.TrimSpace(asset.Address)
	if host == "" {
		return "", 0, 1, errors.New("资产地址为空")
	}
	port := asset.Port
	if port <= 0 {
		port = 3306
	}
	username := strings.TrimSpace(account.Username)
	if username == "" {
		return "", 0, 1, errors.New("账号用户名为空")
	}
	secret, err := decryptJumpSecret(h.secretKey, strings.TrimSpace(account.Secret))
	if err != nil {
		return "", 0, 1, errors.New("账号密钥解密失败")
	}
	password := strings.TrimSpace(secret)
	if password == "" {
		return "", 0, 1, errors.New("账号密码为空")
	}
	dbName := strings.TrimSpace(databaseName)
	if dbName == "" {
		dbName = strings.TrimSpace(asset.Namespace)
	}
	if dbName == "" {
		dbName = "mysql"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=false", username, password, host, port, dbName)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", 0, 1, err
	}
	defer conn.Close()
	conn.SetConnMaxLifetime(30 * time.Second)
	conn.SetMaxOpenConns(2)
	conn.SetMaxIdleConns(1)
	if err := conn.Ping(); err != nil {
		return "", 0, 1, err
	}

	normalized := strings.ToLower(strings.TrimSpace(sqlText))
	queryLike := strings.HasPrefix(normalized, "select") ||
		strings.HasPrefix(normalized, "show") ||
		strings.HasPrefix(normalized, "desc") ||
		strings.HasPrefix(normalized, "describe") ||
		strings.HasPrefix(normalized, "explain")
	if queryLike {
		return runGenericSQLQuery(conn, sqlText)
	}
	res, err := conn.Exec(sqlText)
	if err != nil {
		return "", 0, 1, err
	}
	affected, _ := res.RowsAffected()
	return fmt.Sprintf("ok, affected_rows=%d", affected), affected, 0, nil
}

func (h *JumpHandler) executePostgres(asset *JumpAsset, account *JumpAccount, databaseName, sqlText string) (string, int64, int, error) {
	host := strings.TrimSpace(asset.Address)
	if host == "" {
		return "", 0, 1, errors.New("资产地址为空")
	}
	port := asset.Port
	if port <= 0 {
		port = 5432
	}
	username := strings.TrimSpace(account.Username)
	if username == "" {
		return "", 0, 1, errors.New("账号用户名为空")
	}
	secret, err := decryptJumpSecret(h.secretKey, strings.TrimSpace(account.Secret))
	if err != nil {
		return "", 0, 1, errors.New("账号密钥解密失败")
	}
	password := strings.TrimSpace(secret)
	if password == "" {
		return "", 0, 1, errors.New("账号密码为空")
	}
	dbName := strings.TrimSpace(databaseName)
	if dbName == "" {
		dbName = strings.TrimSpace(asset.Namespace)
	}
	if dbName == "" {
		dbName = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, username, password, dbName)
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", 0, 1, err
	}
	defer conn.Close()
	conn.SetConnMaxLifetime(30 * time.Second)
	conn.SetMaxOpenConns(2)
	conn.SetMaxIdleConns(1)
	if err := conn.Ping(); err != nil {
		return "", 0, 1, err
	}

	normalized := strings.ToLower(strings.TrimSpace(sqlText))
	queryLike := strings.HasPrefix(normalized, "select") ||
		strings.HasPrefix(normalized, "show") ||
		strings.HasPrefix(normalized, "with") ||
		strings.HasPrefix(normalized, "explain")
	if queryLike {
		return runGenericSQLQuery(conn, sqlText)
	}
	res, err := conn.Exec(sqlText)
	if err != nil {
		return "", 0, 1, err
	}
	affected, _ := res.RowsAffected()
	return fmt.Sprintf("ok, affected_rows=%d", affected), affected, 0, nil
}

func runGenericSQLQuery(conn *sql.DB, sqlText string) (string, int64, int, error) {
	rows, err := conn.Query(sqlText)
	if err != nil {
		return "", 0, 1, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", 0, 1, err
	}
	resultRows := make([]map[string]string, 0, 20)
	count := 0
	for rows.Next() {
		values := make([]interface{}, len(cols))
		scanArgs := make([]interface{}, len(cols))
		for i := range values {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return "", 0, 1, err
		}
		row := map[string]string{}
		for i, col := range cols {
			row[col] = sqlValueToString(values[i])
		}
		if len(resultRows) < 20 {
			resultRows = append(resultRows, row)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return "", 0, 1, err
	}
	body, _ := json.Marshal(gin.H{
		"columns": cols,
		"rows":    resultRows,
		"count":   count,
		"trunc":   count > len(resultRows),
	})
	return string(body), int64(count), 0, nil
}

func sqlValueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	case string:
		return t
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}

func validatePolicyPayload(start, end string, maxDuration, concurrentLimit int) string {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if (start == "") != (end == "") {
		return "授权时段需同时填写开始和结束时间"
	}
	if start != "" || end != "" {
		if _, ok := parseHHMM(start); !ok {
			return "开始时间格式错误，应为 HH:MM"
		}
		if _, ok := parseHHMM(end); !ok {
			return "结束时间格式错误，应为 HH:MM"
		}
	}
	if maxDuration < 0 {
		return "最大会话时长不能小于 0"
	}
	if concurrentLimit < 0 {
		return "并发会话上限不能小于 0"
	}
	return ""
}

func intFromAny(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func (h *JumpHandler) createTerminalRelaySession(session *JumpSession, asset *JumpAsset, account *JumpAccount, operator string) (*terminal.TerminalSession, error) {
	host := strings.TrimSpace(asset.Address)
	if host == "" {
		return nil, errors.New("资产缺少地址，无法建立 SSH 代理")
	}
	port := asset.Port
	if port <= 0 {
		port = 22
	}
	username := strings.TrimSpace(account.Username)
	if username == "" {
		return nil, errors.New("账号缺少用户名，无法建立 SSH 代理")
	}

	relay := &terminal.TerminalSession{
		HostID:   asset.SourceRef,
		Host:     host,
		Port:     port,
		Username: username,
		UserID:   session.UserID,
		Operator: operator,
		Status:   0,
	}

	secret, err := decryptJumpSecret(h.secretKey, strings.TrimSpace(account.Secret))
	if err != nil {
		return nil, errors.New("账号凭据解密失败，请重新保存账号密钥")
	}
	secret = strings.TrimSpace(secret)
	switch strings.ToLower(strings.TrimSpace(account.AuthType)) {
	case "", "password", "token":
		relay.Password = secret
	case "key":
		relay.PrivateKey = secret
	default:
		relay.Password = secret
	}
	if relay.Password == "" && relay.PrivateKey == "" {
		return nil, errors.New("账号未配置可用凭据（密码/私钥）")
	}

	if err := h.db.Create(relay).Error; err != nil {
		return nil, err
	}
	_ = h.db.Model(&JumpSession{}).Where("id = ?", session.ID).Update("relay_session_id", relay.ID).Error
	return relay, nil
}

func (h *JumpHandler) resolveClusterIDForK8sAsset(asset *JumpAsset) string {
	if asset == nil {
		return ""
	}
	if asset.Source == "k8s_cluster" && strings.TrimSpace(asset.SourceRef) != "" {
		return asset.SourceRef
	}
	if strings.TrimSpace(asset.Cluster) == "" {
		return ""
	}
	var cluster k8s.Cluster
	if err := h.db.Where("name = ? OR display_name = ?", asset.Cluster, asset.Cluster).First(&cluster).Error; err == nil {
		return cluster.ID
	}
	return ""
}

func (h *JumpHandler) SyncFromCMDBHosts(c *gin.Context) {
	stat, err := h.syncFromCMDBHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stat, "message": fmt.Sprintf("同步完成，新增%d，更新%d", stat.Created, stat.Updated)})
}

func (h *JumpHandler) SyncFromK8sClusters(c *gin.Context) {
	stat, err := h.syncFromK8sClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stat, "message": fmt.Sprintf("同步完成，新增%d，更新%d", stat.Created, stat.Updated)})
}

func (h *JumpHandler) SyncFromDockerHosts(c *gin.Context) {
	stat, err := h.syncFromDockerHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stat, "message": fmt.Sprintf("同步完成，新增%d，更新%d", stat.Created, stat.Updated)})
}

func (h *JumpHandler) SyncAllAssets(c *gin.Context) {
	cmdbStat, err := h.syncFromCMDBHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "CMDB同步失败: " + err.Error()})
		return
	}
	k8sStat, err := h.syncFromK8sClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "K8s同步失败: " + err.Error()})
		return
	}
	dockerStat, err := h.syncFromDockerHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Docker同步失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"cmdb_hosts":   cmdbStat,
		"k8s_clusters": k8sStat,
		"docker_hosts": dockerStat,
	}})
}

func (h *JumpHandler) syncFromCMDBHosts() (syncStat, error) {
	var hosts []cmdb.Host
	if err := h.db.Find(&hosts).Error; err != nil {
		return syncStat{}, err
	}
	stat := syncStat{}
	for i := range hosts {
		host := hosts[i]
		name := strings.TrimSpace(host.Name)
		if name == "" {
			name = strings.TrimSpace(host.IP)
		}
		if name == "" {
			continue
		}
		asset := JumpAsset{
			Name:      name,
			AssetType: "host",
			Protocol:  "ssh",
			Address:   strings.TrimSpace(host.IP),
			Port:      host.Port,
			Source:    "cmdb_host",
			SourceRef: host.ID,
			Tags:      host.Tags,
			Enabled:   host.Status != 0,
		}
		if asset.Port == 0 {
			asset.Port = 22
		}
		if err := h.upsertAsset(asset, &stat); err != nil {
			return stat, err
		}
	}
	return stat, nil
}

func (h *JumpHandler) syncFromK8sClusters() (syncStat, error) {
	var clusters []k8s.Cluster
	if err := h.db.Find(&clusters).Error; err != nil {
		return syncStat{}, err
	}
	stat := syncStat{}
	for i := range clusters {
		cluster := clusters[i]
		name := strings.TrimSpace(cluster.DisplayName)
		if name == "" {
			name = strings.TrimSpace(cluster.Name)
		}
		if name == "" {
			continue
		}
		asset := JumpAsset{
			Name:      name,
			AssetType: "k8s",
			Protocol:  "k8s",
			Address:   strings.TrimSpace(cluster.APIServer),
			Port:      443,
			Cluster:   cluster.Name,
			Source:    "k8s_cluster",
			SourceRef: cluster.ID,
			Enabled:   cluster.Status == 1,
		}
		if err := h.upsertAsset(asset, &stat); err != nil {
			return stat, err
		}
	}
	return stat, nil
}

func (h *JumpHandler) syncFromDockerHosts() (syncStat, error) {
	var hosts []docker.DockerHost
	if err := h.db.Find(&hosts).Error; err != nil {
		return syncStat{}, err
	}
	stat := syncStat{}
	for i := range hosts {
		host := hosts[i]
		name := strings.TrimSpace(host.Name)
		address := ""
		port := 22
		if strings.TrimSpace(host.HostID) != "" {
			var cmdbHost cmdb.Host
			if err := h.db.First(&cmdbHost, "id = ?", host.HostID).Error; err == nil {
				if name == "" {
					name = cmdbHost.Name
				}
				address = cmdbHost.IP
				if cmdbHost.Port > 0 {
					port = cmdbHost.Port
				}
			}
		}
		if name == "" {
			name = host.ID
		}
		asset := JumpAsset{
			Name:      name,
			AssetType: "host",
			Protocol:  "docker",
			Address:   address,
			Port:      port,
			Source:    "docker_host",
			SourceRef: host.ID,
			Enabled:   strings.ToLower(strings.TrimSpace(host.Status)) != "offline",
		}
		if err := h.upsertAsset(asset, &stat); err != nil {
			return stat, err
		}
	}
	return stat, nil
}

func (h *JumpHandler) upsertAsset(asset JumpAsset, stat *syncStat) error {
	var existing JumpAsset
	err := h.db.Where("source = ? AND source_ref = ?", asset.Source, asset.SourceRef).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := h.db.Create(&asset).Error; err != nil {
				return err
			}
			stat.Created++
			return nil
		}
		return err
	}

	updates := map[string]interface{}{
		"name":        asset.Name,
		"asset_type":  asset.AssetType,
		"protocol":    asset.Protocol,
		"address":     asset.Address,
		"port":        asset.Port,
		"cluster":     asset.Cluster,
		"namespace":   asset.Namespace,
		"tags":        asset.Tags,
		"description": asset.Description,
		"enabled":     asset.Enabled,
	}
	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		return err
	}
	stat.Updated++
	return nil
}

func (h *JumpHandler) attachPolicyNames(items []JumpPermissionPolicy) []JumpPolicyView {
	assetNames := map[string]string{}
	accountNames := map[string]string{}

	assetIDs := make([]string, 0)
	accountIDs := make([]string, 0)
	assetSeen := map[string]struct{}{}
	accountSeen := map[string]struct{}{}
	for i := range items {
		if id := items[i].AssetID; id != "" {
			if _, ok := assetSeen[id]; !ok {
				assetSeen[id] = struct{}{}
				assetIDs = append(assetIDs, id)
			}
		}
		if id := items[i].AccountID; id != "" {
			if _, ok := accountSeen[id]; !ok {
				accountSeen[id] = struct{}{}
				accountIDs = append(accountIDs, id)
			}
		}
	}

	if len(assetIDs) > 0 {
		var assets []JumpAsset
		h.db.Where("id IN ?", assetIDs).Find(&assets)
		for i := range assets {
			assetNames[assets[i].ID] = assets[i].Name
		}
	}
	if len(accountIDs) > 0 {
		var accounts []JumpAccount
		h.db.Where("id IN ?", accountIDs).Find(&accounts)
		for i := range accounts {
			accountNames[accounts[i].ID] = accounts[i].Name
		}
	}

	views := make([]JumpPolicyView, 0, len(items))
	for i := range items {
		item := items[i]
		views = append(views, JumpPolicyView{
			JumpPermissionPolicy: item,
			AssetName:            assetNames[item.AssetID],
			AccountName:          accountNames[item.AccountID],
		})
	}
	return views
}

func (h *JumpHandler) resolveAccessPolicy(userID, roleCode, assetID, accountID, protocol string) (*JumpPermissionPolicy, error) {
	if roleCode == "admin" {
		return &JumpPermissionPolicy{
			Name:           "admin-default",
			UserID:         userID,
			RoleCode:       roleCode,
			AssetID:        assetID,
			AccountID:      accountID,
			Protocol:       protocol,
			RequireApprove: false,
			Status:         1,
		}, nil
	}
	now := time.Now()
	query := h.db.Model(&JumpPermissionPolicy{}).
		Where("status = ?", 1).
		Where("asset_id = ?", assetID).
		Where("account_id = ?", accountID).
		Where("(user_id = ? OR role_code = ?)", userID, roleCode).
		Where("(expires_at IS NULL OR expires_at > ?)", now).
		Order("require_approve DESC, updated_at DESC, created_at DESC")
	if strings.TrimSpace(protocol) != "" {
		query = query.Where("(protocol = ? OR protocol = '')", protocol)
	}
	var policy JumpPermissionPolicy
	if err := query.First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (h *JumpHandler) evaluateCommandRisk(protocol, command string) commandRiskDecision {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return commandRiskDecision{}
	}

	var rules []JumpCommandRule
	query := h.db.Model(&JumpCommandRule{}).Where("enabled = ?", true).Order("priority DESC, created_at ASC")
	if p := strings.ToLower(strings.TrimSpace(protocol)); p != "" {
		query = query.Where("(protocol = '' OR protocol = ?)", p)
	}
	if err := query.Find(&rules).Error; err != nil {
		return commandRiskDecision{}
	}
	if len(rules) == 0 {
		return commandRiskDecision{}
	}

	matches := make([]commandRuleHit, 0)
	allowMatches := make([]commandRuleHit, 0)
	riskMatches := make([]commandRuleHit, 0)
	for i := range rules {
		rule := rules[i]
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		matched := false
		switch strings.ToLower(strings.TrimSpace(rule.MatchType)) {
		case "exact":
			matched = cmd == pattern
		case "prefix":
			matched = strings.HasPrefix(cmd, pattern)
		case "regex":
			re, err := regexp.Compile(pattern)
			if err == nil {
				matched = re.MatchString(cmd)
			}
		default:
			matched = strings.Contains(strings.ToLower(cmd), strings.ToLower(pattern))
		}
		if !matched {
			continue
		}
		hit := commandRuleHit{
			ID:       rule.ID,
			Name:     rule.Name,
			RuleKind: normalizeRuleKind(rule.RuleKind),
			Action:   normalizeRuleAction(rule.Action),
			Severity: normalizeRuleSeverity(rule.Severity),
			Priority: rule.Priority,
			Pattern:  pattern,
		}
		matches = append(matches, hit)
		if hit.RuleKind == "allow" {
			allowMatches = append(allowMatches, hit)
			continue
		}
		riskMatches = append(riskMatches, hit)
	}
	if len(matches) == 0 {
		return commandRiskDecision{}
	}
	if len(allowMatches) > 0 {
		ruleIDs := make([]string, 0, len(allowMatches))
		ruleNames := make([]string, 0, len(allowMatches))
		for i := range allowMatches {
			ruleIDs = append(ruleIDs, allowMatches[i].ID)
			ruleNames = append(ruleNames, allowMatches[i].Name)
		}
		return commandRiskDecision{
			Matched:      true,
			WhitelistHit: true,
			RuleID:       strings.Join(ruleIDs, ","),
			RuleName:     strings.Join(ruleNames, ","),
			RuleIDs:      ruleIDs,
			RuleNames:    ruleNames,
			MatchedRules: matches,
			Severity:     "info",
			Action:       "allow",
			Reason:       "命中白名单规则，已放行",
		}
	}
	if len(riskMatches) == 0 {
		return commandRiskDecision{}
	}

	ruleIDs := make([]string, 0, len(riskMatches))
	ruleNames := make([]string, 0, len(riskMatches))
	action := "alert"
	severity := "info"
	for i := range riskMatches {
		ruleIDs = append(ruleIDs, riskMatches[i].ID)
		ruleNames = append(ruleNames, riskMatches[i].Name)
		if riskMatches[i].Action == "block" {
			action = "block"
		}
		if compareSeverity(riskMatches[i].Severity, severity) > 0 {
			severity = riskMatches[i].Severity
		}
	}

	return commandRiskDecision{
		Matched:      true,
		RuleID:       strings.Join(ruleIDs, ","),
		RuleName:     strings.Join(ruleNames, ","),
		RuleIDs:      ruleIDs,
		RuleNames:    ruleNames,
		MatchedRules: matches,
		Severity:     severity,
		Action:       action,
		Reason:       fmt.Sprintf("命中规则[%s]，命令已%s", strings.Join(ruleNames, ","), map[bool]string{true: "阻断", false: "记录告警"}[action == "block"]),
	}
}

func (h *JumpHandler) createCommandRiskAlert(session *JumpSession, decision commandRiskDecision, command string, firedAt time.Time) string {
	if session == nil || !decision.Matched || decision.WhitelistHit || decision.Action == "allow" {
		return ""
	}
	primaryRuleID := decision.RuleID
	if len(decision.RuleIDs) > 0 && strings.TrimSpace(decision.RuleIDs[0]) != "" {
		primaryRuleID = strings.TrimSpace(decision.RuleIDs[0])
	}
	combinedRuleName := strings.Join(decision.RuleNames, ",")
	if combinedRuleName == "" {
		combinedRuleName = decision.RuleName
	}
	labels, _ := json.Marshal(map[string]string{
		"source":       "jump",
		"jump_session": session.ID,
		"protocol":     session.Protocol,
		"asset":        session.AssetName,
		"user":         session.Username,
		"rule_id":      primaryRuleID,
		"action":       decision.Action,
	})
	annotations, _ := json.Marshal(map[string]string{
		"command": command,
		"reason":  decision.Reason,
	})
	hash := sha1.Sum([]byte(session.ID + "|" + strings.Join(decision.RuleIDs, ",") + "|" + command + "|" + firedAt.Format(time.RFC3339Nano)))
	alert := map[string]interface{}{
		"id":          uuid.NewString(),
		"created_at":  firedAt,
		"updated_at":  firedAt,
		"rule_id":     primaryRuleID,
		"rule_name":   "Jump命令风控/" + truncateJumpText(combinedRuleName, 96),
		"fingerprint": hex.EncodeToString(hash[:]),
		"target":      session.AssetName,
		"metric":      "jump_command_risk",
		"value":       truncateJumpText(command, 64),
		"threshold":   truncateJumpText(combinedRuleName, 64),
		"severity":    decision.Severity,
		"status":      0,
		"fired_at":    firedAt,
		"group_key":   "jump-command-risk-" + primaryRuleID,
		"labels":      string(labels),
		"annotations": string(annotations),
	}
	if err := h.db.Table("alerts").Create(alert).Error; err != nil {
		return ""
	}
	id, _ := alert["id"].(string)
	return id
}

func truncateJumpText(in string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(in)
	if len(r) <= max {
		return in
	}
	return string(r[:max])
}

func normalizeRuleMatchType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "exact":
		return "exact"
	case "prefix":
		return "prefix"
	case "regex":
		return "regex"
	default:
		return "contains"
	}
}

func normalizeRuleKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "allow":
		return "allow"
	default:
		return "risk"
	}
}

func normalizeRuleSeverity(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "critical":
		return "critical"
	case "info":
		return "info"
	default:
		return "warning"
	}
}

func normalizeRuleAction(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "block":
		return "block"
	default:
		return "alert"
	}
}

func compareSeverity(a, b string) int {
	rank := map[string]int{
		"info":     1,
		"warning":  2,
		"critical": 3,
	}
	return rank[normalizeRuleSeverity(a)] - rank[normalizeRuleSeverity(b)]
}

func inferAssetType(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "ssh", "docker":
		return "host"
	case "k8s":
		return "k8s"
	case "mysql", "postgres", "redis", "mongodb":
		return "database"
	default:
		return "host"
	}
}

func inferPort(protocol string) int {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "ssh", "docker":
		return 22
	case "k8s":
		return 443
	case "mysql":
		return 3306
	case "postgres":
		return 5432
	case "redis":
		return 6379
	case "mongodb":
		return 27017
	default:
		return 22
	}
}
