package cicd

import (
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// CICDPipeline CI/CD流水线
type CICDPipeline struct {
	core.BaseModel
	Name           string `json:"name" gorm:"size:100"`
	Description    string `json:"description" gorm:"size:500"`
	Provider       string `json:"provider" gorm:"size:50"` // jenkins, gitlab, argocd, github
	CredentialID   string `json:"credential_id,omitempty" gorm:"size:36;index"`
	CredentialName string `json:"credential_name,omitempty" gorm:"-"`
	// Jenkins配置
	JenkinsURL   string `json:"jenkins_url,omitempty" gorm:"size:500"`
	JenkinsJob   string `json:"jenkins_job,omitempty" gorm:"size:200"`
	JenkinsUser  string `json:"jenkins_user,omitempty" gorm:"size:100"`
	JenkinsToken string `json:"jenkins_token,omitempty" gorm:"size:500"`
	// GitLab配置
	GitLabURL       string `json:"gitlab_url,omitempty" gorm:"size:500"`
	GitLabProjectID string `json:"gitlab_project_id,omitempty" gorm:"size:100"`
	GitLabToken     string `json:"gitlab_token,omitempty" gorm:"size:500"`
	GitLabRef       string `json:"gitlab_ref,omitempty" gorm:"size:100"`
	// ArgoCD配置
	ArgoCDURL   string `json:"argocd_url,omitempty" gorm:"size:500"`
	ArgoCDApp   string `json:"argocd_app,omitempty" gorm:"size:200"`
	ArgoCDToken string `json:"argocd_token,omitempty" gorm:"size:500"`
	// GitHub Actions
	GitHubRepo     string `json:"github_repo,omitempty" gorm:"size:200"`
	GitHubToken    string `json:"github_token,omitempty" gorm:"size:500"`
	GitHubWorkflow string `json:"github_workflow,omitempty" gorm:"size:200"`
	// 通用配置
	Parameters string `json:"parameters" gorm:"type:text"` // JSON参数
	EnvVars    string `json:"env_vars" gorm:"type:text"`   // 环境变量
	Status     int    `json:"status" gorm:"default:1"`     // 1启用 0禁用
}

// CICDJob 关联的Job
type CICDJob struct {
	core.BaseModel
	PipelineID  string     `json:"pipeline_id" gorm:"size:36;index"`
	Name        string     `json:"name" gorm:"size:100"`
	RemoteID    string     `json:"remote_id" gorm:"size:100"` // 远程Job ID
	LastBuildNo int        `json:"last_build_no"`
	LastStatus  string     `json:"last_status" gorm:"size:50"`
	LastBuildAt *time.Time `json:"last_build_at"`
}

// CICDExecution 执行记录
type CICDExecution struct {
	core.BaseModel
	PipelineID    string     `json:"pipeline_id" gorm:"size:36;index"`
	PipelineName  string     `json:"pipeline_name" gorm:"size:100"`
	Provider      string     `json:"provider" gorm:"size:50"`
	RemoteBuildID string     `json:"remote_build_id" gorm:"size:100"`
	Status        int        `json:"status" gorm:"default:0"` // 0运行中 1成功 2失败 3取消
	Trigger       string     `json:"trigger" gorm:"size:50"`  // manual, schedule, webhook
	TriggerBy     string     `json:"trigger_by" gorm:"size:100"`
	Parameters    string     `json:"parameters" gorm:"type:text"`
	Logs          string     `json:"logs" gorm:"type:longtext"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	Duration      int        `json:"duration"` // 秒
}

// CICDSchedule 定时发布
type CICDSchedule struct {
	core.BaseModel
	Name        string     `json:"name" gorm:"size:100"`
	PipelineID  string     `json:"pipeline_id" gorm:"size:36;index"`
	Cron        string     `json:"cron" gorm:"size:100"`        // cron表达式
	Parameters  string     `json:"parameters" gorm:"type:text"` // 执行参数
	Enabled     bool       `json:"enabled" gorm:"default:true"`
	LastRunAt   *time.Time `json:"last_run_at"`
	NextRunAt   *time.Time `json:"next_run_at"`
	Description string     `json:"description" gorm:"size:500"`
}

// CICDRelease 发布记录
type CICDRelease struct {
	core.BaseModel
	Name         string     `json:"name" gorm:"size:100"`
	PipelineID   string     `json:"pipeline_id" gorm:"size:36;index"`
	PipelineName string     `json:"pipeline_name" gorm:"size:100"`
	Version      string     `json:"version" gorm:"size:100"`
	Environment  string     `json:"environment" gorm:"size:50"`
	Status       int        `json:"status" gorm:"default:0"` // 0待发布 1已发布 2回滚 3失败
	Notes        string     `json:"notes" gorm:"size:1000"`
	ReleaseAt    *time.Time `json:"release_at"`
	Operator     string     `json:"operator" gorm:"size:100"`
}
