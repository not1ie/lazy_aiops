package cicd

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListRegistries 镜像仓库列表
func (h *CICDHandler) ListRegistries(c *gin.Context) {
	var list []ImageRegistry
	if err := h.db.Order("updated_at desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	// Hide passwords
	for i := range list {
		list[i].Password = ""
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// GetRegistry 获取单个镜像仓库
func (h *CICDHandler) GetRegistry(c *gin.Context) {
	id := c.Param("id")
	var reg ImageRegistry
	if err := h.db.First(&reg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}
	reg.Password = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": reg})
}

// CreateRegistry 创建镜像仓库
func (h *CICDHandler) CreateRegistry(c *gin.Context) {
	var req ImageRegistry
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	if req.IsDefault {
		h.db.Model(&ImageRegistry{}).Where("1 = 1").Update("is_default", false)
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}
	req.Password = ""
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": req})
}

// UpdateRegistry 更新镜像仓库
func (h *CICDHandler) UpdateRegistry(c *gin.Context) {
	id := c.Param("id")
	var current ImageRegistry
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	var req ImageRegistry
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	if req.IsDefault && !current.IsDefault {
		h.db.Model(&ImageRegistry{}).Where("1 = 1").Update("is_default", false)
	}

	current.Name = req.Name
	current.URL = req.URL
	current.Provider = req.Provider
	current.Username = req.Username
	if req.Password != "" {
		current.Password = req.Password
	}
	current.IsDefault = req.IsDefault
	current.Description = req.Description

	if err := h.db.Save(&current).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteRegistry 删除镜像仓库
func (h *CICDHandler) DeleteRegistry(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&ImageRegistry{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListRegistryTags 获取某个仓库的Tags
func (h *CICDHandler) ListRegistryTags(c *gin.Context) {
	id := c.Param("id")
	repoName := c.Query("repository")
	if repoName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "repository 参数不能为空"})
		return
	}

	var reg ImageRegistry
	if err := h.db.First(&reg, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	tags, err := FetchTags(reg, repoName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tags})
}

// ListCleanRules 镜像清理规则列表
func (h *CICDHandler) ListCleanRules(c *gin.Context) {
	var list []RegistryCleanRule
	if err := h.db.Order("updated_at desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// CreateCleanRule 创建清理规则
func (h *CICDHandler) CreateCleanRule(c *gin.Context) {
	var req RegistryCleanRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": req})
}

// UpdateCleanRule 更新清理规则
func (h *CICDHandler) UpdateCleanRule(c *gin.Context) {
	id := c.Param("id")
	var current RegistryCleanRule
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	var req RegistryCleanRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	current.RegistryID = req.RegistryID
	current.RepositoryName = req.RepositoryName
	current.RetainTags = req.RetainTags
	current.TagRegex = req.TagRegex
	current.Enabled = req.Enabled

	if err := h.db.Save(&current).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCleanRule 删除清理规则
func (h *CICDHandler) DeleteCleanRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&RegistryCleanRule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
