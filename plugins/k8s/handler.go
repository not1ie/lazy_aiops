package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sHandler struct {
	db *gorm.DB
}

func NewK8sHandler(db *gorm.DB) *K8sHandler {
	return &K8sHandler{db: db}
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
	var cluster Cluster
	if err := c.ShouldBindJSON(&cluster); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
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
	if err := c.ShouldBindJSON(&cluster); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
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

	// Deployments
	deployments, _ := client.AppsV1().Deployments(ns).List(context.Background(), metav1.ListOptions{})
	for _, d := range deployments.Items {
		images := make([]string, 0)
		for _, c := range d.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		result = append(result, Workload{
			ClusterID: id,
			Namespace: d.Namespace,
			Name:      d.Name,
			Kind:      "Deployment",
			Replicas:  *d.Spec.Replicas,
			Ready:     d.Status.ReadyReplicas,
			Available: d.Status.AvailableReplicas,
			Labels:    d.Labels,
			Images:    images,
			CreatedAt: d.CreationTimestamp.Time,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListDeployments Deployment列表
func (h *K8sHandler) ListDeployments(c *gin.Context) {
	id := c.Param("id")
	ns := c.Param("ns")
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
	client, err := h.getClient(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	pods, err := client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	result := make([]Pod, 0)
	for _, p := range pods.Items {
		pod := Pod{
			ClusterID: id,
			Namespace: p.Namespace,
			Name:      p.Name,
			Status:    string(p.Status.Phase),
			Node:      p.Spec.NodeName,
			IP:        p.Status.PodIP,
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
