package system

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemHandler struct {
	db *gorm.DB
}

func NewSystemHandler(db *gorm.DB) *SystemHandler {
	return &SystemHandler{db: db}
}

// ================= Department =================

// ListDepartments 获取部门树 (性能优化：一次查询，内存组装)
func (h *SystemHandler) ListDepartments(c *gin.Context) {
	var depts []Department
	if err := h.db.Order("sort").Find(&depts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": buildDeptTree(depts, "")})
}

func (h *SystemHandler) CreateDepartment(c *gin.Context) {
	var dept Department
	if err := c.ShouldBindJSON(&dept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.db.Create(&dept)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": dept})
}

func (h *SystemHandler) UpdateDepartment(c *gin.Context) {
	var dept Department
	if err := h.db.First(&dept, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not Found"})
		return
	}
	var req Department
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"name":      req.Name,
		"parent_id": req.ParentID,
		"sort":      req.Sort,
		"leader":    req.Leader,
		"phone":     req.Phone,
		"email":     req.Email,
		"status":    req.Status,
	}
	if err := h.db.Model(&dept).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&dept, "id = ?", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": dept})
}

func (h *SystemHandler) DeleteDepartment(c *gin.Context) {
	h.db.Delete(&Department{}, "id = ?", c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Deleted"})
}

// ================= Menu =================

// ListMenus 获取菜单树
func (h *SystemHandler) ListMenus(c *gin.Context) {
	var menus []Menu
	if err := h.db.Order("sort").Find(&menus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": buildMenuTree(menus, "")})
}

func (h *SystemHandler) CreateMenu(c *gin.Context) {
	var menu Menu
	if err := c.ShouldBindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.db.Create(&menu)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": menu})
}

// ================= Post =================

func (h *SystemHandler) ListPosts(c *gin.Context) {
	var posts []Post
	query := h.db.Order("sort ASC, created_at DESC")

	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": posts})
}

func (h *SystemHandler) CreatePost(c *gin.Context) {
	var post Post
	if err := c.ShouldBindJSON(&post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&post).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": post})
}

func (h *SystemHandler) UpdatePost(c *gin.Context) {
	id := c.Param("id")
	var post Post
	if err := h.db.First(&post, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "岗位不存在"})
		return
	}
	var req Post
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"code":        req.Code,
		"sort":        req.Sort,
		"status":      req.Status,
		"description": req.Description,
	}
	if err := h.db.Model(&post).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_ = h.db.First(&post, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": post})
}

func (h *SystemHandler) DeletePost(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Post{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ================= Captcha =================

func (h *SystemHandler) GetCaptchaConfig(c *gin.Context) {
	var cfg CaptchaConfig
	err := h.db.Order("created_at DESC").First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg = CaptchaConfig{
				Enabled:       true,
				Type:          "math",
				Length:        4,
				ExpireSeconds: 120,
				NoiseLevel:    1,
				Background:    "white",
				CaseSensitive: false,
			}
			if createErr := h.db.Create(&cfg).Error; createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": createErr.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
}

func (h *SystemHandler) UpdateCaptchaConfig(c *gin.Context) {
	var req CaptchaConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Length <= 0 {
		req.Length = 4
	}
	if req.ExpireSeconds <= 0 {
		req.ExpireSeconds = 120
	}

	var cfg CaptchaConfig
	err := h.db.Order("created_at DESC").First(&cfg).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		cfg = req
		if createErr := h.db.Create(&cfg).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": createErr.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
		return
	}

	cfg.Enabled = req.Enabled
	cfg.Type = req.Type
	cfg.Length = req.Length
	cfg.ExpireSeconds = req.ExpireSeconds
	cfg.NoiseLevel = req.NoiseLevel
	cfg.Background = req.Background
	cfg.CaseSensitive = req.CaseSensitive

	if saveErr := h.db.Save(&cfg).Error; saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": saveErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cfg})
}

// ================= Login Logs =================

func (h *SystemHandler) ListLoginLogs(c *gin.Context) {
	var logs []LoginLog
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := h.db.Model(&LoginLog{})
	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if ip := c.Query("ip"); ip != "" {
		query = query.Where("ip LIKE ?", "%"+ip+"%")
	}
	if start := c.Query("start_at"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			query = query.Where("login_at >= ?", t)
		}
	}
	if end := c.Query("end_at"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			query = query.Where("login_at <= ?", t)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if err := query.Order("login_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}})
}

// ================= Helpers (Performance Optimized) =================

func buildDeptTree(all []Department, parentID string) []Department {
	var tree []Department
	for _, node := range all {
		if node.ParentID == parentID {
			node.Children = buildDeptTree(all, node.ID)
			tree = append(tree, node)
		}
	}
	return tree
}

func buildMenuTree(all []Menu, parentID string) []Menu {
	var tree []Menu
	for _, node := range all {
		if node.ParentID == parentID {
			node.Children = buildMenuTree(all, node.ID)
			tree = append(tree, node)
		}
	}
	return tree
}
