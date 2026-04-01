package k8s

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

// Cluster K8s集群
type Cluster struct {
	BaseModel
	Name         string     `gorm:"size:128;uniqueIndex" json:"name"`
	DisplayName  string     `gorm:"size:128" json:"display_name"`
	APIServer    string     `gorm:"size:256" json:"api_server"`
	KubeConfig   string     `gorm:"type:text" json:"-"` // 存储kubeconfig内容
	Version      string     `gorm:"size:32" json:"version"`
	Status       int        `gorm:"default:1" json:"status"` // 1:正常 0:异常 2:维护
	NodeCount    int        `json:"node_count"`
	LastCheckAt  *time.Time `json:"last_check_at"`
	LastOnlineAt *time.Time `json:"last_online_at"`
	StatusReason string     `gorm:"size:256" json:"status_reason"`
	Description  string     `gorm:"size:512" json:"description"`
}

// ClusterNode 集群节点
type ClusterNode struct {
	ClusterID    string    `json:"cluster_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Roles        []string  `json:"roles"`
	InternalIP   string    `json:"internal_ip"`
	OS           string    `json:"os"`
	KubeletVer   string    `json:"kubelet_version"`
	CPU          string    `json:"cpu"`
	Memory       string    `json:"memory"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	PodCount     int       `json:"pod_count"`
	CreationTime time.Time `json:"creation_time"`
}

// Namespace 命名空间
type Namespace struct {
	ClusterID string            `json:"cluster_id"`
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"created_at"`
}

// Workload 工作负载
type Workload struct {
	ClusterID string            `json:"cluster_id"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Kind      string            `json:"kind"` // Deployment, StatefulSet, DaemonSet
	Replicas  int32             `json:"replicas"`
	Ready     int32             `json:"ready"`
	Available int32             `json:"available"`
	Labels    map[string]string `json:"labels"`
	Images    []string          `json:"images"`
	Domains   []string          `json:"domains,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// WorkloadDetail 工作负载详情
type WorkloadDetail struct {
	ClusterID string            `json:"cluster_id"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Kind      string            `json:"kind"`
	Replicas  int32             `json:"replicas"`
	Ready     int32             `json:"ready"`
	Available int32             `json:"available"`
	Labels    map[string]string `json:"labels"`
	Images    []string          `json:"images"`
	Selector  map[string]string `json:"selector"`
	CreatedAt time.Time         `json:"created_at"`
}

type DeploymentEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DeploymentContainerRuntime struct {
	Name  string             `json:"name"`
	Image string             `json:"image"`
	Env   []DeploymentEnvVar `json:"env"`
}

type DeploymentServiceOption struct {
	Name  string  `json:"name"`
	Ports []int32 `json:"ports"`
}

type DeploymentRuntime struct {
	ClusterID          string                       `json:"cluster_id"`
	Namespace          string                       `json:"namespace"`
	Name               string                       `json:"name"`
	Replicas           int32                        `json:"replicas"`
	Ready              int32                        `json:"ready"`
	Updated            int32                        `json:"updated"`
	Available          int32                        `json:"available"`
	Generation         int64                        `json:"generation"`
	ObservedGeneration int64                        `json:"observed_generation"`
	Rolling            bool                         `json:"rolling"`
	Containers         []DeploymentContainerRuntime `json:"containers"`
	Domains            []string                     `json:"domains"`
	ManagedDomains     []string                     `json:"managed_domains"`
	ManagedIngress     string                       `json:"managed_ingress"`
	IngressClass       string                       `json:"ingress_class"`
	ServiceName        string                       `json:"service_name"`
	ServiceCandidates  []DeploymentServiceOption    `json:"service_candidates"`
}

// Pod Pod信息
type Pod struct {
	ClusterID  string            `json:"cluster_id"`
	Namespace  string            `json:"namespace"`
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Phase      string            `json:"phase"`
	Reason     string            `json:"reason,omitempty"`
	Node       string            `json:"node"`
	IP         string            `json:"ip"`
	Labels     map[string]string `json:"labels"`
	OwnerKind  string            `json:"owner_kind"`
	OwnerName  string            `json:"owner_name"`
	Containers []Container       `json:"containers"`
	Restarts   int32             `json:"restarts"`
	Ready      int32             `json:"ready"`
	Total      int32             `json:"total"`
	CreatedAt  time.Time         `json:"created_at"`
}

// Container 容器信息
type Container struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Ready bool   `json:"ready"`
	State string `json:"state"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	ClusterID string            `json:"cluster_id"`
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	ClusterIP string            `json:"cluster_ip"`
	Ports     []string          `json:"ports"`
	Selector  map[string]string `json:"selector"`
	CreatedAt time.Time         `json:"created_at"`
}

// IngressInfo Ingress信息
type IngressInfo struct {
	ClusterID string    `json:"cluster_id"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	ClassName string    `json:"class_name"`
	Hosts     []string  `json:"hosts"`
	CreatedAt time.Time `json:"created_at"`
}

// ConfigMapInfo ConfigMap信息
type ConfigMapInfo struct {
	ClusterID string    `json:"cluster_id"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	DataKeys  []string  `json:"data_keys"`
	CreatedAt time.Time `json:"created_at"`
}

// SecretInfo Secret信息（不返回明文数据）
type SecretInfo struct {
	ClusterID string    `json:"cluster_id"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	DataKeys  []string  `json:"data_keys"`
	CreatedAt time.Time `json:"created_at"`
}

// StorageClassInfo 存储类信息
type StorageClassInfo struct {
	ClusterID      string    `json:"cluster_id"`
	Name           string    `json:"name"`
	Provisioner    string    `json:"provisioner"`
	ReclaimPolicy  string    `json:"reclaim_policy"`
	VolumeBinding  string    `json:"volume_binding"`
	AllowExpansion bool      `json:"allow_expansion"`
	CreatedAt      time.Time `json:"created_at"`
}

// PersistentVolumeInfo PV信息
type PersistentVolumeInfo struct {
	ClusterID  string    `json:"cluster_id"`
	Name       string    `json:"name"`
	Capacity   string    `json:"capacity"`
	AccessMode []string  `json:"access_modes"`
	Status     string    `json:"status"`
	StorageCls string    `json:"storage_class"`
	Claim      string    `json:"claim"`
	CreatedAt  time.Time `json:"created_at"`
}

// PersistentVolumeClaimInfo PVC信息
type PersistentVolumeClaimInfo struct {
	ClusterID  string    `json:"cluster_id"`
	Namespace  string    `json:"namespace"`
	Name       string    `json:"name"`
	Capacity   string    `json:"capacity"`
	AccessMode []string  `json:"access_modes"`
	Status     string    `json:"status"`
	StorageCls string    `json:"storage_class"`
	VolumeName string    `json:"volume_name"`
	CreatedAt  time.Time `json:"created_at"`
}

// EventInfo 事件信息
type EventInfo struct {
	ClusterID string    `json:"cluster_id"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Involved  string    `json:"involved_object"`
	Count     int32     `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}
