package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListSkills 技能列表
func (h *AIHandler) ListSkills(c *gin.Context) {
	var list []AISkill
	if err := h.db.Order("updated_at desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// GetSkill 获取单个技能
func (h *AIHandler) GetSkill(c *gin.Context) {
	id := c.Param("id")
	var skill AISkill
	if err := h.db.First(&skill, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "技能不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": skill})
}

// CreateSkill 创建技能
func (h *AIHandler) CreateSkill(c *gin.Context) {
	var req AISkill
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	req.IsSystem = false
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": req})
}

// UpdateSkill 更新技能
func (h *AIHandler) UpdateSkill(c *gin.Context) {
	id := c.Param("id")
	var current AISkill
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "技能不存在"})
		return
	}
	if current.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "系统预设技能不可修改"})
		return
	}

	var req AISkill
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	current.Name = req.Name
	current.Description = req.Description
	current.SystemPrompt = req.SystemPrompt
	current.ToolBindings = req.ToolBindings
	current.ParametersSchema = req.ParametersSchema

	if err := h.db.Save(&current).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteSkill 删除技能
func (h *AIHandler) DeleteSkill(c *gin.Context) {
	id := c.Param("id")
	var current AISkill
	if err := h.db.First(&current, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "技能不存在"})
		return
	}
	if current.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "系统预设技能不可删除"})
		return
	}
	if err := h.db.Delete(&AISkill{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// RunSkillReq 执行技能请求
type RunSkillReq struct {
	Parameters map[string]interface{} `json:"parameters"`
}

// RunSkill 执行 AI 技能
func (h *AIHandler) RunSkill(c *gin.Context) {
	id := c.Param("id")
	var skill AISkill
	if err := h.db.First(&skill, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "技能不存在"})
		return
	}

	var req RunSkillReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}

	// 委托给 AIService 执行
	result, err := h.service.RunSkill(&skill, req.Parameters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
