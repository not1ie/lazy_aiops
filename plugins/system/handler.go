package system

import (
	"net/http"
	"sort"

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
		c.JSON(404, gin.H{"code": 404, "message": "Not Found"})
		return
	}
	c.ShouldBindJSON(&dept)
	h.db.Save(&dept)
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
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": buildMenuTree(menus, "")})
}

func (h *SystemHandler) CreateMenu(c *gin.Context) {
	var menu Menu
	if err := c.ShouldBindJSON(&menu); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.db.Create(&menu)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": menu})
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
