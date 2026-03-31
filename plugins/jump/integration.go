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
	Total     syncStat `json:"total"`
}

type jumpServerClient struct {
	config      *JumpIntegrationConfig
	secret      string
	client      *http.Client
	bearerToken string
	tokenExpiry time.Time
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

	candidates := []string{
		"/api/v1/users/profile/",
		"/api/v1/users/users/?limit=1",
		"/api/health/",
	}

	var endpoint string
	for _, p := range candidates {
		_, _, reqErr := client.request(http.MethodGet, p, nil, true)
		if reqErr == nil {
			endpoint = p
			break
		}
	}
	if endpoint == "" {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "JumpServer 连接测试失败，请检查地址、认证方式与密钥"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "JumpServer 连接成功", "data": gin.H{"checked_endpoint": endpoint}})
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

func (h *JumpHandler) syncFromJumpServerAssets(strict bool) (jumpServerSyncResult, error) {
	client, err := h.newJumpServerClient(strict)
	if err != nil {
		return jumpServerSyncResult{}, err
	}
	if client == nil {
		return jumpServerSyncResult{}, nil
	}

	result := jumpServerSyncResult{}
	hosts, hostErr := client.listAssets("/api/v1/assets/hosts/")
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

	databases, dbErr := client.listAssets("/api/v1/assets/databases/")
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
			_ = h.db.Model(cfg).Updates(map[string]interface{}{
				"last_sync_at":     now,
				"last_sync_status": "ok",
				"last_sync_msg":    fmt.Sprintf("hosts +%d/%d, databases +%d/%d", result.Hosts.Created, result.Hosts.Updated, result.Databases.Created, result.Databases.Updated),
			}).Error
		}
	}

	if hostErr != nil {
		return result, fmt.Errorf("同步 JumpServer 主机失败: %w", hostErr)
	}
	if dbErr != nil {
		return result, fmt.Errorf("同步 JumpServer 数据库失败: %w", dbErr)
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
		return nil, resp.StatusCode, fmt.Errorf("JumpServer API %s %s 失败: %s", method, path, extractRemoteError(body))
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
