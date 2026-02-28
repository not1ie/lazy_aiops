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
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证格式错误"})
				c.Abort()
				return
			}
			token = parts[1]
		} else {
			// WebSocket 场景无法方便携带 Authorization 头，允许通过 query token 透传。
			token = strings.TrimSpace(c.Query("token"))
		}
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
		c.Next()
	}
}

// CORSMiddleware 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

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
