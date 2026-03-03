package k8s

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/yaml"
)

type K8sHandler struct {
	db   *gorm.DB
	auth *core.AuthService
}

func NewK8sHandler(db *gorm.DB, auth *core.AuthService) *K8sHandler {
	return &K8sHandler{db: db, auth: auth}
}

func (h *K8sHandler) getClient(clusterID string) (*kubernetes.Clientset, error) {
	var cluster Cluster
	if err := h.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func (h *K8sHandler) getRestConfig(clusterID string) (*rest.Config, error) {
	var cluster Cluster
	if err := h.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return nil, err
	}
	return clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
}

// ListClusters 集群列表
func (h *K8sHandler) ListClusters(c *gin.Context) {
	var clusters []Cluster
	if err := h.db.Find(&clusters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": clusters})
}

// CreateCluster 创建集群
func (h *K8sHandler) CreateCluster(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name"`
		APIServer   string `json:"api_server"`
		KubeConfig  string `json:"kube_config"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	cluster := Cluster{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		APIServer:   req.APIServer,
		KubeConfig:  req.KubeConfig,
		Description: req.Description,
		Status:      1,
	}
	if err := h.db.Create(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cluster})
}

// GetCluster 获取集群详情
func (h *K8sHandler) GetCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster Cluster
	if err := h.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "集群不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cluster})
}

// UpdateCluster 更新集群
func (h *K8sHandler) UpdateCluster(c *gin.Context) {
	id := c.Param("id")
	var cluster Cluster
	if err := h.db.First(&cluster, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "集群不存在"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		APIServer   string `json:"api_server"`
		KubeConfig  string `json:"kube_config"`
		Description string `json:"description"`
		Status      *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Name != "" {
		cluster.Name = req.Name
	}
	if req.DisplayName != "" {
		cluster.DisplayName = req.DisplayName
	}
	if req.APIServer != "" {
		cluster.APIServer = req.APIServer
	}
	if req.KubeConfig != "" {
		cluster.KubeConfig = req.KubeConfig
	}
	if req.Description != "" {
		cluster.Description = req.Description
	}
	if req.Status != nil {
		cluster.Status = *req.Status
	}
	if err := h.db.Save(&cluster).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cluster})
}

// DeleteCluster 删除集群
func (h *K8sHandler) DeleteCluster(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&Cluster{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 测试集群连接
func (h *K8sHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "连接失败: " + err.Error()})
		return
	}

	version, err := client.Discovery().ServerVersion()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "获取版本失败: " + err.Error()})
		return
	}

	// 更新集群版本
	h.db.Model(&Cluster{}).Where("id = ?", id).Update("version", version.GitVersion)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"version": version.GitVersion, "status": "connected"}})
}

// ListNodes 节点列表
func (h *K8sHandler) ListNodes(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]ClusterNode, 0)
	for _, node := range nodes.Items {
		n := ClusterNode{
			ClusterID:    id,
			Name:         node.Name,
			KubeletVer:   node.Status.NodeInfo.KubeletVersion,
			OS:           node.Status.NodeInfo.OSImage,
			CreationTime: node.CreationTimestamp.Time,
		}

		// 获取角色
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				n.Roles = append(n.Roles, "master")
			}
		}
		if len(n.Roles) == 0 {
			n.Roles = []string{"worker"}
		}

		// 获取状态
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					n.Status = "Ready"
				} else {
					n.Status = "NotReady"
				}
			}
		}

		// 获取IP
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				n.InternalIP = addr.Address
			}
		}

		// 资源
		n.CPU = node.Status.Capacity.Cpu().String()
		n.Memory = node.Status.Capacity.Memory().String()

		result = append(result, n)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListNamespaces 命名空间列表
func (h *K8sHandler) ListNamespaces(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	nsList, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]Namespace, 0)
	for _, ns := range nsList.Items {
		result = append(result, Namespace{
			ClusterID: id,
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			Labels:    ns.Labels,
			CreatedAt: ns.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListWorkloads 工作负载列表
func (h *K8sHandler) ListWorkloads(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result := make([]Workload, 0)

	services, _ := client.CoreV1().Services(ns).List(context.Background(), metav1.ListOptions{})
	ingresses, _ := client.NetworkingV1().Ingresses(ns).List(context.Background(), metav1.ListOptions{})
	serviceHostMap := buildServiceHostMap(ingresses.Items)

	// Deployments
	deployments, _ := client.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{})
	for _, d := range deployments.Items {
		images := make([]string, 0)
		for _, c := range d.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		domains := resolveDeploymentDomains(d, services.Items, serviceHostMap)
		result = append(result, Workload{
			ClusterID: id,
			Namespace: d.Namespace,
			Name:      d.Name,
			Kind:      "Deployment",
			Replicas:  replicas,
			Ready:     d.Status.ReadyReplicas,
			Available: d.Status.AvailableReplicas,
			Labels:    d.Labels,
			Images:    images,
			Domains:   domains,
			CreatedAt: d.CreationTimestamp.Time,
		})
	}

	// StatefulSets
	statefulSets, _ := client.AppsV1().StatefulSets(ns).List(context.Background(), metav1.ListOptions{})
	for _, s := range statefulSets.Items {
		images := make([]string, 0)
		for _, ctn := range s.Spec.Template.Spec.Containers {
			images = append(images, ctn.Image)
		}
		replicas := int32(0)
		if s.Spec.Replicas != nil {
			replicas = *s.Spec.Replicas
		}
		result = append(result, Workload{
			ClusterID: id,
			Namespace: s.Namespace,
			Name:      s.Name,
			Kind:      "StatefulSet",
			Replicas:  replicas,
			Ready:     s.Status.ReadyReplicas,
			Available: s.Status.ReadyReplicas,
			Labels:    s.Labels,
			Images:    images,
			CreatedAt: s.CreationTimestamp.Time,
		})
	}

	// DaemonSets
	daemonSets, _ := client.AppsV1().DaemonSets(ns).List(context.Background(), metav1.ListOptions{})
	for _, ds := range daemonSets.Items {
		images := make([]string, 0)
		for _, ctn := range ds.Spec.Template.Spec.Containers {
			images = append(images, ctn.Image)
		}
		result = append(result, Workload{
			ClusterID: id,
			Namespace: ds.Namespace,
			Name:      ds.Name,
			Kind:      "DaemonSet",
			Replicas:  ds.Status.DesiredNumberScheduled,
			Ready:     ds.Status.NumberReady,
			Available: ds.Status.NumberAvailable,
			Labels:    ds.Labels,
			Images:    images,
			CreatedAt: ds.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func resolveDeploymentDomains(deploy appsv1.Deployment, services []corev1.Service, serviceHostMap map[string][]string) []string {
	if len(services) == 0 || len(serviceHostMap) == 0 {
		return nil
	}
	podLabels := deploy.Spec.Template.Labels
	if len(podLabels) == 0 {
		return nil
	}
	domainSet := make(map[string]struct{})
	for _, svc := range services {
		if svc.Namespace != deploy.Namespace {
			continue
		}
		if !selectorMatches(svc.Spec.Selector, podLabels) {
			continue
		}
		for _, host := range serviceHostMap[svc.Name] {
			domainSet[host] = struct{}{}
		}
	}
	if len(domainSet) == 0 {
		return nil
	}
	domains := make([]string, 0, len(domainSet))
	for host := range domainSet {
		domains = append(domains, host)
	}
	sort.Strings(domains)
	return domains
}

func buildServiceHostMap(ingresses []networkingv1.Ingress) map[string][]string {
	serviceHosts := make(map[string]map[string]struct{})
	add := func(serviceName, host string) {
		serviceName = strings.TrimSpace(serviceName)
		host = strings.TrimSpace(host)
		if serviceName == "" || host == "" {
			return
		}
		if _, ok := serviceHosts[serviceName]; !ok {
			serviceHosts[serviceName] = make(map[string]struct{})
		}
		serviceHosts[serviceName][host] = struct{}{}
	}

	for _, ing := range ingresses {
		for _, rule := range ing.Spec.Rules {
			host := strings.TrimSpace(rule.Host)
			if host == "" || rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service == nil {
					continue
				}
				add(path.Backend.Service.Name, host)
			}
		}
		if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
			for _, rule := range ing.Spec.Rules {
				if strings.TrimSpace(rule.Host) != "" {
					add(ing.Spec.DefaultBackend.Service.Name, rule.Host)
				}
			}
		}
	}

	result := make(map[string][]string, len(serviceHosts))
	for svc, hosts := range serviceHosts {
		list := make([]string, 0, len(hosts))
		for h := range hosts {
			list = append(list, h)
		}
		sort.Strings(list)
		result[svc] = list
	}
	return result
}

func selectorMatches(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}

// GetWorkload 获取工作负载详情
func (h *K8sHandler) GetWorkload(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	switch strings.ToLower(kind) {
	case "deployment":
		d, err := client.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		images := make([]string, 0)
		for _, ctn := range d.Spec.Template.Spec.Containers {
			images = append(images, ctn.Image)
		}
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": WorkloadDetail{
			ClusterID: id,
			Namespace: d.Namespace,
			Name:      d.Name,
			Kind:      "Deployment",
			Replicas:  replicas,
			Ready:     d.Status.ReadyReplicas,
			Available: d.Status.AvailableReplicas,
			Labels:    d.Labels,
			Images:    images,
			Selector:  d.Spec.Selector.MatchLabels,
			CreatedAt: d.CreationTimestamp.Time,
		}})
		return
	case "statefulset":
		s, err := client.AppsV1().StatefulSets(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		images := make([]string, 0)
		for _, ctn := range s.Spec.Template.Spec.Containers {
			images = append(images, ctn.Image)
		}
		replicas := int32(0)
		if s.Spec.Replicas != nil {
			replicas = *s.Spec.Replicas
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": WorkloadDetail{
			ClusterID: id,
			Namespace: s.Namespace,
			Name:      s.Name,
			Kind:      "StatefulSet",
			Replicas:  replicas,
			Ready:     s.Status.ReadyReplicas,
			Available: s.Status.ReadyReplicas,
			Labels:    s.Labels,
			Images:    images,
			Selector:  s.Spec.Selector.MatchLabels,
			CreatedAt: s.CreationTimestamp.Time,
		}})
		return
	case "daemonset":
		ds, err := client.AppsV1().DaemonSets(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		images := make([]string, 0)
		for _, ctn := range ds.Spec.Template.Spec.Containers {
			images = append(images, ctn.Image)
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": WorkloadDetail{
			ClusterID: id,
			Namespace: ds.Namespace,
			Name:      ds.Name,
			Kind:      "DaemonSet",
			Replicas:  ds.Status.DesiredNumberScheduled,
			Ready:     ds.Status.NumberReady,
			Available: ds.Status.NumberAvailable,
			Labels:    ds.Labels,
			Images:    images,
			Selector:  ds.Spec.Selector.MatchLabels,
			CreatedAt: ds.CreationTimestamp.Time,
		}})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型"})
		return
	}
}

// ScaleWorkload 扩缩容
func (h *K8sHandler) ScaleWorkload(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")

	var req struct {
		Replicas int32 `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	switch strings.ToLower(kind) {
	case "deployment":
		scale, err := client.AppsV1().Deployments(ns).GetScale(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		scale.Spec.Replicas = req.Replicas
		if _, err := client.AppsV1().Deployments(ns).UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "statefulset":
		scale, err := client.AppsV1().StatefulSets(ns).GetScale(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		scale.Spec.Replicas = req.Replicas
		if _, err := client.AppsV1().StatefulSets(ns).UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持Deployment/StatefulSet扩缩容"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": fmt.Sprintf("已调整副本数为 %d", req.Replicas)})
}

// RestartWorkloadByRef 滚动重启工作负载
func (h *K8sHandler) RestartWorkloadByRef(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	restartedAt := time.Now().UTC().Format(time.RFC3339)
	switch strings.ToLower(kind) {
	case "deployment":
		deploy, err := client.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = map[string]string{}
		}
		deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().Deployments(ns).Update(context.Background(), deploy, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "statefulset":
		sts, err := client.AppsV1().StatefulSets(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = map[string]string{}
		}
		sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().StatefulSets(ns).Update(context.Background(), sts, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "daemonset":
		ds, err := client.AppsV1().DaemonSets(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = map[string]string{}
		}
		ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().DaemonSets(ns).Update(context.Background(), ds, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已触发滚动重启"})
}

// GetWorkloadManifest 获取工作负载YAML/JSON
func (h *K8sHandler) GetWorkloadManifest(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")
	format := strings.ToLower(c.DefaultQuery("format", "yaml"))

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var obj interface{}
	switch strings.ToLower(kind) {
	case "deployment":
		obj, err = client.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	case "statefulset":
		obj, err = client.AppsV1().StatefulSets(ns).Get(context.Background(), name, metav1.GetOptions{})
	case "daemonset":
		obj, err = client.AppsV1().DaemonSets(ns).Get(context.Background(), name, metav1.GetOptions{})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	var data []byte
	if format == "json" {
		data, err = json.MarshalIndent(obj, "", "  ")
	} else {
		data, err = yaml.Marshal(obj)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"format":  format,
			"content": string(data),
		},
	})
}

// ApplyWorkloadManifest 应用工作负载YAML/JSON
func (h *K8sHandler) ApplyWorkloadManifest(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")

	var req struct {
		Format  string `json:"format"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.Content), &obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "解析失败: " + err.Error()})
		return
	}

	// 清理不可更新字段，避免冲突
	cleanupManifestMap(obj)

	data, err := json.Marshal(obj)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	switch strings.ToLower(kind) {
	case "deployment":
		_, err = client.AppsV1().Deployments(ns).Patch(context.Background(), name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	case "statefulset":
		_, err = client.AppsV1().StatefulSets(ns).Patch(context.Background(), name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	case "daemonset":
		_, err = client.AppsV1().DaemonSets(ns).Patch(context.Background(), name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已应用更新"})
}

func cleanupManifestMap(obj map[string]interface{}) {
	delete(obj, "status")
	meta, ok := obj["metadata"].(map[string]interface{})
	if ok {
		delete(meta, "creationTimestamp")
		delete(meta, "resourceVersion")
		delete(meta, "uid")
		delete(meta, "managedFields")
		delete(meta, "generation")
	}
}

// ListDeployments Deployment列表
func (h *K8sHandler) ListDeployments(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	if ns == "all" || ns == "*" {
		ns = ""
	}
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	deployments, err := client.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": deployments.Items})
}

// CreateDeployment 创建Deployment
func (h *K8sHandler) CreateDeployment(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	var req struct {
		Name          string            `json:"name" binding:"required"`
		Image         string            `json:"image" binding:"required"`
		Replicas      *int32            `json:"replicas"`
		ContainerName string            `json:"container_name"`
		ContainerPort *int32            `json:"container_port"`
		Labels        map[string]string `json:"labels"`
		Env           map[string]string `json:"env"`
		Command       []string          `json:"command"`
		Args          []string          `json:"args"`
		CPURequest    string            `json:"cpu_request"`
		MemoryRequest string            `json:"memory_request"`
		CPULimit      string            `json:"cpu_limit"`
		MemoryLimit   string            `json:"memory_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Image = strings.TrimSpace(req.Image)
	if req.Name == "" || req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name 和 image 必填"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	replicas := int32(1)
	if req.Replicas != nil && *req.Replicas >= 0 {
		replicas = *req.Replicas
	}

	labels := map[string]string{"app": req.Name}
	for k, v := range req.Labels {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key != "" && val != "" {
			labels[key] = val
		}
	}
	if _, ok := labels["app"]; !ok {
		labels["app"] = req.Name
	}

	containerName := strings.TrimSpace(req.ContainerName)
	if containerName == "" {
		containerName = req.Name
	}
	container := corev1.Container{
		Name:            containerName,
		Image:           req.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
	}
	if req.ContainerPort != nil && *req.ContainerPort > 0 {
		container.Ports = []corev1.ContainerPort{{ContainerPort: *req.ContainerPort}}
	}
	if len(req.Command) > 0 {
		container.Command = req.Command
	}
	if len(req.Args) > 0 {
		container.Args = req.Args
	}

	if len(req.Env) > 0 {
		keys := make([]string, 0, len(req.Env))
		for k := range req.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		envVars := make([]corev1.EnvVar, 0, len(keys))
		for _, k := range keys {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			envVars = append(envVars, corev1.EnvVar{Name: key, Value: req.Env[k]})
		}
		container.Env = envVars
	}

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	parseQuantity := func(field, raw string) (resource.Quantity, error) {
		q, err := resource.ParseQuantity(strings.TrimSpace(raw))
		if err != nil {
			return resource.Quantity{}, fmt.Errorf("%s 格式无效: %v", field, err)
		}
		return q, nil
	}
	if strings.TrimSpace(req.CPURequest) != "" {
		q, qErr := parseQuantity("cpu_request", req.CPURequest)
		if qErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": qErr.Error()})
			return
		}
		resources.Requests[corev1.ResourceCPU] = q
	}
	if strings.TrimSpace(req.MemoryRequest) != "" {
		q, qErr := parseQuantity("memory_request", req.MemoryRequest)
		if qErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": qErr.Error()})
			return
		}
		resources.Requests[corev1.ResourceMemory] = q
	}
	if strings.TrimSpace(req.CPULimit) != "" {
		q, qErr := parseQuantity("cpu_limit", req.CPULimit)
		if qErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": qErr.Error()})
			return
		}
		resources.Limits[corev1.ResourceCPU] = q
	}
	if strings.TrimSpace(req.MemoryLimit) != "" {
		q, qErr := parseQuantity("memory_limit", req.MemoryLimit)
		if qErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": qErr.Error()})
			return
		}
		resources.Limits[corev1.ResourceMemory] = q
	}
	if len(resources.Requests) > 0 || len(resources.Limits) > 0 {
		container.Resources = resources
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": labels["app"]},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	created, err := client.AppsV1().Deployments(ns).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"name":      created.Name,
			"namespace": created.Namespace,
			"replicas":  replicas,
		},
	})
}

// DeleteDeployment 删除Deployment
func (h *K8sHandler) DeleteDeployment(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	propagation := metav1.DeletePropagationBackground
	err = client.AppsV1().Deployments(ns).Delete(context.Background(), name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ScaleDeployment 扩缩容
func (h *K8sHandler) ScaleDeployment(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	var req struct {
		Replicas int32 `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	scale, err := client.AppsV1().Deployments(ns).GetScale(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	scale.Spec.Replicas = req.Replicas
	_, err = client.AppsV1().Deployments(ns).UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": fmt.Sprintf("已调整副本数为 %d", req.Replicas)})
}

// RestartDeployment 重启Deployment
func (h *K8sHandler) RestartDeployment(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	deployment, err := client.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 添加重启注解
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z")

	_, err = client.AppsV1().Deployments(ns).Update(context.Background(), deployment, metav1.UpdateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "重启成功"})
}

// ListPods Pod列表
func (h *K8sHandler) ListPods(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	selector := c.Query("selector")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	opts := metav1.ListOptions{}
	if selector != "" {
		opts.LabelSelector = selector
	}
	pods, err := client.CoreV1().Pods(ns).List(context.Background(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]Pod, 0)
	for _, p := range pods.Items {
		ownerKind, ownerName := resolveOwnerRef(&p, nil)
		pod := Pod{
			ClusterID: id,
			Namespace: p.Namespace,
			Name:      p.Name,
			Status:    string(p.Status.Phase),
			Node:      p.Spec.NodeName,
			IP:        p.Status.PodIP,
			Labels:    p.Labels,
			OwnerKind: ownerKind,
			OwnerName: ownerName,
			CreatedAt: p.CreationTimestamp.Time,
		}

		for _, cs := range p.Status.ContainerStatuses {
			pod.Restarts += cs.RestartCount
			state := "unknown"
			if cs.State.Running != nil {
				state = "running"
			} else if cs.State.Waiting != nil {
				state = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				state = "terminated"
			}
			pod.Containers = append(pod.Containers, Container{
				Name:  cs.Name,
				Image: cs.Image,
				Ready: cs.Ready,
				State: state,
			})
		}

		result = append(result, pod)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetPod Pod详情
func (h *K8sHandler) GetPod(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	p, err := client.CoreV1().Pods(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	ownerKind, ownerName := resolveOwnerRef(p, nil)
	pod := Pod{
		ClusterID: id,
		Namespace: p.Namespace,
		Name:      p.Name,
		Status:    string(p.Status.Phase),
		Node:      p.Spec.NodeName,
		IP:        p.Status.PodIP,
		Labels:    p.Labels,
		OwnerKind: ownerKind,
		OwnerName: ownerName,
		CreatedAt: p.CreationTimestamp.Time,
	}
	for _, cs := range p.Status.ContainerStatuses {
		pod.Restarts += cs.RestartCount
		state := "unknown"
		if cs.State.Running != nil {
			state = "running"
		} else if cs.State.Waiting != nil {
			state = cs.State.Waiting.Reason
		} else if cs.State.Terminated != nil {
			state = cs.State.Terminated.Reason
		}
		pod.Containers = append(pod.Containers, Container{
			Name:  cs.Name,
			Image: cs.Image,
			Ready: cs.Ready,
			State: state,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pod})
}

// RestartPod 删除Pod以触发重建
func (h *K8sHandler) RestartPod(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := client.CoreV1().Pods(ns).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除Pod并触发重建"})
}

// RestartWorkload 对Pod所属工作负载进行滚动重启
func (h *K8sHandler) RestartWorkload(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	p, err := client.CoreV1().Pods(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	kind, ownerName, err := resolveWorkloadOwner(client, ns, p)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if kind == "" || ownerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未找到工作负载控制器"})
		return
	}

	restartedAt := time.Now().UTC().Format(time.RFC3339)
	switch kind {
	case "Deployment":
		deploy, err := client.AppsV1().Deployments(ns).Get(context.Background(), ownerName, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = map[string]string{}
		}
		deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().Deployments(ns).Update(context.Background(), deploy, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "StatefulSet":
		sts, err := client.AppsV1().StatefulSets(ns).Get(context.Background(), ownerName, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = map[string]string{}
		}
		sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().StatefulSets(ns).Update(context.Background(), sts, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	case "DaemonSet":
		ds, err := client.AppsV1().DaemonSets(ns).Get(context.Background(), ownerName, metav1.GetOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = map[string]string{}
		}
		ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
		if _, err := client.AppsV1().DaemonSets(ns).Update(context.Background(), ds, metav1.UpdateOptions{}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的工作负载类型: " + kind})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已触发滚动重启: " + kind + "/" + ownerName})
}

func resolveOwnerRef(pod *corev1.Pod, fallback *metav1.OwnerReference) (string, string) {
	var owner *metav1.OwnerReference
	if pod != nil {
		for i := range pod.OwnerReferences {
			ref := pod.OwnerReferences[i]
			if ref.Controller != nil && *ref.Controller {
				owner = &ref
				break
			}
		}
		if owner == nil && len(pod.OwnerReferences) > 0 {
			owner = &pod.OwnerReferences[0]
		}
	}
	if owner == nil && fallback != nil {
		owner = fallback
	}
	if owner == nil {
		return "", ""
	}
	return owner.Kind, owner.Name
}

func resolveWorkloadOwner(client *kubernetes.Clientset, ns string, pod *corev1.Pod) (string, string, error) {
	kind, name := resolveOwnerRef(pod, nil)
	if kind == "" || name == "" {
		return "", "", nil
	}
	if kind != "ReplicaSet" {
		return kind, name, nil
	}

	rs, err := client.AppsV1().ReplicaSets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	rsKind, rsName := resolveOwnerRef(nil, ownerRefFromList(rs.OwnerReferences))
	if rsKind == "" || rsName == "" {
		return "ReplicaSet", name, nil
	}
	return rsKind, rsName, nil
}

func ownerRefFromList(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		ref := refs[i]
		if ref.Controller != nil && *ref.Controller {
			return &ref
		}
	}
	if len(refs) > 0 {
		return &refs[0]
	}
	return nil
}

// GetPodLogs 获取Pod日志
func (h *K8sHandler) GetPodLogs(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "100"), 10, 64)

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := client.CoreV1().Pods(ns).GetLogs(name, opts)
	stream, err := req.Stream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer stream.Close()

	logs, _ := io.ReadAll(stream)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": string(logs)})
}

// StreamPodLogs 实时日志（SSE）
func (h *K8sHandler) StreamPodLogs(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "100"), 10, 64)

	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
		return
	}
	if _, err := h.auth.ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token无效"})
		return
	}

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	opts := &corev1.PodLogOptions{
		Follow:    true,
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	req := client.CoreV1().Pods(ns).GetLogs(name, opts)
	stream, err := req.Stream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer stream.Close()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "不支持流式输出"})
		return
	}

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				writeSSE(c, line)
				flusher.Flush()
			}
			if err != nil {
				return
			}
		}
	}
}

func writeSSE(c *gin.Context, data []byte) {
	text := strings.ReplaceAll(string(data), "\r", "")
	lines := strings.Split(text, "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		_, _ = c.Writer.Write([]byte("data: " + l + "\n"))
	}
	_, _ = c.Writer.Write([]byte("\n"))
}

// DeletePod 删除Pod
func (h *K8sHandler) DeletePod(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	name := c.Param("name")

	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	err = client.CoreV1().Pods(ns).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ExecPod WebSocket exec to Pod
func (h *K8sHandler) ExecPod(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	pod := c.Param("name")
	if pod == "" {
		pod = c.Param("pod")
	}
	container := c.Query("container")
	command := strings.TrimSpace(c.Query("command"))

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	cfg, err := h.getRestConfig(id)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("获取集群配置失败: "+err.Error()))
		conn.Close()
		return
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("创建客户端失败: "+err.Error()))
		conn.Close()
		return
	}

	if container == "" {
		podObj, err := client.CoreV1().Pods(ns).Get(context.Background(), pod, metav1.GetOptions{})
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("获取Pod信息失败: "+err.Error()))
			conn.Close()
			return
		}
		if len(podObj.Spec.Containers) == 0 {
			conn.WriteMessage(websocket.TextMessage, []byte("Pod没有可用容器"))
			conn.Close()
			return
		}
		container = podObj.Spec.Containers[0].Name
	}

	stdinReader, stdinWriter := io.Pipe()
	defer stdinReader.Close()

	resizeQueue := newTerminalSizeQueue()

	go func() {
		defer stdinWriter.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				var payload struct {
					Type string `json:"type"`
					Cols int    `json:"cols"`
					Rows int    `json:"rows"`
				}
				if json.Unmarshal(message, &payload) == nil && payload.Type == "resize" {
					resizeQueue.Push(payload.Cols, payload.Rows)
					continue
				}
			}
			_, _ = stdinWriter.Write(message)
		}
	}()

	commands := make([][]string, 0, 4)
	if command != "" {
		commands = append(commands, []string{command})
	}
	commands = append(commands,
		[]string{"/bin/sh"},
		[]string{"/bin/bash"},
		[]string{"/bin/ash"},
	)

	var lastErr error
	for _, cmd := range commands {
		req := client.CoreV1().RESTClient().
			Post().
			Resource("pods").
			Name(pod).
			Namespace(ns).
			SubResource("exec")

		req.VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
		if err != nil {
			lastErr = err
			continue
		}

		streamErr := exec.Stream(remotecommand.StreamOptions{
			Stdin:             stdinReader,
			Stdout:            &wsWriter{conn: conn},
			Stderr:            &wsWriter{conn: conn},
			Tty:               true,
			TerminalSizeQueue: resizeQueue,
		})

		if streamErr == nil {
			conn.Close()
			return
		}
		lastErr = streamErr
		if !isCommandNotFound(streamErr) {
			break
		}
	}

	if lastErr != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("执行结束: "+lastErr.Error()))
	}
	conn.Close()
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if w.conn == nil {
		return 0, io.ErrClosedPipe
	}
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type terminalSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func newTerminalSizeQueue() *terminalSizeQueue {
	return &terminalSizeQueue{ch: make(chan remotecommand.TerminalSize, 10)}
}

func (q *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

func (q *terminalSizeQueue) Push(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	select {
	case q.ch <- remotecommand.TerminalSize{Width: uint16(cols), Height: uint16(rows)}:
	default:
	}
}

func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "executable file not found")
}

// ListServices Service列表
func (h *K8sHandler) ListServices(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	services, err := client.CoreV1().Services(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]ServiceInfo, 0)
	for _, svc := range services.Items {
		ports := make([]string, 0)
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		result = append(result, ServiceInfo{
			ClusterID: id,
			Namespace: svc.Namespace,
			Name:      svc.Name,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Ports:     ports,
			Selector:  svc.Spec.Selector,
			CreatedAt: svc.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListIngresses Ingress列表
func (h *K8sHandler) ListIngresses(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	ingresses, err := client.NetworkingV1().Ingresses(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]IngressInfo, 0)
	for _, ing := range ingresses.Items {
		hosts := make([]string, 0)
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		className := ""
		if ing.Spec.IngressClassName != nil {
			className = *ing.Spec.IngressClassName
		}
		result = append(result, IngressInfo{
			ClusterID: id,
			Namespace: ing.Namespace,
			Name:      ing.Name,
			ClassName: className,
			Hosts:     hosts,
			CreatedAt: ing.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListConfigMaps ConfigMap列表
func (h *K8sHandler) ListConfigMaps(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	configMaps, err := client.CoreV1().ConfigMaps(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]ConfigMapInfo, 0)
	for _, cm := range configMaps.Items {
		keys := make([]string, 0, len(cm.Data))
		for k := range cm.Data {
			keys = append(keys, k)
		}
		result = append(result, ConfigMapInfo{
			ClusterID: id,
			Namespace: cm.Namespace,
			Name:      cm.Name,
			DataKeys:  keys,
			CreatedAt: cm.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListSecrets Secret列表（不返回明文）
func (h *K8sHandler) ListSecrets(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	secrets, err := client.CoreV1().Secrets(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]SecretInfo, 0)
	for _, s := range secrets.Items {
		keys := make([]string, 0, len(s.Data))
		for k := range s.Data {
			keys = append(keys, k)
		}
		result = append(result, SecretInfo{
			ClusterID: id,
			Namespace: s.Namespace,
			Name:      s.Name,
			Type:      string(s.Type),
			DataKeys:  keys,
			CreatedAt: s.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListStorageClasses StorageClass列表
func (h *K8sHandler) ListStorageClasses(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	scs, err := client.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]StorageClassInfo, 0)
	for _, sc := range scs.Items {
		reclaim := ""
		if sc.ReclaimPolicy != nil {
			reclaim = string(*sc.ReclaimPolicy)
		}
		binding := ""
		if sc.VolumeBindingMode != nil {
			binding = string(*sc.VolumeBindingMode)
		}
		allow := false
		if sc.AllowVolumeExpansion != nil {
			allow = *sc.AllowVolumeExpansion
		}
		result = append(result, StorageClassInfo{
			ClusterID:      id,
			Name:           sc.Name,
			Provisioner:    sc.Provisioner,
			ReclaimPolicy:  reclaim,
			VolumeBinding:  binding,
			AllowExpansion: allow,
			CreatedAt:      sc.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListPersistentVolumes PV列表
func (h *K8sHandler) ListPersistentVolumes(c *gin.Context) {
	id := c.Param("id")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	pvs, err := client.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]PersistentVolumeInfo, 0)
	for _, pv := range pvs.Items {
		capacity := ""
		if qty, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = qty.String()
		}
		claim := ""
		if pv.Spec.ClaimRef != nil {
			claim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}
		modes := make([]string, 0)
		for _, m := range pv.Spec.AccessModes {
			modes = append(modes, string(m))
		}
		result = append(result, PersistentVolumeInfo{
			ClusterID:  id,
			Name:       pv.Name,
			Capacity:   capacity,
			AccessMode: modes,
			Status:     string(pv.Status.Phase),
			StorageCls: pv.Spec.StorageClassName,
			Claim:      claim,
			CreatedAt:  pv.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListPersistentVolumeClaims PVC列表
func (h *K8sHandler) ListPersistentVolumeClaims(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	pvcs, err := client.CoreV1().PersistentVolumeClaims(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]PersistentVolumeClaimInfo, 0)
	for _, pvc := range pvcs.Items {
		capacity := ""
		if qty, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = qty.String()
		}
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}
		modes := make([]string, 0)
		for _, m := range pvc.Spec.AccessModes {
			modes = append(modes, string(m))
		}
		result = append(result, PersistentVolumeClaimInfo{
			ClusterID:  id,
			Namespace:  pvc.Namespace,
			Name:       pvc.Name,
			Capacity:   capacity,
			AccessMode: modes,
			Status:     string(pvc.Status.Phase),
			StorageCls: storageClass,
			VolumeName: pvc.Spec.VolumeName,
			CreatedAt:  pvc.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListEvents 事件列表
func (h *K8sHandler) ListEvents(c *gin.Context) {
	id := c.Param("id")
	ns := c.DefaultQuery("namespace", "")
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	events, err := client.CoreV1().Events(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]EventInfo, 0)
	for _, e := range events.Items {
		involved := ""
		if e.InvolvedObject.Kind != "" {
			involved = fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name)
		}
		firstSeen := e.FirstTimestamp.Time
		if firstSeen.IsZero() {
			firstSeen = e.EventTime.Time
		}
		lastSeen := e.LastTimestamp.Time
		if lastSeen.IsZero() {
			lastSeen = e.EventTime.Time
		}
		result = append(result, EventInfo{
			ClusterID: id,
			Namespace: e.Namespace,
			Name:      e.Name,
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Involved:  involved,
			Count:     e.Count,
			FirstSeen: firstSeen,
			LastSeen:  lastSeen,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
