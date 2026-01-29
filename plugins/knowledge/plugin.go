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
	
	p.service = NewKnowledgeService(c.DB, c)
	return nil
}

// ... (Start/Stop/Migrate remain same)

// KnowledgeService
type KnowledgeService struct {
	db   *gorm.DB
	core *core.Core
}

func NewKnowledgeService(db *gorm.DB, c *core.Core) *KnowledgeService {
	return &KnowledgeService{db: db, core: c}
}

func (s *KnowledgeService) Ask(ctx context.Context, question string) (*QAResponse, error) {
	// 1. 简单检索
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
	
	// 2. 调用核心 AI 服务进行 RAG 总结
	var answer string
	if len(docs) == 0 {
		answer = "抱歉，知识库中没有找到与您问题相关的文档。建议您添加相关 Runbook。"
	} else {
		// 构建 Prompt
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
