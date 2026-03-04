package jump

import (
	"time"

	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

// JumpAsset 堡垒机资产
// AssetType: host, k8s, database
// Protocol: ssh, k8s, mysql, postgres, redis, docker
// Source: manual, cmdb_host, k8s_cluster, docker_host, cmdb_database
type JumpAsset struct {
	core.BaseModel
	Name         string `gorm:"size:128;index" json:"name"`
	AssetType    string `gorm:"size:32;index" json:"asset_type"`
	Protocol     string `gorm:"size:32;index" json:"protocol"`
	Address      string `gorm:"size:256;index" json:"address"`
	Port         int    `gorm:"default:22" json:"port"`
	Cluster      string `gorm:"size:128" json:"cluster"`
	Namespace    string `gorm:"size:128" json:"namespace"`
	CredentialID string `gorm:"size:36" json:"credential_id"`
	Source       string `gorm:"size:32;index:idx_jump_asset_source_ref,priority:1" json:"source"`
	SourceRef    string `gorm:"size:64;index:idx_jump_asset_source_ref,priority:2" json:"source_ref"`
	Tags         string `gorm:"size:256" json:"tags"`
	Description  string `gorm:"size:512" json:"description"`
	Enabled      bool   `gorm:"default:true" json:"enabled"`
}

// JumpAccount 堡垒机登录账号
type JumpAccount struct {
	core.BaseModel
	Name        string `gorm:"size:128;index" json:"name"`
	Username    string `gorm:"size:128" json:"username"`
	AuthType    string `gorm:"size:32" json:"auth_type"` // password, key, token
	Secret      string `gorm:"type:text" json:"-"`
	Description string `gorm:"size:512" json:"description"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
}

// JumpPermissionPolicy 会话授权策略
type JumpPermissionPolicy struct {
	core.BaseModel
	Name           string     `gorm:"size:128;index" json:"name"`
	UserID         string     `gorm:"size:36;index" json:"user_id"`
	RoleCode       string     `gorm:"size:64;index" json:"role_code"`
	AssetID        string     `gorm:"size:36;index" json:"asset_id"`
	AccountID      string     `gorm:"size:36;index" json:"account_id"`
	Protocol       string     `gorm:"size:32;index" json:"protocol"`
	RequireApprove bool       `gorm:"default:false" json:"require_approve"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Status         int        `gorm:"default:1;index" json:"status"` // 1 active, 0 disabled
	Description    string     `gorm:"size:512" json:"description"`
}

// JumpSession 会话记录
type JumpSession struct {
	core.BaseModel
	SessionNo      string     `gorm:"size:64;uniqueIndex" json:"session_no"`
	UserID         string     `gorm:"size:36;index" json:"user_id"`
	Username       string     `gorm:"size:128;index" json:"username"`
	RoleCode       string     `gorm:"size:64;index" json:"role_code"`
	AssetID        string     `gorm:"size:36;index" json:"asset_id"`
	AssetName      string     `gorm:"size:128" json:"asset_name"`
	AccountID      string     `gorm:"size:36;index" json:"account_id"`
	AccountName    string     `gorm:"size:128" json:"account_name"`
	Protocol       string     `gorm:"size:32;index" json:"protocol"`
	SourceIP       string     `gorm:"size:64" json:"source_ip"`
	Status         string     `gorm:"size:32;index" json:"status"` // pending_approval, active, closed, blocked, rejected
	StartedAt      time.Time  `gorm:"index" json:"started_at"`
	LastCommandAt  *time.Time `json:"last_command_at"`
	EndedAt        *time.Time `gorm:"index" json:"ended_at"`
	DurationSec    int        `json:"duration_sec"`
	CommandCount   int        `json:"command_count"`
	CloseReason    string     `gorm:"size:256" json:"close_reason"`
	RelaySessionID string     `gorm:"size:36;index" json:"relay_session_id"`
}

// JumpCommandRule 命令风控规则
type JumpCommandRule struct {
	core.BaseModel
	Name        string `gorm:"size:128;index" json:"name"`
	Pattern     string `gorm:"type:text" json:"pattern"`
	MatchType   string `gorm:"size:32;default:contains" json:"match_type"`  // contains, prefix, exact, regex
	RuleKind    string `gorm:"size:32;default:risk;index" json:"rule_kind"` // risk, allow
	Protocol    string `gorm:"size:32;index" json:"protocol"`               // ssh, docker, k8s, ...; 空=全部
	Severity    string `gorm:"size:32;default:warning" json:"severity"`     // critical, warning, info
	Action      string `gorm:"size:32;default:alert" json:"action"`         // alert, block
	Priority    int    `gorm:"default:100;index" json:"priority"`
	Enabled     bool   `gorm:"default:true;index" json:"enabled"`
	Description string `gorm:"size:512" json:"description"`
}

// JumpCommand 命令审计
type JumpCommand struct {
	core.BaseModel
	SessionID     string    `gorm:"size:36;index" json:"session_id"`
	Username      string    `gorm:"size:128" json:"username"`
	Command       string    `gorm:"type:text" json:"command"`
	ResultCode    int       `json:"result_code"`
	OutputSnippet string    `gorm:"type:text" json:"output_snippet"`
	RuleID        string    `gorm:"size:256;index" json:"rule_id"`
	RuleName      string    `gorm:"size:512" json:"rule_name"`
	MatchedRules  string    `gorm:"type:text" json:"matched_rules"`
	WhitelistHit  bool      `gorm:"default:false;index" json:"whitelist_hit"`
	RiskLevel     string    `gorm:"size:32;index" json:"risk_level"`
	RiskAction    string    `gorm:"size:32" json:"risk_action"`
	RiskReason    string    `gorm:"size:512" json:"risk_reason"`
	Blocked       bool      `gorm:"default:false;index" json:"blocked"`
	AlertID       string    `gorm:"size:36;index" json:"alert_id"`
	ExecutedAt    time.Time `gorm:"index" json:"executed_at"`
}

// SafeJumpAccount 隐去密钥字段的账号响应
type SafeJumpAccount struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	HasSecret   bool      `json:"has_secret"`
}

func toSafeAccount(a *JumpAccount) SafeJumpAccount {
	if a == nil {
		return SafeJumpAccount{}
	}
	return SafeJumpAccount{
		ID:          a.ID,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
		Name:        a.Name,
		Username:    a.Username,
		AuthType:    a.AuthType,
		Description: a.Description,
		Enabled:     a.Enabled,
		HasSecret:   a.Secret != "",
	}
}
