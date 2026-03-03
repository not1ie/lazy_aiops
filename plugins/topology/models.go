package topology

import (
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// ServiceNode 服务节点
type ServiceNode struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100;uniqueIndex:idx_service_nodes_lookup"`
	Type        string `json:"type" gorm:"size:50"`  // service, database, cache, mq, gateway
	Icon        string `json:"icon" gorm:"size:100"` // 图标
	Description string `json:"description" gorm:"size:500"`
	Namespace   string `json:"namespace" gorm:"size:100;default:'';uniqueIndex:idx_service_nodes_lookup"`
	Cluster     string `json:"cluster" gorm:"size:100;default:'';uniqueIndex:idx_service_nodes_lookup"`
	Endpoints   string `json:"endpoints" gorm:"type:text"` // JSON数组
	Metadata    string `json:"metadata" gorm:"type:text"`  // JSON
	Status      int    `json:"status" gorm:"default:1"`    // 1正常 2告警 3故障 0未知
	HealthURL   string `json:"health_url" gorm:"size:500"`
	X           int    `json:"x"` // 画布X坐标
	Y           int    `json:"y"` // 画布Y坐标
}

// ServiceEdge 服务关系
type ServiceEdge struct {
	core.BaseModel
	SourceID    string  `json:"source_id" gorm:"size:36;index"`
	TargetID    string  `json:"target_id" gorm:"size:36;index"`
	SourceName  string  `json:"source_name" gorm:"size:100"`
	TargetName  string  `json:"target_name" gorm:"size:100"`
	Type        string  `json:"type" gorm:"size:50"`     // http, grpc, tcp, mq
	Protocol    string  `json:"protocol" gorm:"size:50"` // HTTP/1.1, HTTP/2, gRPC
	Port        int     `json:"port"`
	Description string  `json:"description" gorm:"size:500"`
	Latency     int     `json:"latency"`    // 平均延迟ms
	QPS         int     `json:"qps"`        // 每秒请求数
	ErrorRate   float64 `json:"error_rate"` // 错误率
}

// TopologyView 拓扑视图
type TopologyView struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Description string `json:"description" gorm:"size:500"`
	Filter      string `json:"filter" gorm:"type:text"` // 过滤条件JSON
	Layout      string `json:"layout" gorm:"type:text"` // 布局配置JSON
	IsDefault   bool   `json:"is_default"`
	CreatedBy   string `json:"created_by" gorm:"size:100"`
}

// ServiceDependency 服务依赖分析
type ServiceDependency struct {
	core.BaseModel
	ServiceID       string `json:"service_id" gorm:"size:36;index"`
	ServiceName     string `json:"service_name" gorm:"size:100"`
	UpstreamCount   int    `json:"upstream_count"`   // 上游服务数
	DownstreamCount int    `json:"downstream_count"` // 下游服务数
	CriticalPath    bool   `json:"critical_path"`    // 是否关键路径
	ImpactScore     int    `json:"impact_score"`     // 影响分数
}
