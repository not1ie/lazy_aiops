package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
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
			if err := tx.Where("source_id IN ? OR target_id IN ?", ids, ids).Delete(&ServiceEdge{}).Error; err != nil {
				return nil, rollback(err)
			}
			if err := tx.Where("service_id IN ?", ids).Delete(&ServiceDependency{}).Error; err != nil {
				return nil, rollback(err)
			}
			if err := query.Delete(&ServiceNode{}).Error; err != nil {
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
		err := tx.Where("name = ?", name).First(&existing).Error
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
