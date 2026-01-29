package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RBACHandler struct {
	db *gorm.DB
}

func NewRBACHandler(db *gorm.DB) *RBACHandler {
	return &RBACHandler{db: db}
}

// ========== 用户管理 ==========

// ListUsers 用户列表
func (h *RBACHandler) ListUsers(c *gin.Context) {
	var users []core.User
	query := h.db.Preload("Role")

	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users})
}

// CreateUser 创建用户
func (h *RBACHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		RoleID   string `json:"role_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查用户名是否已存在
	var count int64
	h.db.Model(&core.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 加密密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}

	user := core.User{
		Username: req.Username,
		Password: string(hashedPwd),
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		RoleID:   req.RoleID,
		Status:   1,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 记录操作日志
	h.logOperation(c, "user", "create", user.Username, "创建用户")

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

// GetUser 获取用户详情
func (h *RBACHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	var user core.User
	if err := h.db.Preload("Role").First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

// UpdateUser 更新用户
func (h *RBACHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user core.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		RoleID   string `json:"role_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := map[string]interface{}{
		"nickname": req.Nickname,
		"email":    req.Email,
		"phone":    req.Phone,
		"role_id":  req.RoleID,
	}

	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "user", "update", user.Username, "更新用户信息")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

// DeleteUser 删除用户
func (h *RBACHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	
	// 不能删除admin用户
	var user core.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}
	
	if user.Username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能删除管理员账号"})
		return
	}

	if err := h.db.Delete(&core.User{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "user", "delete", user.Username, "删除用户")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ChangePassword 修改密码
func (h *RBACHandler) ChangePassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var user core.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 验证旧密码（如果提供）
	if req.OldPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "原密码错误"})
			return
		}
	}

	// 加密新密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}

	if err := h.db.Model(&user).Update("password", string(hashedPwd)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "user", "change_password", user.Username, "修改密码")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "密码修改成功"})
}

// ChangeStatus 修改用户状态
func (h *RBACHandler) ChangeStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status int `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var user core.User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	if user.Username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能禁用管理员账号"})
		return
	}

	if err := h.db.Model(&user).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	action := "启用用户"
	if req.Status == 0 {
		action = "禁用用户"
	}
	h.logOperation(c, "user", "change_status", user.Username, action)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "状态修改成功"})
}

// ========== 角色管理 ==========

// ListRoles 角色列表
func (h *RBACHandler) ListRoles(c *gin.Context) {
	var roles []core.Role
	if err := h.db.Preload("Permissions").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": roles})
}

// CreateRole 创建角色
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var role core.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.db.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "role", "create", role.Name, "创建角色")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": role})
}

// GetRole 获取角色详情
func (h *RBACHandler) GetRole(c *gin.Context) {
	id := c.Param("id")
	var role core.Role
	if err := h.db.Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": role})
}

// UpdateRole 更新角色
func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var role core.Role
	if err := h.db.First(&role, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.db.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "role", "update", role.Name, "更新角色")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": role})
}

// DeleteRole 删除角色
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")
	
	var role core.Role
	if err := h.db.First(&role, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}
	
	// 不能删除admin角色
	if role.Code == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能删除管理员角色"})
		return
	}
	
	// 检查是否有用户使用该角色
	var count int64
	h.db.Model(&core.User{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该角色下还有用户，无法删除"})
		return
	}

	if err := h.db.Delete(&core.Role{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "role", "delete", role.Name, "删除角色")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// UpdateRolePermissions 更新角色权限
func (h *RBACHandler) UpdateRolePermissions(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		PermissionIDs []string `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var role core.Role
	if err := h.db.First(&role, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	// 查询权限
	var permissions []*core.Permission
	if len(req.PermissionIDs) > 0 {
		h.db.Where("id IN ?", req.PermissionIDs).Find(&permissions)
	}

	// 更新关联
	if err := h.db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "role", "update_permissions", role.Name, "更新角色权限")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限更新成功"})
}

// ========== 权限管理 ==========

// ListPermissions 权限列表
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	var permissions []core.Permission
	query := h.db.Order("created_at ASC")

	if permType := c.Query("type"); permType != "" {
		query = query.Where("type = ?", permType)
	}

	if err := query.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": permissions})
}

// CreatePermission 创建权限
func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var permission core.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.db.Create(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "permission", "create", permission.Name, "创建权限")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": permission})
}

// GetPermission 获取权限详情
func (h *RBACHandler) GetPermission(c *gin.Context) {
	id := c.Param("id")
	var permission core.Permission
	if err := h.db.First(&permission, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "权限不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": permission})
}

// UpdatePermission 更新权限
func (h *RBACHandler) UpdatePermission(c *gin.Context) {
	id := c.Param("id")
	var permission core.Permission
	if err := h.db.First(&permission, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "权限不存在"})
		return
	}

	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.db.Save(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "permission", "update", permission.Name, "更新权限")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": permission})
}

// DeletePermission 删除权限
func (h *RBACHandler) DeletePermission(c *gin.Context) {
	id := c.Param("id")
	
	var permission core.Permission
	if err := h.db.First(&permission, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "权限不存在"})
		return
	}

	if err := h.db.Delete(&core.Permission{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	h.logOperation(c, "permission", "delete", permission.Name, "删除权限")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// GetPermissionTree 获取权限树
func (h *RBACHandler) GetPermissionTree(c *gin.Context) {
	var permissions []core.Permission
	if err := h.db.Order("created_at ASC").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 构建树形结构
	tree := buildPermissionTree(permissions, "")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tree})
}

// ========== 操作日志 ==========

// ListOperationLogs 操作日志列表
func (h *RBACHandler) ListOperationLogs(c *gin.Context) {
	var logs []core.OperationLog
	query := h.db.Order("created_at DESC")

	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}

	if err := query.Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": logs})
}

// GetOperationLog 获取操作日志详情
func (h *RBACHandler) GetOperationLog(c *gin.Context) {
	id := c.Param("id")
	var log core.OperationLog
	if err := h.db.First(&log, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "日志不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": log})
}

// ========== 辅助函数 ==========

// logOperation 记录操作日志
func (h *RBACHandler) logOperation(c *gin.Context, module, action, target, detail string) {
	log := core.OperationLog{
		UserID:    c.GetString("user_id"),
		Username:  c.GetString("username"),
		Module:    module,
		Action:    action,
		Target:    target,
		Detail:    detail,
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Status:    1,
	}
	h.db.Create(&log)
}

// buildPermissionTree 构建权限树
func buildPermissionTree(permissions []core.Permission, parentID string) []map[string]interface{} {
	tree := []map[string]interface{}{}
	
	for _, perm := range permissions {
		if perm.ParentID == parentID {
			node := map[string]interface{}{
				"id":       perm.ID,
				"name":     perm.Name,
				"code":     perm.Code,
				"type":     perm.Type,
				"children": buildPermissionTree(permissions, perm.ID),
			}
			tree = append(tree, node)
		}
	}
	
	return tree
}
