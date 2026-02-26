package topology

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

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
	var nodes []ServiceNode
	var edges []ServiceEdge
	if err := h.db.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.Find(&edges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	nodeMap := make(map[string]ServiceNode, len(nodes))
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeMap[node.ID] = node
		ids = append(ids, node.ID)
	}

	upstreamSet := make(map[string]map[string]struct{}, len(nodes))
	downstreamSet := make(map[string]map[string]struct{}, len(nodes))
	for _, edge := range edges {
		if edge.SourceID == "" || edge.TargetID == "" || edge.SourceID == edge.TargetID {
			continue
		}
		if _, ok := nodeMap[edge.SourceID]; !ok {
			continue
		}
		if _, ok := nodeMap[edge.TargetID]; !ok {
			continue
		}
		if _, ok := downstreamSet[edge.SourceID]; !ok {
			downstreamSet[edge.SourceID] = map[string]struct{}{}
		}
		if _, ok := upstreamSet[edge.TargetID]; !ok {
			upstreamSet[edge.TargetID] = map[string]struct{}{}
		}
		downstreamSet[edge.SourceID][edge.TargetID] = struct{}{}
		upstreamSet[edge.TargetID][edge.SourceID] = struct{}{}
	}

	results := make([]ServiceDependency, 0, len(nodes))
	for _, node := range nodes {
		upstreamCount := len(upstreamSet[node.ID])
		downstreamCount := len(downstreamSet[node.ID])
		impactScore := downstreamCount*3 + upstreamCount*2
		if strings.EqualFold(node.Type, "gateway") || strings.EqualFold(node.Type, "mq") {
			impactScore += 3
		}
		if node.Status == 2 {
			impactScore += 2
		}
		if node.Status == 3 {
			impactScore += 4
		}
		criticalPath := downstreamCount >= 3 || impactScore >= 12

		dep := ServiceDependency{
			ServiceID:       node.ID,
			ServiceName:     node.Name,
			UpstreamCount:   upstreamCount,
			DownstreamCount: downstreamCount,
			CriticalPath:    criticalPath,
			ImpactScore:     impactScore,
		}

		var existing ServiceDependency
		err := h.db.Where("service_id = ?", node.ID).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if createErr := h.db.Create(&dep).Error; createErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": createErr.Error()})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
		} else if updateErr := h.db.Model(&existing).Updates(map[string]interface{}{
			"service_name":     dep.ServiceName,
			"upstream_count":   dep.UpstreamCount,
			"downstream_count": dep.DownstreamCount,
			"critical_path":    dep.CriticalPath,
			"impact_score":     dep.ImpactScore,
		}).Error; updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": updateErr.Error()})
			return
		}
		results = append(results, dep)
	}

	if len(ids) == 0 {
		h.db.Where("1=1").Delete(&ServiceDependency{})
	} else {
		h.db.Where("service_id NOT IN ?", ids).Delete(&ServiceDependency{})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].ImpactScore == results[j].ImpactScore {
			return results[i].ServiceName < results[j].ServiceName
		}
		return results[i].ImpactScore > results[j].ImpactScore
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "分析完成", "data": results})
}

func (h *TopologyHandler) SyncFromK8s(c *gin.Context) {
	type syncNode struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Namespace   string `json:"namespace"`
		Cluster     string `json:"cluster"`
		Endpoints   string `json:"endpoints"`
		Metadata    string `json:"metadata"`
		Status      int    `json:"status"`
		HealthURL   string `json:"health_url"`
		Description string `json:"description"`
	}
	type syncEdge struct {
		SourceID    string `json:"source_id"`
		TargetID    string `json:"target_id"`
		SourceName  string `json:"source_name"`
		TargetName  string `json:"target_name"`
		Type        string `json:"type"`
		Protocol    string `json:"protocol"`
		Port        int    `json:"port"`
		Description string `json:"description"`
	}
	type syncRequest struct {
		Cluster   string     `json:"cluster"`
		Namespace string     `json:"namespace"`
		Replace   bool       `json:"replace"`
		Nodes     []syncNode `json:"nodes"`
		Edges     []syncEdge `json:"edges"`
	}

	var req syncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if len(req.Nodes) == 0 && len(req.Edges) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未提供可同步的节点或边"})
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": tx.Error.Error()})
		return
	}

	rollback := func(err error) {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
	}

	if req.Replace {
		query := tx.Model(&ServiceNode{})
		if strings.TrimSpace(req.Cluster) != "" {
			query = query.Where("cluster = ?", req.Cluster)
		}
		if strings.TrimSpace(req.Namespace) != "" {
			query = query.Where("namespace = ?", req.Namespace)
		}
		var staleNodes []ServiceNode
		if err := query.Find(&staleNodes).Error; err != nil {
			rollback(err)
			return
		}
		if len(staleNodes) > 0 {
			ids := make([]string, 0, len(staleNodes))
			for _, item := range staleNodes {
				ids = append(ids, item.ID)
			}
			if err := tx.Where("source_id IN ? OR target_id IN ?", ids, ids).Delete(&ServiceEdge{}).Error; err != nil {
				rollback(err)
				return
			}
			if err := tx.Where("service_id IN ?", ids).Delete(&ServiceDependency{}).Error; err != nil {
				rollback(err)
				return
			}
			if err := query.Delete(&ServiceNode{}).Error; err != nil {
				rollback(err)
				return
			}
		}
	}

	nodeCreated := 0
	nodeUpdated := 0
	nodeNameToID := make(map[string]string, len(req.Nodes))

	for _, raw := range req.Nodes {
		name := strings.TrimSpace(raw.Name)
		if name == "" {
			continue
		}
		nodeType := strings.TrimSpace(raw.Type)
		if nodeType == "" {
			nodeType = "service"
		}
		namespace := strings.TrimSpace(raw.Namespace)
		if namespace == "" {
			namespace = strings.TrimSpace(req.Namespace)
		}
		cluster := strings.TrimSpace(raw.Cluster)
		if cluster == "" {
			cluster = strings.TrimSpace(req.Cluster)
		}
		status := raw.Status
		if status < 0 || status > 3 {
			status = 1
		}
		node := ServiceNode{
			Name:        name,
			Type:        nodeType,
			Namespace:   namespace,
			Cluster:     cluster,
			Endpoints:   strings.TrimSpace(raw.Endpoints),
			Metadata:    strings.TrimSpace(raw.Metadata),
			Status:      status,
			HealthURL:   strings.TrimSpace(raw.HealthURL),
			Description: strings.TrimSpace(raw.Description),
		}

		var existing ServiceNode
		err := tx.Where("name = ?", name).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if createErr := tx.Create(&node).Error; createErr != nil {
					rollback(createErr)
					return
				}
				nodeCreated++
				nodeNameToID[name] = node.ID
				continue
			}
			rollback(err)
			return
		}

		updates := map[string]interface{}{
			"type":        node.Type,
			"namespace":   node.Namespace,
			"cluster":     node.Cluster,
			"status":      node.Status,
			"description": node.Description,
		}
		if node.Endpoints != "" {
			updates["endpoints"] = node.Endpoints
		}
		if node.Metadata != "" {
			updates["metadata"] = node.Metadata
		}
		if node.HealthURL != "" {
			updates["health_url"] = node.HealthURL
		}
		if updateErr := tx.Model(&existing).Updates(updates).Error; updateErr != nil {
			rollback(updateErr)
			return
		}
		nodeUpdated++
		nodeNameToID[name] = existing.ID
	}

	var currentNodes []ServiceNode
	if err := tx.Find(&currentNodes).Error; err != nil {
		rollback(err)
		return
	}
	for _, item := range currentNodes {
		nodeNameToID[item.Name] = item.ID
	}

	edgeCreated := 0
	edgeUpdated := 0
	for _, raw := range req.Edges {
		sourceID := strings.TrimSpace(raw.SourceID)
		targetID := strings.TrimSpace(raw.TargetID)
		sourceName := strings.TrimSpace(raw.SourceName)
		targetName := strings.TrimSpace(raw.TargetName)

		if sourceID == "" && sourceName != "" {
			sourceID = nodeNameToID[sourceName]
		}
		if targetID == "" && targetName != "" {
			targetID = nodeNameToID[targetName]
		}
		if sourceID == "" || targetID == "" || sourceID == targetID {
			continue
		}

		if sourceName == "" {
			for name, id := range nodeNameToID {
				if id == sourceID {
					sourceName = name
					break
				}
			}
		}
		if targetName == "" {
			for name, id := range nodeNameToID {
				if id == targetID {
					targetName = name
					break
				}
			}
		}
		if sourceName == "" || targetName == "" {
			continue
		}

		edgeType := strings.TrimSpace(raw.Type)
		if edgeType == "" {
			edgeType = "http"
		}
		protocol := strings.TrimSpace(raw.Protocol)
		if protocol == "" {
			protocol = "HTTP/1.1"
		}

		var existing ServiceEdge
		err := tx.Where("source_id = ? AND target_id = ? AND type = ?", sourceID, targetID, edgeType).
			First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				edge := ServiceEdge{
					SourceID:    sourceID,
					TargetID:    targetID,
					SourceName:  sourceName,
					TargetName:  targetName,
					Type:        edgeType,
					Protocol:    protocol,
					Port:        raw.Port,
					Description: strings.TrimSpace(raw.Description),
				}
				if createErr := tx.Create(&edge).Error; createErr != nil {
					rollback(createErr)
					return
				}
				edgeCreated++
				continue
			}
			rollback(err)
			return
		}

		if updateErr := tx.Model(&existing).Updates(map[string]interface{}{
			"source_name": sourceName,
			"target_name": targetName,
			"protocol":    protocol,
			"port":        raw.Port,
			"description": strings.TrimSpace(raw.Description),
		}).Error; updateErr != nil {
			rollback(updateErr)
			return
		}
		edgeUpdated++
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "同步成功",
		"data": gin.H{
			"nodes_created": nodeCreated,
			"nodes_updated": nodeUpdated,
			"edges_created": edgeCreated,
			"edges_updated": edgeUpdated,
		},
	})
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
	var nodes []ServiceNode
	var edges []ServiceEdge
	if err := h.db.Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.db.Find(&edges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if len(nodes) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "没有可布局节点", "data": gin.H{"updated": 0}})
		return
	}

	nameByID := make(map[string]string, len(nodes))
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nameByID[node.ID] = node.Name
		ids = append(ids, node.ID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return strings.ToLower(nameByID[ids[i]]) < strings.ToLower(nameByID[ids[j]])
	})

	inDegree := make(map[string]int, len(nodes))
	adj := make(map[string][]string, len(nodes))
	for _, id := range ids {
		inDegree[id] = 0
	}
	for _, edge := range edges {
		if edge.SourceID == "" || edge.TargetID == "" || edge.SourceID == edge.TargetID {
			continue
		}
		if _, ok := inDegree[edge.SourceID]; !ok {
			continue
		}
		if _, ok := inDegree[edge.TargetID]; !ok {
			continue
		}
		adj[edge.SourceID] = append(adj[edge.SourceID], edge.TargetID)
		inDegree[edge.TargetID]++
	}
	for source := range adj {
		sort.Slice(adj[source], func(i, j int) bool {
			return strings.ToLower(nameByID[adj[source][i]]) < strings.ToLower(nameByID[adj[source][j]])
		})
	}

	queue := make([]string, 0)
	for _, id := range ids {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	layer := make(map[string]int, len(nodes))
	visited := make(map[string]bool, len(nodes))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited[current] = true
		for _, next := range adj[current] {
			if layer[next] < layer[current]+1 {
				layer[next] = layer[current] + 1
			}
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	maxLayer := 0
	for _, v := range layer {
		if v > maxLayer {
			maxLayer = v
		}
	}
	for _, id := range ids {
		if !visited[id] {
			maxLayer++
			layer[id] = maxLayer
		}
	}

	layerIDs := make(map[int][]string)
	layerKeys := make([]int, 0)
	for _, id := range ids {
		lv := layer[id]
		if _, ok := layerIDs[lv]; !ok {
			layerKeys = append(layerKeys, lv)
		}
		layerIDs[lv] = append(layerIDs[lv], id)
	}
	sort.Ints(layerKeys)
	for _, lv := range layerKeys {
		sort.Slice(layerIDs[lv], func(i, j int) bool {
			return strings.ToLower(nameByID[layerIDs[lv][i]]) < strings.ToLower(nameByID[layerIDs[lv][j]])
		})
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": tx.Error.Error()})
		return
	}

	updated := 0
	for _, lv := range layerKeys {
		for index, id := range layerIDs[lv] {
			x := 120 + lv*360
			y := 80 + index*140
			if err := tx.Model(&ServiceNode{}).Where("id = ?", id).Updates(map[string]interface{}{
				"x": x,
				"y": y,
			}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
			updated++
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("自动布局完成，更新 %d 个节点", updated),
		"data": gin.H{
			"updated":     updated,
			"layer_count": len(layerKeys),
		},
	})
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
