package knowledge

import (
	"context"
	"fmt"
	"net/http"
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
	
	// 复用 AI 配置（如果存在），否则使用默认
	// 注意：在真实微服务架构中，这里应该调用 Core 提供的 AI 接口，但为了解耦，我们这里简单读取配置
	apiKey := ""
	if v, ok := cfg["api_key"].(string); ok {
		apiKey = v
	}
	
p.service = NewKnowledgeService(c.DB, apiKey)
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": docs})
}

func (h *KnowledgeHandler) CreateDoc(c *gin.Context) {
	var doc Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	doc.CreatedBy = c.GetString("username")
	if err := h.db.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) GetDoc(c *gin.Context) {
	var doc Document
	if err := h.db.First(&doc, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文档不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) UpdateDoc(c *gin.Context) {
	var doc Document
	if err := h.db.First(&doc, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文档不存在"})
		return
	}
	
	var updateData Document
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	
doc.Title = updateData.Title
doc.Content = updateData.Content
doc.Tags = updateData.Tags
doc.Category = updateData.Category
	h.db.Save(&doc)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": doc})
}

func (h *KnowledgeHandler) DeleteDoc(c *gin.Context) {
	h.db.Delete(&Document{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *KnowledgeHandler) Ask(c *gin.Context) {
	var req QARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	resp, err := h.service.Ask(context.Background(), req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// Simple Service
type KnowledgeService struct {
	db     *gorm.DB
	apiKey string
}

func NewKnowledgeService(db *gorm.DB, apiKey string) *KnowledgeService {
	return &KnowledgeService{db: db, apiKey: apiKey}
}

// Ask 实现简易的 RAG 流程：
// 1. 关键词检索相关文档
// 2. 组装 Prompt
// 3. 调用 LLM (此处简化为返回检索结果，真实环境需调用 OpenAI)
func (s *KnowledgeService) Ask(ctx context.Context, question string) (*QAResponse, error) {
	// 1. 简单检索 (Poor man's retrieval)
	// 在生产环境中，这里应该使用向量检索 (Vector Search)
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
	
	// 取最相关的前3条
	if err := query.Limit(3).Find(&docs).Error; err != nil {
		return nil, err
	}
	
	// 2. 模拟 AI 回答 (如果没有配置 API Key)
	// 真实场景：在这里调用 s.callLLM(question, docs)
	
	answer := "我找到了以下相关文档，可能对你有帮助：\n\n"
	if len(docs) == 0 {
		answer = "抱歉，知识库中没有找到与您问题相关的文档。建议您添加相关 Runbook。"
	} else {
		for i, doc := range docs {
			answer += fmt.Sprintf("%d. **%s** (分类: %s)\n", i+1, doc.Title, doc.Category)
			// 简单的摘要（取前100字）
			summary := doc.Content
			if len(summary) > 100 {
				summary = summary[:100] + "..."
			}
			answer += fmt.Sprintf("   > %s\n\n", summary)
		}
		answer += "\n(注：配置 OpenAI API Key 后，我可以为您综合总结这些文档的内容)"
	}

	return &QAResponse{
		Answer:     answer,
		References: docs,
	},
	}
}
