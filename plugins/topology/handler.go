package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/docker"
	k8splugin "github.com/lazyautoops/lazy-auto-ops/plugins/k8s"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type TopologyHandler struct {
	db *gorm.DB
}

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

type syncStats struct {
	NodesCreated int `json:"nodes_created"`
	NodesUpdated int `json:"nodes_updated"`
	EdgesCreated int `json:"edges_created"`
	EdgesUpdated int `json:"edges_updated"`
}

type envDependencyHint struct {
	Key      string `json:"key"`
	Host     string `json:"host"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type discoverRequest struct {
	Sources       []string `json:"sources"`
	Replace       bool     `json:"replace"`
	AutoLayout    bool     `json:"auto_layout"`
	ClusterIDs    []string `json:"cluster_ids"`
	DockerHostIDs []string `json:"docker_host_ids"`
	Namespace     string   `json:"namespace"`
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
	var req syncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if len(req.Nodes) == 0 && len(req.Edges) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未提供可同步的节点或边"})
		return
	}

	stats, err := h.applySync(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "同步成功",
		"data":    stats,
	})
}

func (h *TopologyHandler) applySync(req syncRequest) (*syncStats, error) {

	tx := h.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	rollback := func(err error) error {
		tx.Rollback()
		return err
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
			return nil, rollback(err)
		}
		if len(staleNodes) > 0 {
			ids := make([]string, 0, len(staleNodes))
			for _, item := range staleNodes {
				ids = append(ids, item.ID)
			}
			if err := tx.Unscoped().Where("source_id IN ? OR target_id IN ?", ids, ids).Delete(&ServiceEdge{}).Error; err != nil {
				return nil, rollback(err)
			}
			if err := tx.Unscoped().Where("service_id IN ?", ids).Delete(&ServiceDependency{}).Error; err != nil {
				return nil, rollback(err)
			}
			// 这里必须硬删除，否则软删除记录仍会占用 name 唯一索引，下一次自动发现会触发 UNIQUE 约束错误。
			if err := query.Unscoped().Delete(&ServiceNode{}).Error; err != nil {
				return nil, rollback(err)
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
		err := tx.Unscoped().Where("name = ?", name).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if createErr := tx.Create(&node).Error; createErr != nil {
					return nil, rollback(createErr)
				}
				nodeCreated++
				nodeNameToID[name] = node.ID
				continue
			}
			return nil, rollback(err)
		}

		updates := map[string]interface{}{
			"type":        node.Type,
			"namespace":   node.Namespace,
			"cluster":     node.Cluster,
			"status":      node.Status,
			"description": node.Description,
			"deleted_at":  nil,
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
		if updateErr := tx.Unscoped().Model(&ServiceNode{}).Where("id = ?", existing.ID).Updates(updates).Error; updateErr != nil {
			return nil, rollback(updateErr)
		}
		nodeUpdated++
		nodeNameToID[name] = existing.ID
	}

	var currentNodes []ServiceNode
	if err := tx.Find(&currentNodes).Error; err != nil {
		return nil, rollback(err)
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
					return nil, rollback(createErr)
				}
				edgeCreated++
				continue
			}
			return nil, rollback(err)
		}

		if updateErr := tx.Model(&existing).Updates(map[string]interface{}{
			"source_name": sourceName,
			"target_name": targetName,
			"protocol":    protocol,
			"port":        raw.Port,
			"description": strings.TrimSpace(raw.Description),
		}).Error; updateErr != nil {
			return nil, rollback(updateErr)
		}
		edgeUpdated++
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &syncStats{
		NodesCreated: nodeCreated,
		NodesUpdated: nodeUpdated,
		EdgesCreated: edgeCreated,
		EdgesUpdated: edgeUpdated,
	}, nil
}

func (h *TopologyHandler) Discover(c *gin.Context) {
	var req discoverRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if len(req.Sources) == 0 {
		req.Sources = []string{"k8s", "swarm", "docker"}
	}

	sourceSet := make(map[string]bool, len(req.Sources))
	for _, raw := range req.Sources {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		sourceSet[key] = true
	}
	if len(sourceSet) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "至少选择一个发现源"})
		return
	}

	clusterSet := toSet(req.ClusterIDs)
	hostSet := toSet(req.DockerHostIDs)

	nodes := make([]syncNode, 0, 256)
	edges := make([]syncEdge, 0, 512)
	nodeSeen := map[string]struct{}{}
	edgeSeen := map[string]struct{}{}
	warnings := make([]string, 0)
	detail := gin.H{}

	if sourceSet["k8s"] {
		k8sNodes, k8sEdges, k8sMeta, k8sWarnings := h.discoverFromK8s(clusterSet, req.Namespace)
		nodes, edges = mergeDiscovered(nodes, edges, nodeSeen, edgeSeen, k8sNodes, k8sEdges)
		warnings = append(warnings, k8sWarnings...)
		detail["k8s"] = k8sMeta
	}
	if sourceSet["swarm"] || sourceSet["docker"] {
		dockerNodes, dockerEdges, dockerMeta, dockerWarnings := h.discoverFromDocker(hostSet, sourceSet["swarm"], sourceSet["docker"])
		nodes, edges = mergeDiscovered(nodes, edges, nodeSeen, edgeSeen, dockerNodes, dockerEdges)
		warnings = append(warnings, dockerWarnings...)
		detail["docker"] = dockerMeta
	}

	inferredEdges, inferMeta := inferDependencyEdges(nodes, edges)
	if len(inferredEdges) > 0 {
		nodes, edges = mergeDiscovered(nodes, edges, nodeSeen, edgeSeen, nil, inferredEdges)
	}
	detail["inference"] = inferMeta

	if len(nodes) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "自动发现完成，但未发现可同步节点",
			"data": gin.H{
				"discovered_nodes": 0,
				"discovered_edges": 0,
				"warnings":         warnings,
				"detail":           detail,
			},
		})
		return
	}

	if req.Replace {
		if err := h.cleanupManagedAutoNodes(sourceSet); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	stats, err := h.applySync(syncRequest{
		Replace: false,
		Nodes:   nodes,
		Edges:   edges,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if req.AutoLayout {
		if _, _, err := h.autoLayoutInternal(); err != nil {
			warnings = append(warnings, "自动布局失败: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "自动发现完成",
		"data": gin.H{
			"discovered_nodes": len(nodes),
			"discovered_edges": len(edges),
			"sync":             stats,
			"warnings":         warnings,
			"detail":           detail,
		},
	})
}

type workloadRef struct {
	NodeName   string
	Namespace  string
	Labels     map[string]string
	Cluster    string
	ClusterID  string
	Resource   string
	ResourceNS string
}

func (h *TopologyHandler) discoverFromK8s(clusterSet map[string]struct{}, namespace string) ([]syncNode, []syncEdge, gin.H, []string) {
	var clusters []k8splugin.Cluster
	query := h.db.Model(&k8splugin.Cluster{})
	if len(clusterSet) > 0 {
		ids := make([]string, 0, len(clusterSet))
		for id := range clusterSet {
			ids = append(ids, id)
		}
		query = query.Where("id IN ?", ids)
	}
	if err := query.Find(&clusters).Error; err != nil {
		return nil, nil, gin.H{"error": err.Error()}, []string{fmt.Sprintf("K8s集群查询失败: %v", err)}
	}

	nodes := make([]syncNode, 0, 256)
	edges := make([]syncEdge, 0, 512)
	workloads := make([]workloadRef, 0, 512)
	warnings := make([]string, 0)
	metric := gin.H{
		"clusters":               len(clusters),
		"workloads":              0,
		"services":               0,
		"ingresses":              0,
		"workload_service_edges": 0,
		"ingress_service_edges":  0,
	}

	nsQuery := strings.TrimSpace(namespace)
	if nsQuery == "" {
		nsQuery = metav1.NamespaceAll
	}
	if nsQuery == "*" || strings.EqualFold(nsQuery, "all") {
		nsQuery = metav1.NamespaceAll
	}

	for _, cluster := range clusters {
		restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("K8s[%s] kubeconfig无效: %v", cluster.Name, err))
			continue
		}
		client, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("K8s[%s] 客户端创建失败: %v", cluster.Name, err))
			continue
		}

		clusterName := strings.TrimSpace(cluster.DisplayName)
		if clusterName == "" {
			clusterName = strings.TrimSpace(cluster.Name)
		}
		if clusterName == "" {
			clusterName = cluster.ID
		}

		kinds := []struct {
			kind string
			list func() ([]workloadRef, error)
		}{
			{
				kind: "deployment",
				list: func() ([]workloadRef, error) {
					items, err := client.AppsV1().Deployments(nsQuery).List(context.Background(), metav1.ListOptions{})
					if err != nil {
						return nil, err
					}
					result := make([]workloadRef, 0, len(items.Items))
					for _, item := range items.Items {
						envHints := collectContainerDependencyHints(item.Spec.Template.Spec.Containers)
						name := fmt.Sprintf("k8s/%s/%s/deployment/%s", clusterName, item.Namespace, item.Name)
						nodes = append(nodes, syncNode{
							Name:      name,
							Type:      "service",
							Namespace: item.Namespace,
							Cluster:   clusterName,
							Endpoints: jsonArrayString(containerPorts(item.Spec.Template.Spec.Containers)),
							Metadata: jsonObjectString(gin.H{
								"managed_by":      "auto-discover",
								"topology_source": "k8s",
								"cluster_id":      cluster.ID,
								"resource":        "Deployment",
								"resource_name":   item.Name,
								"namespace":       item.Namespace,
								"labels":          item.Labels,
								"dep_hints":       envHints,
							}),
							Status:      1,
							Description: fmt.Sprintf("Deployment %s/%s", item.Namespace, item.Name),
						})
						result = append(result, workloadRef{
							NodeName:  name,
							Namespace: item.Namespace,
							Labels:    item.Spec.Template.Labels,
							Cluster:   clusterName,
							ClusterID: cluster.ID,
							Resource:  "Deployment",
						})
					}
					return result, nil
				},
			},
			{
				kind: "statefulset",
				list: func() ([]workloadRef, error) {
					items, err := client.AppsV1().StatefulSets(nsQuery).List(context.Background(), metav1.ListOptions{})
					if err != nil {
						return nil, err
					}
					result := make([]workloadRef, 0, len(items.Items))
					for _, item := range items.Items {
						envHints := collectContainerDependencyHints(item.Spec.Template.Spec.Containers)
						name := fmt.Sprintf("k8s/%s/%s/statefulset/%s", clusterName, item.Namespace, item.Name)
						nodes = append(nodes, syncNode{
							Name:      name,
							Type:      "service",
							Namespace: item.Namespace,
							Cluster:   clusterName,
							Endpoints: jsonArrayString(containerPorts(item.Spec.Template.Spec.Containers)),
							Metadata: jsonObjectString(gin.H{
								"managed_by":      "auto-discover",
								"topology_source": "k8s",
								"cluster_id":      cluster.ID,
								"resource":        "StatefulSet",
								"resource_name":   item.Name,
								"namespace":       item.Namespace,
								"labels":          item.Labels,
								"dep_hints":       envHints,
							}),
							Status:      1,
							Description: fmt.Sprintf("StatefulSet %s/%s", item.Namespace, item.Name),
						})
						result = append(result, workloadRef{
							NodeName:  name,
							Namespace: item.Namespace,
							Labels:    item.Spec.Template.Labels,
							Cluster:   clusterName,
							ClusterID: cluster.ID,
							Resource:  "StatefulSet",
						})
					}
					return result, nil
				},
			},
			{
				kind: "daemonset",
				list: func() ([]workloadRef, error) {
					items, err := client.AppsV1().DaemonSets(nsQuery).List(context.Background(), metav1.ListOptions{})
					if err != nil {
						return nil, err
					}
					result := make([]workloadRef, 0, len(items.Items))
					for _, item := range items.Items {
						envHints := collectContainerDependencyHints(item.Spec.Template.Spec.Containers)
						name := fmt.Sprintf("k8s/%s/%s/daemonset/%s", clusterName, item.Namespace, item.Name)
						nodes = append(nodes, syncNode{
							Name:      name,
							Type:      "service",
							Namespace: item.Namespace,
							Cluster:   clusterName,
							Endpoints: jsonArrayString(containerPorts(item.Spec.Template.Spec.Containers)),
							Metadata: jsonObjectString(gin.H{
								"managed_by":      "auto-discover",
								"topology_source": "k8s",
								"cluster_id":      cluster.ID,
								"resource":        "DaemonSet",
								"resource_name":   item.Name,
								"namespace":       item.Namespace,
								"labels":          item.Labels,
								"dep_hints":       envHints,
							}),
							Status:      1,
							Description: fmt.Sprintf("DaemonSet %s/%s", item.Namespace, item.Name),
						})
						result = append(result, workloadRef{
							NodeName:  name,
							Namespace: item.Namespace,
							Labels:    item.Spec.Template.Labels,
							Cluster:   clusterName,
							ClusterID: cluster.ID,
							Resource:  "DaemonSet",
						})
					}
					return result, nil
				},
			},
		}

		for _, item := range kinds {
			listed, err := item.list()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("K8s[%s] 拉取%s失败: %v", clusterName, item.kind, err))
				continue
			}
			workloads = append(workloads, listed...)
			metric["workloads"] = metric["workloads"].(int) + len(listed)
		}

		services, err := client.CoreV1().Services(nsQuery).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("K8s[%s] 拉取Service失败: %v", clusterName, err))
		} else {
			for _, svc := range services.Items {
				svcNode := fmt.Sprintf("k8s/%s/%s/service/%s", clusterName, svc.Namespace, svc.Name)
				ports := make([]string, 0, len(svc.Spec.Ports))
				for _, p := range svc.Spec.Ports {
					ports = append(ports, fmt.Sprintf("%d/%s", p.Port, strings.ToUpper(string(p.Protocol))))
				}
				nodeType := "service"
				if svc.Spec.Type == corev1.ServiceTypeNodePort || svc.Spec.Type == corev1.ServiceTypeLoadBalancer || svc.Spec.Type == corev1.ServiceTypeExternalName {
					nodeType = "gateway"
				}
				nodes = append(nodes, syncNode{
					Name:      svcNode,
					Type:      nodeType,
					Namespace: svc.Namespace,
					Cluster:   clusterName,
					Endpoints: jsonArrayString(appendServiceEndpoints(svc.Spec.ClusterIP, ports)),
					Metadata: jsonObjectString(gin.H{
						"managed_by":      "auto-discover",
						"topology_source": "k8s",
						"cluster_id":      cluster.ID,
						"resource":        "Service",
						"resource_name":   svc.Name,
						"namespace":       svc.Namespace,
						"selector":        svc.Spec.Selector,
					}),
					Status:      1,
					Description: fmt.Sprintf("Service %s/%s", svc.Namespace, svc.Name),
				})
				metric["services"] = metric["services"].(int) + 1

				if len(svc.Spec.Selector) == 0 {
					continue
				}
				for _, wk := range workloads {
					if wk.ClusterID != cluster.ID || wk.Namespace != svc.Namespace {
						continue
					}
					if !matchSelector(svc.Spec.Selector, wk.Labels) {
						continue
					}
					edges = append(edges, syncEdge{
						SourceName:  svcNode,
						TargetName:  wk.NodeName,
						Type:        "http",
						Protocol:    "HTTP/1.1",
						Port:        firstServicePort(svc.Spec.Ports),
						Description: "service selector route",
					})
					metric["workload_service_edges"] = metric["workload_service_edges"].(int) + 1
				}
			}
		}

		ingresses, err := client.NetworkingV1().Ingresses(nsQuery).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("K8s[%s] 拉取Ingress失败: %v", clusterName, err))
		} else {
			for _, ing := range ingresses.Items {
				ingNode := fmt.Sprintf("k8s/%s/%s/ingress/%s", clusterName, ing.Namespace, ing.Name)
				hosts := ingressHosts(ing)
				nodes = append(nodes, syncNode{
					Name:      ingNode,
					Type:      "gateway",
					Namespace: ing.Namespace,
					Cluster:   clusterName,
					Endpoints: jsonArrayString(hosts),
					Metadata: jsonObjectString(gin.H{
						"managed_by":      "auto-discover",
						"topology_source": "k8s",
						"cluster_id":      cluster.ID,
						"resource":        "Ingress",
						"resource_name":   ing.Name,
						"namespace":       ing.Namespace,
					}),
					Status:      1,
					Description: fmt.Sprintf("Ingress %s/%s", ing.Namespace, ing.Name),
				})
				metric["ingresses"] = metric["ingresses"].(int) + 1

				for _, edge := range ingressToServiceEdges(clusterName, ingNode, ing) {
					edges = append(edges, edge)
					metric["ingress_service_edges"] = metric["ingress_service_edges"].(int) + 1
				}
			}
		}
	}

	return nodes, edges, metric, warnings
}

func (h *TopologyHandler) discoverFromDocker(hostSet map[string]struct{}, includeSwarm, includeDocker bool) ([]syncNode, []syncEdge, gin.H, []string) {
	var hosts []docker.DockerHost
	query := h.db.Model(&docker.DockerHost{})
	if len(hostSet) > 0 {
		ids := make([]string, 0, len(hostSet))
		for id := range hostSet {
			ids = append(ids, id)
		}
		query = query.Where("id IN ?", ids)
	}
	if err := query.Find(&hosts).Error; err != nil {
		return nil, nil, gin.H{"error": err.Error()}, []string{fmt.Sprintf("Docker主机查询失败: %v", err)}
	}

	nodes := make([]syncNode, 0, 128)
	edges := make([]syncEdge, 0, 256)
	warnings := make([]string, 0)
	metric := gin.H{
		"hosts":             len(hosts),
		"swarm_nodes":       0,
		"swarm_services":    0,
		"docker_containers": 0,
	}

	for _, host := range hosts {
		executor, err := docker.GetExecutorByHost(h.db, host.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Docker[%s] 连接失败: %v", host.Name, err))
			continue
		}
		hostName := strings.TrimSpace(host.Name)
		if hostName == "" {
			hostName = host.ID
		}

		hostNode := fmt.Sprintf("docker/%s/host/%s", hostName, host.ID[:8])
		nodes = append(nodes, syncNode{
			Name:      hostNode,
			Type:      "gateway",
			Cluster:   hostName,
			Namespace: "docker",
			Metadata: jsonObjectString(gin.H{
				"managed_by":      "auto-discover",
				"topology_source": "docker",
				"resource":        "Host",
				"docker_host_id":  host.ID,
			}),
			Status:      1,
			Description: fmt.Sprintf("Docker Host %s", hostName),
		})
		metric["swarm_nodes"] = metric["swarm_nodes"].(int) + 1

		if includeSwarm {
			swarmOut, swarmErr, swarmExecErr := executor.Execute("docker info --format '{{json .Swarm}}'")
			if swarmExecErr == nil && strings.Contains(strings.ToLower(swarmOut), "\"localnodestate\":\"active\"") {
				serviceOut, serviceErr, serviceExecErr := executor.Execute("docker service ls --format '{{.Name}}||{{.Image}}||{{.Ports}}'")
				if serviceExecErr != nil {
					msg := strings.TrimSpace(serviceErr)
					if msg == "" {
						msg = serviceExecErr.Error()
					}
					warnings = append(warnings, fmt.Sprintf("Swarm[%s] 读取服务失败: %s", hostName, msg))
				} else {
					for _, line := range splitLines(serviceOut) {
						parts := strings.Split(line, "||")
						if len(parts) < 1 {
							continue
						}
						serviceName := strings.TrimSpace(parts[0])
						if serviceName == "" {
							continue
						}
						image := ""
						if len(parts) > 1 {
							image = strings.TrimSpace(parts[1])
						}
						ports := ""
						if len(parts) > 2 {
							ports = strings.TrimSpace(parts[2])
						}

						serviceNode := fmt.Sprintf("swarm/%s/service/%s", hostName, serviceName)
						serviceHints, hintErr := inspectSwarmServiceDependencyHints(executor, serviceName)
						if hintErr != nil {
							warnings = append(warnings, fmt.Sprintf("Swarm[%s] 服务 %s 依赖提取失败: %v", hostName, serviceName, hintErr))
						}
						nodes = append(nodes, syncNode{
							Name:      serviceNode,
							Type:      "service",
							Namespace: "swarm",
							Cluster:   hostName,
							Endpoints: jsonArrayString(nonEmptyValues(ports)),
							Metadata: jsonObjectString(gin.H{
								"managed_by":      "auto-discover",
								"topology_source": "swarm",
								"resource":        "Service",
								"docker_host_id":  host.ID,
								"service_name":    serviceName,
								"image":           image,
								"dep_hints":       serviceHints,
							}),
							Status:      1,
							Description: fmt.Sprintf("Swarm Service %s", serviceName),
						})
						edges = append(edges, syncEdge{
							SourceName:  hostNode,
							TargetName:  serviceNode,
							Type:        "tcp",
							Protocol:    "TCP",
							Description: "scheduled on host",
						})
						metric["swarm_services"] = metric["swarm_services"].(int) + 1
					}
				}
			} else if swarmExecErr != nil {
				msg := strings.TrimSpace(swarmErr)
				if msg == "" {
					msg = swarmExecErr.Error()
				}
				warnings = append(warnings, fmt.Sprintf("Swarm[%s] 识别失败: %s", hostName, msg))
			}
		}

		if includeDocker {
			containerOut, containerErr, containerExecErr := executor.Execute("docker ps --format '{{.Names}}||{{.Image}}||{{.Ports}}'")
			if containerExecErr != nil {
				msg := strings.TrimSpace(containerErr)
				if msg == "" {
					msg = containerExecErr.Error()
				}
				warnings = append(warnings, fmt.Sprintf("Docker[%s] 读取容器失败: %s", hostName, msg))
				continue
			}

			for _, line := range splitLines(containerOut) {
				parts := strings.Split(line, "||")
				if len(parts) < 1 {
					continue
				}
				name := strings.TrimSpace(parts[0])
				if name == "" {
					continue
				}
				if strings.Count(name, ".") >= 2 {
					// Swarm task container, avoid duplicate with service nodes.
					continue
				}
				image := ""
				if len(parts) > 1 {
					image = strings.TrimSpace(parts[1])
				}
				ports := ""
				if len(parts) > 2 {
					ports = strings.TrimSpace(parts[2])
				}
				containerNode := fmt.Sprintf("docker/%s/container/%s", hostName, name)
				containerHints, hintErr := inspectContainerDependencyHints(executor, name)
				if hintErr != nil {
					warnings = append(warnings, fmt.Sprintf("Docker[%s] 容器 %s 依赖提取失败: %v", hostName, name, hintErr))
				}
				nodes = append(nodes, syncNode{
					Name:      containerNode,
					Type:      "service",
					Namespace: "docker",
					Cluster:   hostName,
					Endpoints: jsonArrayString(nonEmptyValues(ports)),
					Metadata: jsonObjectString(gin.H{
						"managed_by":      "auto-discover",
						"topology_source": "docker",
						"resource":        "Container",
						"docker_host_id":  host.ID,
						"container_name":  name,
						"image":           image,
						"dep_hints":       containerHints,
					}),
					Status:      1,
					Description: fmt.Sprintf("Docker Container %s", name),
				})
				edges = append(edges, syncEdge{
					SourceName:  hostNode,
					TargetName:  containerNode,
					Type:        "tcp",
					Protocol:    "TCP",
					Description: "runs on host",
				})
				metric["docker_containers"] = metric["docker_containers"].(int) + 1
			}
		}
	}

	return nodes, edges, metric, warnings
}

func (h *TopologyHandler) cleanupManagedAutoNodes(sourceSet map[string]bool) error {
	var nodes []ServiceNode
	if err := h.db.Find(&nodes).Error; err != nil {
		return err
	}
	ids := make([]string, 0)
	for _, node := range nodes {
		if strings.TrimSpace(node.Metadata) == "" {
			continue
		}
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(node.Metadata), &meta); err != nil {
			continue
		}
		managedBy, _ := meta["managed_by"].(string)
		source, _ := meta["topology_source"].(string)
		if managedBy != "auto-discover" {
			continue
		}
		if source == "" || sourceSet[source] {
			ids = append(ids, node.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	tx := h.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("source_id IN ? OR target_id IN ?", ids, ids).Delete(&ServiceEdge{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("service_id IN ?", ids).Delete(&ServiceDependency{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("id IN ?", ids).Delete(&ServiceNode{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func mergeDiscovered(
	baseNodes []syncNode,
	baseEdges []syncEdge,
	nodeSeen map[string]struct{},
	edgeSeen map[string]struct{},
	appendNodes []syncNode,
	appendEdges []syncEdge,
) ([]syncNode, []syncEdge) {
	for _, node := range appendNodes {
		key := strings.TrimSpace(node.Name)
		if key == "" {
			continue
		}
		if _, exists := nodeSeen[key]; exists {
			continue
		}
		nodeSeen[key] = struct{}{}
		baseNodes = append(baseNodes, node)
	}
	for _, edge := range appendEdges {
		source := strings.TrimSpace(edge.SourceName)
		target := strings.TrimSpace(edge.TargetName)
		typ := strings.TrimSpace(edge.Type)
		if source == "" || target == "" {
			continue
		}
		key := source + "->" + target + "|" + typ
		if _, exists := edgeSeen[key]; exists {
			continue
		}
		edgeSeen[key] = struct{}{}
		baseEdges = append(baseEdges, edge)
	}
	return baseNodes, baseEdges
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func splitLines(raw string) []string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func nonEmptyValues(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func jsonObjectString(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func jsonArrayString(values []string) string {
	raw, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func appendServiceEndpoints(clusterIP string, ports []string) []string {
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		if strings.TrimSpace(clusterIP) != "" && clusterIP != "None" {
			out = append(out, fmt.Sprintf("%s:%s", clusterIP, p))
		} else {
			out = append(out, p)
		}
	}
	return out
}

func firstServicePort(ports []corev1.ServicePort) int {
	if len(ports) == 0 {
		return 0
	}
	return int(ports[0].Port)
}

func containerPorts(containers []corev1.Container) []string {
	out := make([]string, 0)
	for _, c := range containers {
		for _, p := range c.Ports {
			out = append(out, fmt.Sprintf("%s:%d/%s", c.Name, p.ContainerPort, strings.ToUpper(string(p.Protocol))))
		}
	}
	return out
}

func ingressHosts(ing networkingv1.Ingress) []string {
	values := make([]string, 0, len(ing.Spec.Rules))
	for _, rule := range ing.Spec.Rules {
		host := strings.TrimSpace(rule.Host)
		if host != "" {
			values = append(values, host)
		}
	}
	if len(values) == 0 && ing.Spec.DefaultBackend != nil {
		values = append(values, "default-backend")
	}
	return values
}

func ingressToServiceEdges(clusterName, ingressNode string, ing networkingv1.Ingress) []syncEdge {
	edges := make([]syncEdge, 0)
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				continue
			}
			target := fmt.Sprintf("k8s/%s/%s/service/%s", clusterName, ing.Namespace, path.Backend.Service.Name)
			port := 0
			if path.Backend.Service.Port.Number > 0 {
				port = int(path.Backend.Service.Port.Number)
			}
			edges = append(edges, syncEdge{
				SourceName:  ingressNode,
				TargetName:  target,
				Type:        "http",
				Protocol:    "HTTP",
				Port:        port,
				Description: "ingress route",
			})
		}
	}
	if len(edges) == 0 && ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		target := fmt.Sprintf("k8s/%s/%s/service/%s", clusterName, ing.Namespace, ing.Spec.DefaultBackend.Service.Name)
		edges = append(edges, syncEdge{
			SourceName:  ingressNode,
			TargetName:  target,
			Type:        "http",
			Protocol:    "HTTP",
			Port:        int(ing.Spec.DefaultBackend.Service.Port.Number),
			Description: "default backend",
		})
	}
	return edges
}

func matchSelector(selector map[string]string, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for key, val := range selector {
		if labels == nil {
			return false
		}
		if labels[key] != val {
			return false
		}
	}
	return true
}

func collectContainerDependencyHints(containers []corev1.Container) []envDependencyHint {
	envItems := make([]string, 0, 64)
	for _, container := range containers {
		for _, item := range container.Env {
			if strings.TrimSpace(item.Value) == "" {
				continue
			}
			envItems = append(envItems, fmt.Sprintf("%s=%s", item.Name, item.Value))
		}
	}
	return parseEnvDependencyHints(envItems)
}

func inspectSwarmServiceDependencyHints(executor docker.CommandExecutor, serviceName string) ([]envDependencyHint, error) {
	cmd := fmt.Sprintf("docker service inspect %s --format '{{json .Spec.TaskTemplate.ContainerSpec.Env}}'", shellQuote(serviceName))
	output, stderr, err := executor.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return parseEnvHintsFromJSON(output)
}

func inspectContainerDependencyHints(executor docker.CommandExecutor, containerName string) ([]envDependencyHint, error) {
	cmd := fmt.Sprintf("docker inspect %s --format '{{json .Config.Env}}'", shellQuote(containerName))
	output, stderr, err := executor.Execute(cmd)
	if err != nil {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return parseEnvHintsFromJSON(output)
}

func parseEnvHintsFromJSON(raw string) ([]envDependencyHint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	var envItems []string
	if err := json.Unmarshal([]byte(trimmed), &envItems); err != nil {
		return nil, err
	}
	return parseEnvDependencyHints(envItems), nil
}

func parseEnvDependencyHints(items []string) []envDependencyHint {
	const maxHints = 24
	hints := make([]envDependencyHint, 0, maxHints)
	seen := map[string]struct{}{}
	for _, item := range items {
		idx := strings.Index(item, "=")
		if idx <= 0 || idx == len(item)-1 {
			continue
		}
		key := strings.TrimSpace(item[:idx])
		value := strings.TrimSpace(item[idx+1:])
		if key == "" || value == "" {
			continue
		}
		if !isDependencyEnvKey(key, value) {
			continue
		}
		for _, parsed := range parseDependencyValue(key, value) {
			if parsed.Host == "" {
				continue
			}
			sign := fmt.Sprintf("%s|%s|%d|%s", strings.ToUpper(parsed.Key), parsed.Host, parsed.Port, strings.ToLower(parsed.Protocol))
			if _, exists := seen[sign]; exists {
				continue
			}
			seen[sign] = struct{}{}
			hints = append(hints, parsed)
			if len(hints) >= maxHints {
				return hints
			}
		}
	}
	return hints
}

func parseDependencyValue(key, value string) []envDependencyHint {
	result := make([]envDependencyHint, 0, 8)
	parts := splitEnvValueTokens(value)
	if len(parts) == 0 {
		parts = []string{value}
	}
	for _, token := range parts {
		host, protocol, port := normalizeHostAndProtocol(token)
		if host == "" {
			continue
		}
		if protocol == "" {
			protocol = inferProtocolByContext(key, token)
		}
		result = append(result, envDependencyHint{
			Key:      key,
			Host:     host,
			Protocol: protocol,
			Port:     port,
		})
	}
	return result
}

func inferDependencyEdges(nodes []syncNode, existingEdges []syncEdge) ([]syncEdge, gin.H) {
	if len(nodes) == 0 {
		return nil, gin.H{"inferred_edges": 0, "source_nodes": 0}
	}
	aliasToNodes := map[string][]syncNode{}
	nodeHints := map[string][]envDependencyHint{}
	hintNodes := 0
	totalHints := 0
	for _, node := range nodes {
		for _, alias := range collectNodeAliases(node) {
			aliasToNodes[alias] = append(aliasToNodes[alias], node)
		}
		hints := parseHintsFromNodeMetadata(node.Metadata)
		if len(hints) > 0 {
			hintNodes++
			totalHints += len(hints)
			nodeHints[node.Name] = hints
		}
	}

	edgeSeen := map[string]struct{}{}
	for _, edge := range existingEdges {
		source := strings.TrimSpace(edge.SourceName)
		target := strings.TrimSpace(edge.TargetName)
		if source == "" || target == "" {
			continue
		}
		edgeSeen[source+"->"+target+"|"+strings.TrimSpace(edge.Type)] = struct{}{}
	}

	inferred := make([]syncEdge, 0, 256)
	inferredByNode := map[string]int{}
	const maxEdgePerSource = 18
	for _, node := range nodes {
		sourceName := strings.TrimSpace(node.Name)
		if sourceName == "" {
			continue
		}
		hints := nodeHints[sourceName]
		if len(hints) == 0 {
			continue
		}
		targetRecord := map[string]envDependencyHint{}
		for _, hint := range hints {
			for _, alias := range hostLookupCandidates(hint.Host) {
				targets := aliasToNodes[alias]
				for _, targetNode := range targets {
					targetName := strings.TrimSpace(targetNode.Name)
					if targetName == "" || targetName == sourceName {
						continue
					}
					if _, exists := targetRecord[targetName]; exists {
						continue
					}
					targetRecord[targetName] = hint
					if len(targetRecord) >= maxEdgePerSource {
						break
					}
				}
				if len(targetRecord) >= maxEdgePerSource {
					break
				}
			}
			if len(targetRecord) >= maxEdgePerSource {
				break
			}
		}

		for targetName, hint := range targetRecord {
			edgeType, protocol := inferEdgeTypeAndProtocol(hint)
			signature := sourceName + "->" + targetName + "|" + edgeType
			if _, exists := edgeSeen[signature]; exists {
				continue
			}
			edgeSeen[signature] = struct{}{}
			inferred = append(inferred, syncEdge{
				SourceName:  sourceName,
				TargetName:  targetName,
				Type:        edgeType,
				Protocol:    protocol,
				Port:        hint.Port,
				Description: fmt.Sprintf("auto inferred by %s", hint.Key),
			})
			inferredByNode[sourceName]++
		}
	}

	return inferred, gin.H{
		"source_nodes":          len(nodes),
		"nodes_with_hints":      hintNodes,
		"hint_count":            totalHints,
		"inferred_edges":        len(inferred),
		"inferred_source_nodes": len(inferredByNode),
	}
}

func parseHintsFromNodeMetadata(raw string) []envDependencyHint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var meta map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	value, ok := meta["dep_hints"]
	if !ok {
		return nil
	}
	array, ok := value.([]interface{})
	if !ok {
		return nil
	}
	hints := make([]envDependencyHint, 0, len(array))
	for _, item := range array {
		switch typed := item.(type) {
		case map[string]interface{}:
			key, _ := typed["key"].(string)
			host, _ := typed["host"].(string)
			protocol, _ := typed["protocol"].(string)
			port := 0
			switch p := typed["port"].(type) {
			case float64:
				port = int(p)
			case int:
				port = p
			case string:
				if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
					port = v
				}
			}
			host = strings.ToLower(strings.TrimSpace(host))
			if host == "" {
				continue
			}
			hints = append(hints, envDependencyHint{
				Key:      key,
				Host:     host,
				Protocol: protocol,
				Port:     port,
			})
		case string:
			host := strings.ToLower(strings.TrimSpace(typed))
			if host == "" {
				continue
			}
			hints = append(hints, envDependencyHint{
				Key:  "DEP_HINT",
				Host: host,
			})
		}
	}
	return hints
}

func collectNodeAliases(node syncNode) []string {
	set := map[string]struct{}{}
	add := func(value string) {
		key := strings.ToLower(strings.TrimSpace(value))
		if key != "" {
			set[key] = struct{}{}
		}
	}
	add(strings.TrimSpace(node.Name))

	segments := strings.Split(strings.TrimSpace(node.Name), "/")
	if len(segments) > 0 {
		add(segments[len(segments)-1])
	}

	meta := map[string]interface{}{}
	if strings.TrimSpace(node.Metadata) != "" {
		_ = json.Unmarshal([]byte(node.Metadata), &meta)
	}
	for _, key := range []string{"resource_name", "service_name", "container_name"} {
		if v, ok := meta[key].(string); ok {
			add(v)
			if i := strings.Index(v, "_"); i > 0 && i < len(v)-1 {
				add(v[i+1:])
			}
		}
	}

	resource, _ := meta["resource"].(string)
	resource = strings.ToLower(strings.TrimSpace(resource))
	ns := strings.ToLower(strings.TrimSpace(node.Namespace))
	if resource == "service" {
		if v, ok := meta["resource_name"].(string); ok {
			svc := strings.ToLower(strings.TrimSpace(v))
			add(svc)
			if svc != "" && ns != "" {
				add(fmt.Sprintf("%s.%s", svc, ns))
				add(fmt.Sprintf("%s.%s.svc", svc, ns))
				add(fmt.Sprintf("%s.%s.svc.cluster.local", svc, ns))
			}
		}
	}

	out := make([]string, 0, len(set))
	for key := range set {
		normalized := normalizeHostCandidate(key)
		if normalized != "" {
			out = append(out, normalized)
		}
		if key != "" {
			out = append(out, key)
		}
	}
	return uniqueStrings(out)
}

func hostLookupCandidates(host string) []string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return nil
	}
	candidates := []string{host}
	if normalized := normalizeHostCandidate(host); normalized != "" && normalized != host {
		candidates = append(candidates, normalized)
	}
	if idx := strings.Index(host, "."); idx > 0 {
		candidates = append(candidates, host[:idx])
	}
	return uniqueStrings(candidates)
}

func inferEdgeTypeAndProtocol(hint envDependencyHint) (string, string) {
	protocol := strings.ToLower(strings.TrimSpace(hint.Protocol))
	key := strings.ToUpper(strings.TrimSpace(hint.Key))
	switch {
	case strings.Contains(protocol, "http"):
		if protocol == "https" {
			return "http", "HTTPS"
		}
		return "http", "HTTP"
	case strings.Contains(protocol, "grpc"):
		return "grpc", "gRPC"
	case strings.Contains(protocol, "redis"):
		return "cache", "Redis"
	case strings.Contains(protocol, "mysql"):
		return "tcp", "MySQL"
	case strings.Contains(protocol, "postgres"):
		return "tcp", "PostgreSQL"
	case strings.Contains(protocol, "mongo"):
		return "tcp", "MongoDB"
	case strings.Contains(protocol, "kafka") || strings.Contains(protocol, "amqp") || strings.Contains(protocol, "rabbit") || strings.Contains(protocol, "nats"):
		return "mq", strings.ToUpper(protocol)
	case strings.Contains(key, "REDIS"):
		return "cache", "Redis"
	case strings.Contains(key, "KAFKA") || strings.Contains(key, "RABBIT") || strings.Contains(key, "MQ") || strings.Contains(key, "NATS"):
		return "mq", "MQ"
	default:
		return "tcp", "TCP"
	}
}

func isDependencyEnvKey(key, value string) bool {
	keyUpper := strings.ToUpper(strings.TrimSpace(key))
	if keyUpper == "" {
		return false
	}
	if strings.Contains(value, "://") {
		return true
	}
	words := []string{
		"HOST", "URL", "ADDR", "ADDRESS", "ENDPOINT", "DSN",
		"DATABASE", "DB_", "REDIS", "MYSQL", "POSTGRES", "MONGO",
		"KAFKA", "RABBIT", "MQ", "NATS", "ELASTIC", "ES_", "GRPC", "API",
	}
	for _, word := range words {
		if strings.Contains(keyUpper, word) {
			return true
		}
	}
	return false
}

func parsePort(value string) int {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 0 || port > 65535 {
		return 0
	}
	return port
}

func inferProtocolByContext(key, raw string) string {
	keyUpper := strings.ToUpper(key)
	value := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(value, "https://"), strings.Contains(keyUpper, "HTTPS"):
		return "https"
	case strings.HasPrefix(value, "http://"), strings.Contains(keyUpper, "HTTP"), strings.Contains(keyUpper, "API"):
		return "http"
	case strings.HasPrefix(value, "grpc://"), strings.Contains(keyUpper, "GRPC"):
		return "grpc"
	case strings.Contains(keyUpper, "REDIS"), strings.HasPrefix(value, "redis://"):
		return "redis"
	case strings.Contains(keyUpper, "MYSQL"), strings.HasPrefix(value, "mysql://"):
		return "mysql"
	case strings.Contains(keyUpper, "POSTGRES"), strings.HasPrefix(value, "postgres://"), strings.HasPrefix(value, "postgresql://"):
		return "postgres"
	case strings.Contains(keyUpper, "MONGO"), strings.HasPrefix(value, "mongodb://"):
		return "mongodb"
	case strings.Contains(keyUpper, "KAFKA"), strings.HasPrefix(value, "kafka://"):
		return "kafka"
	case strings.Contains(keyUpper, "RABBIT"), strings.HasPrefix(value, "amqp://"), strings.HasPrefix(value, "amqps://"):
		return "amqp"
	case strings.Contains(keyUpper, "NATS"), strings.HasPrefix(value, "nats://"):
		return "nats"
	default:
		return ""
	}
}

func splitEnvValueTokens(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\n', '\t':
			return true
		default:
			return false
		}
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		item := strings.Trim(strings.TrimSpace(field), `"'`)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

var hostValuePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func normalizeHostAndProtocol(raw string) (string, string, int) {
	token := strings.Trim(strings.TrimSpace(raw), `"'`)
	if token == "" {
		return "", "", 0
	}

	if strings.Contains(token, "://") {
		if parsed, err := url.Parse(token); err == nil {
			host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
			host = normalizeHostCandidate(host)
			if host == "" {
				return "", "", 0
			}
			return host, strings.ToLower(strings.TrimSpace(parsed.Scheme)), parsePort(parsed.Port())
		}
	}

	if at := strings.LastIndex(token, "@"); at >= 0 && at+1 < len(token) {
		token = token[at+1:]
	}
	if slash := strings.Index(token, "/"); slash >= 0 {
		token = token[:slash]
	}
	if strings.HasPrefix(token, "[") && strings.Contains(token, "]") {
		end := strings.Index(token, "]")
		host := strings.TrimSpace(token[1:end])
		host = normalizeHostCandidate(host)
		if host == "" {
			return "", "", 0
		}
		port := 0
		if end+1 < len(token) && token[end+1] == ':' {
			port = parsePort(token[end+2:])
		}
		return host, "", port
	}

	host := token
	port := 0
	if colon := strings.LastIndex(token, ":"); colon > 0 && colon < len(token)-1 {
		if parsed := parsePort(token[colon+1:]); parsed > 0 {
			host = token[:colon]
			port = parsed
		}
	}
	host = normalizeHostCandidate(host)
	if host == "" {
		return "", "", 0
	}
	return host, "", port
}

func normalizeHostCandidate(value string) string {
	host := strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return ""
	}
	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0", "::1":
		return ""
	}
	if strings.HasPrefix(host, "$") || strings.Contains(host, "{") || strings.Contains(host, "}") {
		return ""
	}
	if !hostValuePattern.MatchString(host) {
		return ""
	}
	return host
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
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
	updated, layers, err := h.autoLayoutInternal()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "没有可布局节点", "data": gin.H{"updated": 0}})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("自动布局完成，更新 %d 个节点", updated),
		"data": gin.H{
			"updated":     updated,
			"layer_count": layers,
		},
	})
}

func (h *TopologyHandler) autoLayoutInternal() (int, int, error) {
	var nodes []ServiceNode
	var edges []ServiceEdge
	if err := h.db.Find(&nodes).Error; err != nil {
		return 0, 0, err
	}
	if err := h.db.Find(&edges).Error; err != nil {
		return 0, 0, err
	}
	if len(nodes) == 0 {
		return 0, 0, nil
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
		return 0, 0, tx.Error
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
				return 0, 0, err
			}
			updated++
		}
	}
	if err := tx.Commit().Error; err != nil {
		return 0, 0, err
	}

	return updated, len(layerKeys), nil
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
