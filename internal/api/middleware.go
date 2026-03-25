package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/gorm"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(auth *core.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				token = strings.TrimSpace(parts[1])
			}
		}
		isWebSocketRequest := strings.EqualFold(strings.TrimSpace(c.GetHeader("Upgrade")), "websocket")
		// 仅在 WebSocket 场景允许 query token，避免 Token 泄漏到 URL。
		if token == "" && isWebSocketRequest {
			token = strings.TrimSpace(c.Query("token"))
		}
		// 兼容 access_token 参数（仅限 WebSocket 场景）。
		if token == "" && isWebSocketRequest {
			token = strings.TrimSpace(c.Query("access_token"))
		}
		// 兼容通过 Cookie 透传 token 的场景。
		if token == "" {
			if cookieToken, err := c.Cookie("token"); err == nil {
				token = strings.TrimSpace(cookieToken)
			}
		}
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role_code", claims.RoleCode)
		c.Set("force_password_change", claims.ForcePasswordChange)
		c.Next()
	}
}

// ForcePasswordChangeMiddleware 强制默认密码用户先修改密码
func ForcePasswordChangeMiddleware(auth *core.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		forceChange, _ := c.Get("force_password_change")
		force, _ := forceChange.(bool)
		if !force {
			c.Next()
			return
		}

		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
			c.Abort()
			return
		}

		// 放行用户信息和本人改密接口
		path := c.Request.URL.Path
		method := strings.ToUpper(c.Request.Method)
		if method == http.MethodGet && path == "/api/v1/user/info" {
			c.Next()
			return
		}
		if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/rbac/users/") && strings.HasSuffix(path, "/password") && c.Param("id") == userID {
			c.Next()
			return
		}

		need, err := auth.NeedPasswordChange(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "用户状态校验失败"})
			c.Abort()
			return
		}
		if !need {
			c.Next()
			return
		}

		c.JSON(http.StatusPreconditionRequired, gin.H{
			"code":    428,
			"message": "当前账号仍在使用默认密码，请先修改密码后继续操作",
		})
		c.Abort()
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	originMap := make(map[string]struct{})
	allowAny := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAny = true
			continue
		}
		originMap[origin] = struct{}{}
	}
	if len(originMap) == 0 && !allowAny {
		allowAny = true
	}

	return func(c *gin.Context) {
		requestOrigin := strings.TrimSpace(c.GetHeader("Origin"))
		originAllowed := allowAny
		if !allowAny && requestOrigin != "" {
			_, originAllowed = originMap[requestOrigin]
		}

		if allowAny {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if originAllowed && requestOrigin != "" {
			c.Header("Access-Control-Allow-Origin", requestOrigin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if !allowAny {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if requestOrigin != "" && !originAllowed {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "跨域来源未授权"})
			c.Abort()
			return
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// RBACMiddleware 权限控制中间件
func RBACMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleCode := c.GetString("role_code")
		if roleCode == "admin" {
			c.Next()
			return
		}

		required := permissionForRequest(c)
		if required == "" {
			c.Next()
			return
		}

		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证信息"})
			c.Abort()
			return
		}

		if ok, err := hasPermission(db, userID, required); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限校验失败"})
			c.Abort()
			return
		} else if !ok {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OperationLogMiddleware 操作日志中间件
func OperationLogMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if db == nil || !shouldLogRequest(c) {
			return
		}

		module, action := parseModuleAction(c)
		log := core.OperationLog{
			UserID:    c.GetString("user_id"),
			Username:  c.GetString("username"),
			Module:    module,
			Action:    action,
			Target:    c.Request.URL.Path,
			Detail:    c.Request.Method + " " + c.Request.URL.Path,
			IP:        c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Status:    1,
		}
		if c.Writer.Status() >= 400 {
			log.Status = 0
		}
		_ = start // reserved for future duration logging
		db.Create(&log)
	}
}

func shouldLogRequest(c *gin.Context) bool {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/api/v1/login") {
		return false
	}
	if strings.HasPrefix(path, "/api/v1/rbac/logs") {
		return false
	}
	return c.Request.Method != http.MethodGet
}

func parseModuleAction(c *gin.Context) (string, string) {
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/v1/")
	module := strings.SplitN(path, "/", 2)[0]
	action := strings.ToLower(c.Request.Method)
	return module, action
}

func permissionForRequest(c *gin.Context) string {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/api/v1/user") || strings.HasPrefix(path, "/api/v1/plugins") {
		return ""
	}

	if strings.HasPrefix(path, "/api/v1/rbac/") {
		rest := strings.TrimPrefix(path, "/api/v1/rbac/")
		switch {
		case strings.HasPrefix(rest, "users"):
			// 放行本人改密，不要求 system:user 权限
			if strings.EqualFold(c.Request.Method, http.MethodPut) &&
				strings.HasPrefix(path, "/api/v1/rbac/users/") &&
				strings.HasSuffix(path, "/password") &&
				c.Param("id") == c.GetString("user_id") {
				return ""
			}
			return "system:user"
		case strings.HasPrefix(rest, "roles"):
			return "system:role"
		case strings.HasPrefix(rest, "permissions"):
			return "system:permission"
		case strings.HasPrefix(rest, "logs"):
			return "system:log"
		default:
			return "system"
		}
	}

	if strings.HasPrefix(path, "/api/v1/jump/") {
		rest := strings.TrimPrefix(path, "/api/v1/jump/")
		switch {
		case strings.HasPrefix(rest, "assets"),
			strings.HasPrefix(rest, "accounts"),
			strings.HasPrefix(rest, "sync/"):
			return "jump:asset"
		case strings.HasPrefix(rest, "policies"):
			return "jump:policy"
		case strings.HasPrefix(rest, "command-rules"):
			return "jump:rule"
		case strings.HasPrefix(rest, "sessions"):
			return "jump:session"
		default:
			return "jump"
		}
	}

	if strings.HasPrefix(path, "/api/v1/system/") {
		rest := strings.TrimPrefix(path, "/api/v1/system/")
		switch {
		case strings.HasPrefix(rest, "depts"):
			return "system:dept"
		case strings.HasPrefix(rest, "posts"):
			return "system:post"
		case strings.HasPrefix(rest, "captcha"):
			return "system:captcha"
		case strings.HasPrefix(rest, "login-logs"):
			return "system:loginlog"
		case strings.HasPrefix(rest, "menus"):
			return "system:permission"
		default:
			return "system"
		}
	}

	if strings.HasPrefix(path, "/api/v1/orchestrator/") {
		return "workflow"
	}

	module := strings.TrimPrefix(path, "/api/v1/")
	module = strings.SplitN(module, "/", 2)[0]
	if module == "" {
		return ""
	}
	return module
}

func hasPermission(db *gorm.DB, userID, required string) (bool, error) {
	if required == "" {
		return true, nil
	}
	var user core.User
	if err := db.Preload("Role.Permissions").First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	if user.Role == nil {
		return false, nil
	}
	for _, perm := range user.Role.Permissions {
		if perm == nil {
			continue
		}
		if perm.Code == required {
			return true, nil
		}
		if strings.HasPrefix(required, perm.Code+":") {
			return true, nil
		}
	}
	return false, nil
}
