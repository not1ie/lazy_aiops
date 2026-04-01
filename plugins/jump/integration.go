package jump

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	jumpProviderJumpServer = "jumpserver"
	jumpAuthBearerToken    = "bearer_token"
	jumpAuthPrivateToken   = "private_token"
	jumpAuthPassword       = "password"
	jumpSourceLocal        = "local"
	jumpSourceJumpServer   = "jumpserver"

	defaultJumpOrgID = "00000000-0000-0000-0000-000000000002"
)

var (
	errJumpIntegrationDisabled      = errors.New("jumpserver integration disabled")
	errJumpIntegrationNotConfigured = errors.New("jumpserver integration not configured")
)

type jumpIntegrationUpdateRequest struct {
	Enabled      bool   `json:"enabled"`
	BaseURL      string `json:"base_url"`
	OrgID        string `json:"org_id"`
	AuthType     string `json:"auth_type"`
	AuthUsername string `json:"auth_username"`
	AuthSecret   string `json:"auth_secret"`
	VerifyTLS    *bool  `json:"verify_tls"`
	AutoSync     *bool  `json:"auto_sync"`
}

type jumpServerSyncResult struct {
	Hosts     syncStat `json:"hosts"`
	Databases syncStat `json:"databases"`
	Sessions  syncStat `json:"sessions"`
	Commands  syncStat `json:"commands"`
	Total     syncStat `json:"total"`
}

type jumpServerClient struct {
	config      *JumpIntegrationConfig
	secret      string
	client      *http.Client
	bearerToken string
	tokenExpiry time.Time
}

type jumpServerAPIError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
}

func (e *jumpServerAPIError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("JumpServer API %s %s 失败(%d): %s", e.Method, e.Path, e.StatusCode, e.Message)
}

func (h *JumpHandler) GetIntegrationConfig(c *gin.Context) {
	cfg, err := h.getOrCreateJumpIntegrationConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toJumpIntegrationView(cfg)})
}

func (h *JumpHandler) UpdateIntegrationConfig(c *gin.Context) {
	cfg, err := h.getOrCreateJumpIntegrationConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	var req jumpIntegrationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	authType := normalizeJumpAuthType(req.AuthType)
	authUsername := strings.TrimSpace(req.AuthUsername)
	orgID := strings.TrimSpace(req.OrgID)
	if orgID == "" {
		orgID = defaultJumpOrgID
	}
	if req.Enabled && baseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "启用集成时，JumpServer 地址不能为空"})
		return
	}
	if req.Enabled && (authType == jumpAuthBearerToken || authType == jumpAuthPrivateToken) &&
		strings.TrimSpace(req.AuthSecret) == "" && strings.TrimSpace(cfg.AuthSecret) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "启用集成时，请提供 Token / Private Token"})
		return
	}
	if authType == jumpAuthPassword && authUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "密码认证模式下必须提供用户名"})
		return
	}

	updates := map[string]interface{}{
		"provider":      jumpProviderJumpServer,
		"enabled":       req.Enabled,
		"base_url":      normalizeBaseURL(baseURL),
		"org_id":        orgID,
		"auth_type":     authType,
		"auth_username": authUsername,
	}
	if req.VerifyTLS != nil {
		updates["verify_tls"] = *req.VerifyTLS
	}
	if req.AutoSync != nil {
		updates["auto_sync"] = *req.AutoSync
	}
	if secret := strings.TrimSpace(req.AuthSecret); secret != "" {
		enc, encErr := encryptJumpSecret(h.secretKey, secret)
		if encErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "接入密钥加密失败"})
			return
		}
		updates["auth_secret"] = enc
	}

	if err := h.db.Model(cfg).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(cfg, "id = ?", cfg.ID).Error
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": toJumpIntegrationView(cfg)})
}

func (h *JumpHandler) TestIntegrationConnection(c *gin.Context) {
	client, err := h.newJumpServerClient(true)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, errJumpIntegrationNotConfigured) && !errors.Is(err, errJumpIntegrationDisabled) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var profileEndpoint string
	var profileErr error
	for _, p := range jumpProfileCandidates() {
		_, _, reqErr := client.request(http.MethodGet, p, nil, true)
		if reqErr == nil {
			profileEndpoint = p
			break
		}
		profileErr = reqErr
	}
	if profileEndpoint == "" {
		msg := "JumpServer 连接测试失败，请检查地址、认证方式与密钥"
		if profileErr != nil {
			msg = describeJumpConnectionError(profileErr)
		}
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": msg})
		return
	}

	_, hostEndpoint, hostErr := client.listByCandidates(jumpHostAssetCandidates())
	if hostErr != nil {
		status := http.StatusBadGateway
		if apiErr := asJumpServerAPIError(hostErr); apiErr != nil && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"code":    403,
			"message": describeJumpSyncError(hostErr, "主机资产"),
			"data": gin.H{
				"checked_endpoint":       profileEndpoint,
				"required_sync_endpoint": jumpHostAssetCandidates()[0],
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "JumpServer 连接成功，资产读取权限正常",
		"data": gin.H{
			"checked_endpoint":       profileEndpoint,
			"required_sync_endpoint": hostEndpoint,
		},
	})
}

func (h *JumpHandler) SyncFromJumpServer(c *gin.Context) {
	result, err := h.syncFromJumpServerAssets(true)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, errJumpIntegrationNotConfigured) && !errors.Is(err, errJumpIntegrationDisabled) {
			status = http.StatusBadGateway
		}
		c.JSON(status, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "message": fmt.Sprintf("JumpServer 同步完成，新增%d，更新%d", result.Total.Created, result.Total.Updated)})
}

func (h *JumpHandler) SyncJumpServerSessions(c *gin.Context) {
	result, err := h.syncFromJumpServerSessions(true)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, errJumpIntegrationNotConfigured) && !errors.Is(err, errJumpIntegrationDisabled) {
			status = http.StatusBadGateway
		}
		c.JSON(status, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    result,
		"message": fmt.Sprintf("JumpServer 会话同步完成，会话新增%d更新%d，命令新增%d更新%d", result.Sessions.Created, result.Sessions.Updated, result.Commands.Created, result.Commands.Updated),
	})
}

func (h *JumpHandler) syncFromJumpServerAssets(strict bool) (jumpServerSyncResult, error) {
	client, err := h.newJumpServerClient(strict)
	if err != nil {
		return jumpServerSyncResult{}, err
	}
	if client == nil {
		return jumpServerSyncResult{}, nil
	}

	result := jumpServerSyncResult{}
	hosts, hostEndpoint, hostErr := client.listByCandidates(jumpHostAssetCandidates())
	if hostErr == nil {
		for i := range hosts {
			asset := buildJumpAssetFromJumpServerHost(hosts[i])
			if strings.TrimSpace(asset.SourceRef) == "" || strings.TrimSpace(asset.Name) == "" {
				continue
			}
			if upsertErr := h.upsertAsset(asset, &result.Hosts); upsertErr != nil {
				hostErr = upsertErr
				break
			}
		}
	}
	if hostErr != nil {
		hostErr = errors.New(describeJumpSyncError(hostErr, "主机资产"))
	}

	databases, dbEndpoint, dbErr := client.listByCandidates(jumpDatabaseAssetCandidates())
	if dbErr == nil {
		for i := range databases {
			asset := buildJumpAssetFromJumpServerDatabase(databases[i])
			if strings.TrimSpace(asset.SourceRef) == "" || strings.TrimSpace(asset.Name) == "" {
				continue
			}
			if upsertErr := h.upsertAsset(asset, &result.Databases); upsertErr != nil {
				dbErr = upsertErr
				break
			}
		}
	}
	if dbErr != nil {
		dbErr = errors.New(describeJumpSyncError(dbErr, "数据库资产"))
	}

	result.Total.Created = result.Hosts.Created + result.Databases.Created
	result.Total.Updated = result.Hosts.Updated + result.Databases.Updated

	cfg := client.config
	now := time.Now()
	if cfg != nil {
		if hostErr != nil || dbErr != nil {
			msg := strings.TrimSpace(nonEmptyError(hostErr, dbErr))
			_ = h.db.Model(cfg).Updates(map[string]interface{}{
				"last_sync_at":     now,
				"last_sync_status": "failed",
				"last_sync_msg":    truncateJumpText(msg, 500),
			}).Error
		} else {
			hostLabel := hostEndpoint
			if strings.TrimSpace(hostLabel) == "" {
				hostLabel = jumpHostAssetCandidates()[0]
			}
			dbLabel := dbEndpoint
			if strings.TrimSpace(dbLabel) == "" {
				dbLabel = jumpDatabaseAssetCandidates()[0]
			}
			_ = h.db.Model(cfg).Updates(map[string]interface{}{
				"last_sync_at":     now,
				"last_sync_status": "ok",
				"last_sync_msg": fmt.Sprintf(
					"hosts(%s) +%d/%d, databases(%s) +%d/%d",
					hostLabel,
					result.Hosts.Created,
					result.Hosts.Updated,
					dbLabel,
					result.Databases.Created,
					result.Databases.Updated,
				),
			}).Error
		}
	}

	if hostErr != nil {
		return result, fmt.Errorf("同步 JumpServer 主机失败: %s", hostErr.Error())
	}
	if dbErr != nil {
		return result, fmt.Errorf("同步 JumpServer 数据库失败: %s", dbErr.Error())
	}
	return result, nil
}

func (h *JumpHandler) syncFromJumpServerSessions(strict bool) (jumpServerSyncResult, error) {
	client, err := h.newJumpServerClient(strict)
	if err != nil {
		return jumpServerSyncResult{}, err
	}
	if client == nil {
		return jumpServerSyncResult{}, nil
	}

	result := jumpServerSyncResult{}
	sessionItems, _, sessionErr := client.listByCandidates([]string{
		"/api/v1/terminal/sessions/",
		"/api/v1/audits/sessions/",
		"/api/v1/terminal/session/",
	})
	if sessionErr != nil {
		return result, fmt.Errorf("同步 JumpServer 会话失败: %w", sessionErr)
	}

	localSessionIDByRemote := make(map[string]string, len(sessionItems))
	for i := range sessionItems {
		session := buildJumpSessionFromJumpServer(sessionItems[i])
		if strings.TrimSpace(session.SourceRef) == "" {
			continue
		}
		if strings.TrimSpace(session.SessionNo) == "" {
			session.SessionNo = "JMS-" + strings.ToUpper(shortHash(session.SourceRef, 12))
		}
		if strings.TrimSpace(session.Source) == "" {
			session.Source = jumpSourceJumpServer
		}
		if strings.TrimSpace(session.Status) == "" {
			session.Status = "closed"
		}
		id, upsertErr := h.upsertJumpSession(session, &result.Sessions)
		if upsertErr != nil {
			return result, upsertErr
		}
		if id != "" {
			localSessionIDByRemote[session.SourceRef] = id
		}
	}

	commandItems, _, commandErr := client.listByCandidates([]string{
		"/api/v1/terminal/commands/",
		"/api/v1/audits/commands/",
		"/api/v1/terminal/session-commands/",
	})
	if commandErr == nil {
		for i := range commandItems {
			cmd := buildJumpCommandFromJumpServer(commandItems[i])
			if strings.TrimSpace(cmd.SourceRef) == "" {
				continue
			}
			remoteSessionID := strings.TrimSpace(extractSessionSourceRef(commandItems[i]))
			if remoteSessionID == "" {
				continue
			}
			localSessionID := localSessionIDByRemote[remoteSessionID]
			if localSessionID == "" {
				localSessionID = h.findLocalSessionIDByRemote(remoteSessionID)
				if localSessionID != "" {
					localSessionIDByRemote[remoteSessionID] = localSessionID
				}
			}
			if localSessionID == "" {
				continue
			}
			cmd.SessionID = localSessionID
			if strings.TrimSpace(cmd.Source) == "" {
				cmd.Source = jumpSourceJumpServer
			}
			if upsertErr := h.upsertJumpCommand(cmd, &result.Commands); upsertErr != nil {
				return result, upsertErr
			}
		}
	}

	result.Total.Created = result.Sessions.Created + result.Commands.Created
	result.Total.Updated = result.Sessions.Updated + result.Commands.Updated

	cfg := client.config
	now := time.Now()
	if cfg != nil {
		if commandErr != nil {
			_ = h.db.Model(cfg).Updates(map[string]interface{}{
				"last_sync_at":     now,
				"last_sync_status": "ok",
				"last_sync_msg": fmt.Sprintf(
					"sessions +%d/%d, commands skipped: %s",
					result.Sessions.Created,
					result.Sessions.Updated,
					truncateJumpText(commandErr.Error(), 220),
				),
			}).Error
		} else {
			_ = h.db.Model(cfg).Updates(map[string]interface{}{
				"last_sync_at":     now,
				"last_sync_status": "ok",
				"last_sync_msg": fmt.Sprintf(
					"sessions +%d/%d, commands +%d/%d",
					result.Sessions.Created,
					result.Sessions.Updated,
					result.Commands.Created,
					result.Commands.Updated,
				),
			}).Error
		}
	}
	return result, nil
}

func (h *JumpHandler) getOrCreateJumpIntegrationConfig() (*JumpIntegrationConfig, error) {
	var cfg JumpIntegrationConfig
	err := h.db.Where("provider = ?", jumpProviderJumpServer).Order("created_at DESC").First(&cfg).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cfg = JumpIntegrationConfig{
		Provider:     jumpProviderJumpServer,
		Enabled:      false,
		OrgID:        defaultJumpOrgID,
		AuthType:     jumpAuthBearerToken,
		VerifyTLS:    true,
		AutoSync:     false,
		AuthUsername: "",
	}
	if err := h.db.Create(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func toJumpIntegrationView(cfg *JumpIntegrationConfig) JumpIntegrationConfigView {
	if cfg == nil {
		return JumpIntegrationConfigView{}
	}
	return JumpIntegrationConfigView{
		ID:             cfg.ID,
		CreatedAt:      cfg.CreatedAt,
		UpdatedAt:      cfg.UpdatedAt,
		Provider:       cfg.Provider,
		Enabled:        cfg.Enabled,
		BaseURL:        cfg.BaseURL,
		OrgID:          cfg.OrgID,
		AuthType:       cfg.AuthType,
		AuthUsername:   cfg.AuthUsername,
		HasAuthSecret:  strings.TrimSpace(cfg.AuthSecret) != "",
		VerifyTLS:      cfg.VerifyTLS,
		AutoSync:       cfg.AutoSync,
		LastSyncAt:     cfg.LastSyncAt,
		LastSyncStatus: cfg.LastSyncStatus,
		LastSyncMsg:    cfg.LastSyncMsg,
	}
}

func (h *JumpHandler) newJumpServerClient(strict bool) (*jumpServerClient, error) {
	cfg, err := h.getOrCreateJumpIntegrationConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		if strict {
			return nil, errJumpIntegrationNotConfigured
		}
		return nil, nil
	}
	if !cfg.Enabled {
		if strict {
			return nil, errJumpIntegrationDisabled
		}
		return nil, nil
	}
	baseURL := normalizeBaseURL(cfg.BaseURL)
	if baseURL == "" {
		return nil, errJumpIntegrationNotConfigured
	}
	if _, parseErr := url.Parse(baseURL); parseErr != nil {
		return nil, fmt.Errorf("JumpServer 地址格式错误: %w", parseErr)
	}
	secret, err := decryptJumpSecret(h.secretKey, strings.TrimSpace(cfg.AuthSecret))
	if err != nil {
		return nil, errors.New("JumpServer 接入密钥解密失败，请重新保存配置")
	}
	secret = strings.TrimSpace(secret)
	authType := normalizeJumpAuthType(cfg.AuthType)
	if authType == jumpAuthPassword && strings.TrimSpace(cfg.AuthUsername) == "" {
		return nil, errors.New("密码认证模式下必须配置用户名")
	}
	if secret == "" {
		return nil, errJumpIntegrationNotConfigured
	}

	cfg.BaseURL = baseURL
	cfg.AuthType = authType
	if strings.TrimSpace(cfg.OrgID) == "" {
		cfg.OrgID = defaultJumpOrgID
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	if !cfg.VerifyTLS {
		tr.TLSClientConfig.InsecureSkipVerify = true
	}
	return &jumpServerClient{
		config: cfg,
		secret: secret,
		client: &http.Client{
			Timeout:   20 * time.Second,
			Transport: tr,
		},
	}, nil
}

func (c *jumpServerClient) listAssets(endpoint string) ([]map[string]interface{}, error) {
	all := make([]map[string]interface{}, 0, 128)
	next := ""
	limit := 100
	offset := 0

	for page := 0; page < 100; page++ {
		target := endpoint
		if strings.TrimSpace(next) != "" {
			target = next
		} else {
			target = withPagination(endpoint, limit, offset)
		}
		raw, _, err := c.request(http.MethodGet, target, nil, true)
		if err != nil {
			return nil, err
		}
		items, nextURL, count, paged := extractListPayload(raw)
		all = append(all, items...)

		if strings.TrimSpace(nextURL) != "" {
			next = nextURL
			offset += limit
			continue
		}
		if paged {
			offset += limit
			if count > 0 && len(all) < count {
				continue
			}
		}
		break
	}
	return all, nil
}

func (c *jumpServerClient) listByCandidates(candidates []string) ([]map[string]interface{}, string, error) {
	var lastErr error
	var preferredErr error
	for i := range candidates {
		endpoint := strings.TrimSpace(candidates[i])
		if endpoint == "" {
			continue
		}
		items, err := c.listAssets(endpoint)
		if err == nil {
			return items, endpoint, nil
		}
		if preferredErr == nil {
			if apiErr := asJumpServerAPIError(err); apiErr != nil &&
				(apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
				preferredErr = err
			}
		}
		lastErr = err
	}
	if preferredErr != nil {
		return nil, "", preferredErr
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return []map[string]interface{}{}, "", errors.New("未配置可用的 JumpServer 列表接口")
}

func (c *jumpServerClient) request(method, path string, payload interface{}, withAuth bool) (interface{}, int, error) {
	target := buildTargetURL(c.config.BaseURL, path)
	var bodyReader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, target, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lazy-auto-ops/jump-integration")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		if err := c.applyAuth(req); err != nil {
			return nil, 0, err
		}
	}
	if strings.TrimSpace(c.config.OrgID) != "" {
		req.Header.Set("X-JMS-ORG", c.config.OrgID)
		req.Header.Set("X-JMS-ORGID", c.config.OrgID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, &jumpServerAPIError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Message:    extractRemoteError(body),
		}
	}
	if len(body) == 0 {
		return map[string]interface{}{}, resp.StatusCode, nil
	}
	var decoded interface{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return map[string]interface{}{"raw": string(body)}, resp.StatusCode, nil
	}
	return decoded, resp.StatusCode, nil
}

func (c *jumpServerClient) applyAuth(req *http.Request) error {
	switch normalizeJumpAuthType(c.config.AuthType) {
	case jumpAuthPrivateToken:
		req.Header.Set("Authorization", "Token "+c.secret)
		req.Header.Set("X-JMS-TOKEN", c.secret)
		return nil
	case jumpAuthPassword:
		token, err := c.ensurePasswordToken()
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	default:
		req.Header.Set("Authorization", "Bearer "+c.secret)
		return nil
	}
}

func (c *jumpServerClient) ensurePasswordToken() (string, error) {
	if strings.TrimSpace(c.bearerToken) != "" && time.Now().Before(c.tokenExpiry) {
		return c.bearerToken, nil
	}
	username := strings.TrimSpace(c.config.AuthUsername)
	if username == "" {
		return "", errors.New("密码认证缺少用户名")
	}
	payload := map[string]string{
		"username": username,
		"password": c.secret,
	}
	raw, _, err := c.request(http.MethodPost, "/api/v1/authentication/auth/", payload, false)
	if err != nil {
		return "", err
	}
	token := extractToken(raw)
	if token == "" {
		return "", errors.New("无法从 JumpServer 登录响应中解析 token")
	}
	c.bearerToken = token
	c.tokenExpiry = time.Now().Add(30 * time.Minute)
	return token, nil
}

func extractToken(raw interface{}) string {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"token", "access", "jwt"} {
		if v := strings.TrimSpace(anyToString(m[key])); v != "" {
			return v
		}
	}
	if nested, ok := m["data"].(map[string]interface{}); ok {
		for _, key := range []string{"token", "access", "jwt"} {
			if v := strings.TrimSpace(anyToString(nested[key])); v != "" {
				return v
			}
		}
	}
	return ""
}

func extractListPayload(raw interface{}) ([]map[string]interface{}, string, int, bool) {
	toMapList := func(v interface{}) []map[string]interface{} {
		items := make([]map[string]interface{}, 0)
		arr, ok := v.([]interface{})
		if !ok {
			return items
		}
		for i := range arr {
			if item, ok := arr[i].(map[string]interface{}); ok {
				items = append(items, item)
			}
		}
		return items
	}

	if arr, ok := raw.([]interface{}); ok {
		items := make([]map[string]interface{}, 0, len(arr))
		for i := range arr {
			if m, ok := arr[i].(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
		return items, "", len(items), false
	}

	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, "", 0, false
	}
	if results, ok := m["results"]; ok {
		items := toMapList(results)
		next := strings.TrimSpace(anyToString(m["next"]))
		count := anyToInt(m["count"])
		return items, next, count, true
	}
	for _, key := range []string{"data", "items", "list"} {
		if v, ok := m[key]; ok {
			items := toMapList(v)
			if len(items) > 0 {
				return items, strings.TrimSpace(anyToString(m["next"])), anyToInt(m["count"]), true
			}
		}
	}
	return nil, "", 0, false
}

func buildJumpAssetFromJumpServerHost(item map[string]interface{}) JumpAsset {
	protocol := strings.ToLower(strings.TrimSpace(firstNonEmpty(item, "protocol", "protocols")))
	if protocol == "" {
		protocol = "ssh"
	}
	port := anyToInt(item["port"])
	if port <= 0 {
		port = inferPort(protocol)
	}

	name := firstNonEmpty(item, "name", "hostname", "ip", "address")
	address := firstNonEmpty(item, "address", "ip", "hostname")
	sourceRef := firstNonEmpty(item, "id", "uuid")

	return JumpAsset{
		Name:        strings.TrimSpace(name),
		AssetType:   "host",
		Protocol:    protocol,
		Address:     strings.TrimSpace(address),
		Port:        port,
		Source:      "jumpserver_host",
		SourceRef:   strings.TrimSpace(sourceRef),
		Tags:        strings.TrimSpace(firstNonEmpty(item, "labels", "tags")),
		Description: strings.TrimSpace(firstNonEmpty(item, "comment", "description")),
		Enabled:     anyToBool(item["is_active"], true),
	}
}

func buildJumpAssetFromJumpServerDatabase(item map[string]interface{}) JumpAsset {
	engine := strings.ToLower(strings.TrimSpace(firstNonEmpty(item, "db_type", "engine", "type", "protocol")))
	protocol := normalizeDBProtocol(engine)
	if protocol == "" {
		protocol = "mysql"
	}
	port := anyToInt(item["port"])
	if port <= 0 {
		port = inferPort(protocol)
	}

	name := firstNonEmpty(item, "name", "hostname", "address")
	address := firstNonEmpty(item, "host", "address", "ip")
	sourceRef := firstNonEmpty(item, "id", "uuid")
	namespace := firstNonEmpty(item, "database", "db_name", "schema")

	return JumpAsset{
		Name:        strings.TrimSpace(name),
		AssetType:   "database",
		Protocol:    protocol,
		Address:     strings.TrimSpace(address),
		Port:        port,
		Namespace:   strings.TrimSpace(namespace),
		Source:      "jumpserver_database",
		SourceRef:   strings.TrimSpace(sourceRef),
		Tags:        strings.TrimSpace(firstNonEmpty(item, "labels", "tags")),
		Description: strings.TrimSpace(firstNonEmpty(item, "comment", "description")),
		Enabled:     anyToBool(item["is_active"], true),
	}
}

func buildJumpSessionFromJumpServer(item map[string]interface{}) JumpSession {
	sourceRef := strings.TrimSpace(firstNonEmpty(item, "id", "uuid", "session_id", "sid"))
	sessionNo := strings.TrimSpace(firstNonEmpty(item, "session", "session_id", "id"))
	startedAt := parseAnyTime(lookupString(item, "date_start"), parseAnyTime(lookupString(item, "created_at"), time.Now()))
	endedAt := parseNullableTime(
		lookupString(item, "date_end"),
		lookupString(item, "finished_at"),
		lookupString(item, "end_time"),
	)

	status := normalizeJumpSessionStatus(firstNonEmpty(
		item,
		"status",
		"state",
		"is_finished",
		"is_success",
	))
	if status == "" {
		if endedAt != nil {
			status = "closed"
		} else {
			status = "active"
		}
	}

	protocol := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		item,
		"protocol",
		"asset_protocol",
		"type",
		"terminal_type",
	)))
	if protocol == "" {
		protocol = "ssh"
	}

	assetRemoteID := lookupString(item, "asset.id")
	accountRemoteID := lookupString(item, "system_user.id")
	assetName := firstNonEmpty(item, "asset_name", "asset")
	if assetName == "" {
		assetName = lookupString(item, "asset.name")
	}
	accountName := firstNonEmpty(item, "account_name", "system_user")
	if accountName == "" {
		accountName = lookupString(item, "system_user.name")
	}
	username := firstNonEmpty(item, "username", "user")
	if username == "" {
		username = lookupString(item, "user.username")
	}
	if username == "" {
		username = lookupString(item, "user.name")
	}
	commandCount := anyToInt(lookupAny(item, "command_count", "cmd_count", "command_amount"))
	duration := anyToInt(lookupAny(item, "duration", "duration_sec"))
	if duration <= 0 && endedAt != nil {
		duration = int(endedAt.Sub(startedAt).Seconds())
		if duration < 0 {
			duration = 0
		}
	}

	result := JumpSession{
		Source:       jumpSourceJumpServer,
		SourceRef:    sourceRef,
		SessionNo:    sessionNo,
		UserID:       "",
		Username:     truncateJumpText(strings.TrimSpace(username), 128),
		RoleCode:     "",
		AssetID:      "",
		AssetName:    truncateJumpText(strings.TrimSpace(assetName), 128),
		AccountID:    "",
		AccountName:  truncateJumpText(strings.TrimSpace(accountName), 128),
		PolicyID:     "",
		Protocol:     truncateJumpText(protocol, 32),
		SourceIP:     truncateJumpText(firstNonEmpty(item, "remote_addr", "remote_ip", "addr", "client_ip"), 64),
		Status:       status,
		StartedAt:    startedAt,
		CommandCount: commandCount,
		DurationSec:  duration,
		CloseReason:  truncateJumpText(firstNonEmpty(item, "close_reason", "reason"), 256),
		ApprovedBy:   truncateJumpText(firstNonEmpty(item, "approved_by", "reviewer"), 128),
	}
	if endedAt != nil {
		result.EndedAt = endedAt
	}
	if t := parseNullableTime(lookupString(item, "approved_at")); t != nil {
		result.ApprovedAt = t
	}
	if t := parseNullableTime(lookupString(item, "last_command_at")); t != nil {
		result.LastCommandAt = t
	}
	if sourceRef != "" {
		result.RelaySessionID = ""
	}
	result.AssetID = strings.TrimSpace(assetRemoteID)
	result.AccountID = strings.TrimSpace(accountRemoteID)
	return result
}

func (h *JumpHandler) upsertJumpSession(session JumpSession, stat *syncStat) (string, error) {
	localAssetID := h.findLocalAssetIDByRemote(session.AssetID)
	if localAssetID != "" {
		session.AssetID = localAssetID
	} else {
		session.AssetID = ""
	}
	session.AccountID = ""

	var existing JumpSession
	err := h.db.Where("source = ? AND source_ref = ?", jumpSourceJumpServer, session.SourceRef).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if strings.TrimSpace(session.SessionNo) == "" {
				session.SessionNo = "JMS-" + strings.ToUpper(shortHash(session.SourceRef, 12))
			}
			if h.db.Where("session_no = ?", session.SessionNo).First(&JumpSession{}).Error == nil {
				session.SessionNo = "JMS-" + strings.ToUpper(shortHash(session.SourceRef+time.Now().String(), 16))
			}
			if err := h.db.Create(&session).Error; err != nil {
				return "", err
			}
			if stat != nil {
				stat.Created++
			}
			return session.ID, nil
		}
		return "", err
	}

	updates := map[string]interface{}{
		"session_no":      session.SessionNo,
		"username":        session.Username,
		"asset_name":      session.AssetName,
		"account_name":    session.AccountName,
		"protocol":        session.Protocol,
		"source_ip":       session.SourceIP,
		"status":          session.Status,
		"started_at":      session.StartedAt,
		"last_command_at": session.LastCommandAt,
		"approved_by":     session.ApprovedBy,
		"approved_at":     session.ApprovedAt,
		"ended_at":        session.EndedAt,
		"duration_sec":    session.DurationSec,
		"command_count":   session.CommandCount,
		"close_reason":    session.CloseReason,
	}
	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		return "", err
	}
	if stat != nil {
		stat.Updated++
	}
	return existing.ID, nil
}

func buildJumpCommandFromJumpServer(item map[string]interface{}) JumpCommand {
	cmd := strings.TrimSpace(firstNonEmpty(item, "command", "input", "cmd", "content"))
	if cmd == "" {
		cmd = strings.TrimSpace(lookupString(item, "body"))
	}
	riskLevel := normalizeRuleSeverity(firstNonEmpty(item, "risk_level", "severity", "level"))
	riskAction := normalizeRuleAction(firstNonEmpty(item, "risk_action", "action", "risk_action_display"))
	if riskAction == "" {
		riskAction = "alert"
	}
	outputSnippet := strings.TrimSpace(firstNonEmpty(item, "output", "output_snippet", "result", "response"))
	if len(outputSnippet) > 2000 {
		outputSnippet = outputSnippet[:2000]
	}
	return JumpCommand{
		Source:        jumpSourceJumpServer,
		SourceRef:     strings.TrimSpace(firstNonEmpty(item, "id", "uuid", "command_id")),
		Username:      truncateJumpText(strings.TrimSpace(firstNonEmpty(item, "username", "user")), 128),
		CommandType:   normalizeCommandType(firstNonEmpty(item, "command_type", "type")),
		Command:       truncateJumpText(cmd, 4000),
		ResultCode:    anyToInt(lookupAny(item, "result_code", "exit_code", "code")),
		OutputSnippet: outputSnippet,
		RuleID:        truncateJumpText(strings.TrimSpace(firstNonEmpty(item, "rule_id", "strategy_id")), 256),
		RuleName:      truncateJumpText(strings.TrimSpace(firstNonEmpty(item, "rule_name", "strategy_name")), 512),
		WhitelistHit:  anyToBool(lookupAny(item, "whitelist_hit", "is_allow"), false),
		RiskLevel:     riskLevel,
		RiskAction:    riskAction,
		RiskReason:    truncateJumpText(strings.TrimSpace(firstNonEmpty(item, "risk_reason", "reason", "message")), 512),
		Blocked:       anyToBool(lookupAny(item, "blocked", "is_blocked"), false) || riskAction == "block",
		AlertID:       truncateJumpText(strings.TrimSpace(firstNonEmpty(item, "alert_id")), 36),
		ExecutedAt: parseAnyTime(
			lookupString(item, "executed_at"),
			parseAnyTime(lookupString(item, "date_created"), time.Now()),
		),
	}
}

func (h *JumpHandler) upsertJumpCommand(cmd JumpCommand, stat *syncStat) error {
	if strings.TrimSpace(cmd.SourceRef) == "" {
		return nil
	}
	var existing JumpCommand
	err := h.db.Where("source = ? AND source_ref = ?", jumpSourceJumpServer, cmd.SourceRef).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := h.db.Create(&cmd).Error; err != nil {
				return err
			}
			if stat != nil {
				stat.Created++
			}
			return nil
		}
		return err
	}
	updates := map[string]interface{}{
		"session_id":     cmd.SessionID,
		"username":       cmd.Username,
		"command_type":   cmd.CommandType,
		"command":        cmd.Command,
		"result_code":    cmd.ResultCode,
		"output_snippet": cmd.OutputSnippet,
		"rule_id":        cmd.RuleID,
		"rule_name":      cmd.RuleName,
		"whitelist_hit":  cmd.WhitelistHit,
		"risk_level":     cmd.RiskLevel,
		"risk_action":    cmd.RiskAction,
		"risk_reason":    cmd.RiskReason,
		"blocked":        cmd.Blocked,
		"alert_id":       cmd.AlertID,
		"executed_at":    cmd.ExecutedAt,
	}
	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		return err
	}
	if stat != nil {
		stat.Updated++
	}
	return nil
}

func (h *JumpHandler) findLocalSessionIDByRemote(sourceRef string) string {
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" {
		return ""
	}
	var session JumpSession
	if err := h.db.Select("id").Where("source = ? AND source_ref = ?", jumpSourceJumpServer, sourceRef).First(&session).Error; err != nil {
		return ""
	}
	return session.ID
}

func (h *JumpHandler) findLocalAssetIDByRemote(sourceRef string) string {
	sourceRef = strings.TrimSpace(sourceRef)
	if sourceRef == "" {
		return ""
	}
	var asset JumpAsset
	if err := h.db.Select("id").Where("source_ref = ? AND source IN ?", sourceRef, []string{"jumpserver_host", "jumpserver_database"}).First(&asset).Error; err != nil {
		return ""
	}
	return asset.ID
}

func extractSessionSourceRef(item map[string]interface{}) string {
	if item == nil {
		return ""
	}
	if v := strings.TrimSpace(firstNonEmpty(item, "session_id", "sid")); v != "" {
		return v
	}
	if v := strings.TrimSpace(lookupString(item, "session.id")); v != "" {
		return v
	}
	return ""
}

func normalizeJumpSessionStatus(raw string) string {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "pending", "pending_approval", "waiting", "wait":
		return "pending_approval"
	case "active", "running", "connected", "online":
		return "active"
	case "rejected", "deny", "denied":
		return "rejected"
	case "blocked", "forbidden":
		return "blocked"
	case "closed", "finished", "done", "success", "failed", "disconnected":
		return "closed"
	case "true":
		return "closed"
	case "false":
		return "active"
	default:
		return ""
	}
}

func normalizeCommandType(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "sql":
		return "sql"
	default:
		return "shell"
	}
}

func parseAnyTime(raw string, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000000Z",
	}
	for i := range layouts {
		if t, err := time.Parse(layouts[i], raw); err == nil {
			return t
		}
	}
	if ts, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if ts > 1e12 {
			return time.UnixMilli(ts)
		}
		return time.Unix(ts, 0)
	}
	return fallback
}

func parseNullableTime(values ...string) *time.Time {
	for i := range values {
		v := strings.TrimSpace(values[i])
		if v == "" {
			continue
		}
		t := parseAnyTime(v, time.Time{})
		if !t.IsZero() {
			tt := t
			return &tt
		}
	}
	return nil
}

func lookupAny(item map[string]interface{}, keys ...string) interface{} {
	for i := range keys {
		key := strings.TrimSpace(keys[i])
		if key == "" {
			continue
		}
		if strings.Contains(key, ".") {
			parts := strings.Split(key, ".")
			var current interface{} = item
			ok := true
			for _, p := range parts {
				m, isMap := current.(map[string]interface{})
				if !isMap {
					ok = false
					break
				}
				next, exists := m[p]
				if !exists {
					ok = false
					break
				}
				current = next
			}
			if ok && current != nil {
				return current
			}
			continue
		}
		if v, ok := item[key]; ok && v != nil {
			return v
		}
	}
	return nil
}

func lookupString(item map[string]interface{}, key string) string {
	return strings.TrimSpace(anyToString(lookupAny(item, key)))
}

func shortHash(raw string, max int) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "_", "")
	if len(v) > max {
		return v[:max]
	}
	return v
}

func normalizeDBProtocol(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "mysql", "mariadb":
		return "mysql"
	case "postgres", "postgresql":
		return "postgres"
	case "redis":
		return "redis"
	case "mongodb", "mongo":
		return "mongodb"
	default:
		return ""
	}
}

func normalizeBaseURL(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	v = strings.TrimRight(v, "/")
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		v = "http://" + v
	}
	return v
}

func normalizeJumpAuthType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case jumpAuthPrivateToken:
		return jumpAuthPrivateToken
	case jumpAuthPassword:
		return jumpAuthPassword
	default:
		return jumpAuthBearerToken
	}
}

func withPagination(path string, limit, offset int) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	query := u.Query()
	if query.Get("limit") == "" {
		query.Set("limit", strconv.Itoa(limit))
	}
	if query.Get("offset") == "" {
		query.Set("offset", strconv.Itoa(offset))
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func buildTargetURL(baseURL, path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func jumpProfileCandidates() []string {
	return []string{
		"/api/v1/users/profile/",
		"/api/v1/users/users/?limit=1",
		"/api/health/",
	}
}

func jumpHostAssetCandidates() []string {
	return []string{
		"/api/v1/assets/hosts/",
		"/api/v1/assets/hosts",
	}
}

func jumpDatabaseAssetCandidates() []string {
	return []string{
		"/api/v1/assets/databases/",
		"/api/v1/assets/databases",
	}
}

func asJumpServerAPIError(err error) *jumpServerAPIError {
	var apiErr *jumpServerAPIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func describeJumpConnectionError(err error) string {
	apiErr := asJumpServerAPIError(err)
	if apiErr == nil {
		return "JumpServer 连接测试失败，请检查地址、认证方式与密钥"
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("JumpServer 认证失败(401)：%s", apiErr.Message)
	case http.StatusForbidden:
		return fmt.Sprintf("JumpServer 权限不足(403)：%s", apiErr.Message)
	case http.StatusNotFound:
		return fmt.Sprintf("JumpServer 接口不存在(404)：%s", apiErr.Path)
	default:
		return fmt.Sprintf("JumpServer API 异常(%d)：%s", apiErr.StatusCode, apiErr.Message)
	}
}

func describeJumpSyncError(err error, resource string) string {
	apiErr := asJumpServerAPIError(err)
	if apiErr == nil {
		return fmt.Sprintf("%s读取失败：%s", resource, err.Error())
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Sprintf("%s读取失败：认证失败(401)，请检查认证方式和密钥 (%s)", resource, apiErr.Message)
	case http.StatusForbidden:
		return fmt.Sprintf("%s读取失败：当前账号无权限(403)，请在 JumpServer 为该账号授权资产读取权限（接口 %s）", resource, apiErr.Path)
	case http.StatusNotFound:
		return fmt.Sprintf("%s读取失败：接口不存在(404)，请确认 JumpServer 版本和API路径（接口 %s）", resource, apiErr.Path)
	default:
		return fmt.Sprintf("%s读取失败：JumpServer API %s 返回 %d (%s)", resource, apiErr.Path, apiErr.StatusCode, apiErr.Message)
	}
}

func anyToString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	default:
		return fmt.Sprint(v)
	}
}

func anyToInt(v interface{}) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

func anyToBool(v interface{}, defaultValue bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(t))
		if err == nil {
			return parsed
		}
	case int:
		return t != 0
	case float64:
		return t != 0
	}
	return defaultValue
}

func firstNonEmpty(item map[string]interface{}, keys ...string) string {
	for i := range keys {
		key := keys[i]
		if strings.EqualFold(key, "protocols") {
			if v, ok := item[key]; ok {
				switch arr := v.(type) {
				case []interface{}:
					for j := range arr {
						if s := strings.TrimSpace(anyToString(arr[j])); s != "" {
							return s
						}
					}
				}
			}
		}
		if s := strings.TrimSpace(anyToString(item[key])); s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func extractRemoteError(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return "unknown error"
	}
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return truncateJumpText(text, 300)
	}
	if m, ok := raw.(map[string]interface{}); ok {
		for _, key := range []string{"detail", "message", "msg", "error"} {
			if v := strings.TrimSpace(anyToString(m[key])); v != "" {
				return truncateJumpText(v, 300)
			}
		}
	}
	return truncateJumpText(text, 300)
}

func nonEmptyError(primary, fallback error) string {
	if primary != nil {
		return primary.Error()
	}
	if fallback != nil {
		return fallback.Error()
	}
	return ""
}
