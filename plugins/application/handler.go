package application

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppHandler struct {
	db *gorm.DB
}

func NewAppHandler(db *gorm.DB) *AppHandler {
	return &AppHandler{db: db}
}

// ListApps 应用列表
func (h *AppHandler) ListApps(c *gin.Context) {
	var apps []Application
	query := h.db.Model(&Application{})
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	query.Find(&apps)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": apps})
}

// CreateApp 创建应用
func (h *AppHandler) CreateApp(c *gin.Context) {
	var app Application
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.db.Create(&app)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": app})
}

// GetAppConfigs 获取应用环境配置
func (h *AppHandler) GetAppConfigs(c *gin.Context) {
	var configs []AppEnvironment
	h.db.Where("app_id = ?", c.Param("id")).Find(&configs)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// CreateAppConfig 创建环境配置
func (h *AppHandler) CreateAppConfig(c *gin.Context) {
	var conf AppEnvironment
	if err := c.ShouldBindJSON(&conf); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.db.Create(&conf)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conf})
}
