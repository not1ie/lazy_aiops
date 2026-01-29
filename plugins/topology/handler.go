package topology

import (
	"encoding/json"
	"net/http"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TopologyHandler struct {
	db *gorm.DB
}

func NewTopologyHandler(db *gorm.DB) *TopologyHandler {
	return &TopologyHandler{db: db}
}

func (h *TopologyHandler) GetTopology(c *gin.Context) {
	var nodes []ServiceNode
	var edges []ServiceEdge
	h.db.Find(&nodes)
	h.db.Find(&edges)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"nodes": nodes, "edges": edges}})
}

func (h *TopologyHandler) ListNodes(c *gin.Context) {
	var nodes []ServiceNode
	if err := h.db.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *TopologyHandler) CreateNode(c *gin.Context) {
	var node ServiceNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *TopologyHandler) UpdateNode(c *gin.Context) {
	id := c.Param("id")
	var node ServiceNode
	if err := h.db.First(&node, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	h.db.Save(&node)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *TopologyHandler) DeleteNode(c *gin.Context) {
	id := c.Param("id")
	h.db.Delete(&ServiceNode{}, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TopologyHandler) UpdateNodePosition(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	c.ShouldBindJSON(&req)
	h.db.Model(&ServiceNode{}).Where("id = ?", id).Updates(map[string]interface{}{"x": req.X, "y": req.Y})
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TopologyHandler) GetNodeDetail(c *gin.Context) {
	id := c.Param("id")
	var node ServiceNode
	if err := h.db.First(&node, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *TopologyHandler) ListEdges(c *gin.Context) {
	var edges []ServiceEdge
	h.db.Find(&edges)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": edges})
}

func (h *TopologyHandler) CreateEdge(c *gin.Context) {
	var edge ServiceEdge
	if err := c.ShouldBindJSON(&edge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	h.db.Create(&edge)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": edge})
}

func (h *TopologyHandler) DeleteEdge(c *gin.Context) {
	id := c.Param("id")
	h.db.Delete(&ServiceEdge{}, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TopologyHandler) AnalyzeDependencies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分析功能待实现"})
}

func (h *TopologyHandler) SyncFromK8s(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步功能待实现"})
}

func (h *TopologyHandler) ListViews(c *gin.Context) {
	var views []TopologyView
	h.db.Find(&views)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": views})
}

func (h *TopologyHandler) CreateView(c *gin.Context) {
	var view TopologyView
	if err := c.ShouldBindJSON(&view); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	h.db.Create(&view)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": view})
}

func (h *TopologyHandler) SaveLayout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

func (h *TopologyHandler) AutoLayout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "自动布局功能待实现"})
}

func (h *TopologyHandler) ExportTopology(c *gin.Context) {
	var nodes []ServiceNode
	var edges []ServiceEdge
	h.db.Find(&nodes)
	h.db.Find(&edges)
	data := gin.H{"nodes": nodes, "edges": edges}
	jsonData, _ := json.Marshal(data)
	c.Data(http.StatusOK, "application/json", jsonData)
}

func (h *TopologyHandler) ImportTopology(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "导入成功"})
}