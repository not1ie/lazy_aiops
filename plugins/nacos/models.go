package nacos

import (
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// NacosServer Nacos服务器
type NacosServer struct {
	core.BaseModel
	Name        string `json:"name" gorm:"size:100"`
	Address     string `json:"address" gorm:"size:500"` // http://ip:port
	Namespace   string `json:"namespace" gorm:"size:100"`
	Username    string `json:"username" gorm:"size:100"`
	Password    string `json:"password" gorm:"size:200"`
	Description string `json:"description" gorm:"size:500"`
	Status      int    `json:"status" gorm:"default:1"` // 1正常 0异常
}

// NacosConfig 配置项
type NacosConfig struct {
	core.BaseModel
	ServerID    string `json:"server_id" gorm:"size:36;index"`
	DataID      string `json:"data_id" gorm:"size:200;index"`
	Group       string `json:"group" gorm:"size:100;index"`
	Content     string `json:"content" gorm:"type:longtext"`
	ContentType string `json:"content_type" gorm:"size:50"` // yaml, json, properties, text
	MD5         string `json:"md5" gorm:"size:64"`
	Description string `json:"description" gorm:"size:500"`
	AppName     string `json:"app_name" gorm:"size:100"`
}

// NacosConfigHistory 配置历史
type NacosConfigHistory struct {
	core.BaseModel
	ConfigID    string    `json:"config_id" gorm:"size:36;index"`
	DataID      string    `json:"data_id" gorm:"size:200"`
	Group       string    `json:"group" gorm:"size:100"`
	Content     string    `json:"content" gorm:"type:longtext"`
	MD5         string    `json:"md5" gorm:"size:64"`
	Operator    string    `json:"operator" gorm:"size:100"`
	OperateType string    `json:"operate_type" gorm:"size:50"` // create, update, delete
	OperateAt   time.Time `json:"operate_at"`
}

// NacosService 服务
type NacosService struct {
	core.BaseModel
	ServerID      string `json:"server_id" gorm:"size:36;index"`
	ServiceName   string `json:"service_name" gorm:"size:200;index"`
	GroupName     string `json:"group_name" gorm:"size:100"`
	ClusterCount  int    `json:"cluster_count"`
	InstanceCount int    `json:"instance_count"`
	HealthyCount  int    `json:"healthy_count"`
	Metadata      string `json:"metadata" gorm:"type:text"`
}

// NacosInstance 服务实例
type NacosInstance struct {
	core.BaseModel
	ServerID    string  `json:"server_id" gorm:"size:36;index"`
	ServiceName string  `json:"service_name" gorm:"size:200;index"`
	IP          string  `json:"ip" gorm:"size:50"`
	Port        int     `json:"port"`
	Weight      float64 `json:"weight"`
	Healthy     bool    `json:"healthy"`
	Enabled     bool    `json:"enabled"`
	Ephemeral   bool    `json:"ephemeral"`
	ClusterName string  `json:"cluster_name" gorm:"size:100"`
	Metadata    string  `json:"metadata" gorm:"type:text"`
}

// NacosNamespace 命名空间
type NacosNamespace struct {
	core.BaseModel
	ServerID    string `json:"server_id" gorm:"size:36;index"`
	NamespaceID string `json:"namespace_id" gorm:"size:100"`
	Name        string `json:"name" gorm:"size:100"`
	ConfigCount int    `json:"config_count"`
	Quota       int    `json:"quota"`
	Type        int    `json:"type"`
}
