package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"github.com/lazyautoops/lazy-auto-ops/internal/plugin"
	"gorm.io/gorm"
)

func init() {
	plugin.Register("knowledge", func() plugin.Plugin {
		return &KnowledgePlugin{}
	})
}

type KnowledgePlugin struct {
	core    *core.Core
	cfg     map[string]interface{}
	service *KnowledgeService
}

func (p *KnowledgePlugin) Name() string        { return "knowledge" }
func (p *KnowledgePlugin) Version() string     { return "1.0.0" }
func (p *KnowledgePlugin) Description() string { return "AI 知识库 - 运维经验沉淀与智能检索" }

func (p *KnowledgePlugin) Init(c *core.Core, cfg map[string]interface{}) error {
	p.core = c
	p.cfg = cfg
	
p.service = NewKnowledgeService(c.DB, c)
	return nil
}

func (p *KnowledgePlugin) Start() error { return nil }
func (p *KnowledgePlugin) Stop() error  { return nil }

func (p *KnowledgePlugin) Migrate() error {
	return p.core.DB.AutoMigrate(&Document{})
}

func (p *KnowledgePlugin) RegisterRoutes(g *gin.RouterGroup) {
	h := &KnowledgeHandler{
		db:      p.core.DB,
		service: p.service,
	}

	// 文档管理
	g.GET("/docs", h.ListDocs)
	g.POST("/docs", h.CreateDoc)
	g.GET("/docs/:id", h.GetDoc)
	g.PUT("/docs/:id", h.UpdateDoc)
	g.DELETE("/docs/:id", h.DeleteDoc)

	// 智能问答
	g.POST("/ask", h.Ask)
}

// Handler
type KnowledgeHandler struct {
	db      *gorm.DB
	service *KnowledgeService
}

func (h *KnowledgeHandler) ListDocs(c *gin.Context) {
	var docs []Document
	query := h.db.Model(&Document{})
	
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	
	if err := query.Order("updated_at desc").Find(&docs).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": docs})
}

func (h *KnowledgeHandler) CreateDoc(c *gin.Context) {
	var doc Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	doc.CreatedBy = c.GetString("username")
	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) GetDoc(c *gin.Context) {
	var doc Document
	if err := h.db.First(&doc, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "文档不存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) UpdateDoc(c *gin.Context) {
	var doc Document
	if err := h.db.First(&doc, c.Param("id")).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "message": "文档不存在"})
		return
	}
	
	var updateData Document
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	
doc.Title = updateData.Title
doc.Content = updateData.Content
doc.Tags = updateData.Tags
doc.Category = updateData.Category
	h.db.Save(&doc)
	c.JSON(200, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) DeleteDoc(c *gin.Context) {
	h.db.Delete(&Document{}, c.Param("id"))
	c.JSON(200, gin.H{"code": 0, "message": "删除成功"})
}

func (h *KnowledgeHandler) Ask(c *gin.Context) {
	var req QARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.Ask(context.Background(), req.Question)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 0, "data": resp})
}

// KnowledgeService
type KnowledgeService struct {
	db   *gorm.DB
	core *core.Core
}

func NewKnowledgeService(db *gorm.DB, c *core.Core) *KnowledgeService {
	return &KnowledgeService{db: db, core: c}
}

func (s *KnowledgeService) Ask(ctx context.Context, question string) (*QAResponse, error) {
	var docs []Document
	keywords := strings.Fields(question)
	query := s.db.Model(&Document{})
	
	if len(keywords) > 0 {
		sql := ""
		vars := []interface{}{}
		for i, k := range keywords {
			if i > 0 {
				sql += " OR "
			}
			sql += "title LIKE ? OR content LIKE ? OR tags LIKE ?"
			vars = append(vars, "%"+k+"%", "%"+k+"%", "%"+k+"%")
		}
		query = query.Where(sql, vars...)
	}
	
	if err := query.Limit(3).Find(&docs).Error; err != nil {
		return nil, err
	}
	
	var answer string
	if len(docs) == 0 {
		answer = "抱歉，知识库中没有找到与您问题相关的文档。建议您添加相关 Runbook。"
	} else {
		contextStr := "以下是相关的知识库文档：\n"
		for i, doc := range docs {
			contextStr += fmt.Sprintf("文档%d [%s]:\n%s\n\n", i+1, doc.Title, doc.Content)
		}
		
		prompt := fmt.Sprintf(`请根据提供的知识库文档回答用户的问题。如果文档中没有相关信息，请诚实说明。

%s

用户问题: %s`, contextStr, question)
		
		var err error
		answer, _, err = s.core.AI.CallSimple(prompt)
		if err != nil {
			return nil, err
		}
	}

	return &QAResponse{
		Answer:     answer,
		References: docs,
	},
	nil
}