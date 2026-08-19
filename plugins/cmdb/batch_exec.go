package cmdb

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
)

type BatchExecRequest struct {
	HostIDs        []string `json:"host_ids" binding:"required"`
	Command        string   `json:"command" binding:"required"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	ForceConfirm   bool     `json:"force_confirm"`
}

type HostExecResult struct {
	HostID     string `json:"host_id"`
	Hostname   string `json:"hostname"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Status     string `json:"status"` // success, failed, skipped
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type BatchExecResponse struct {
	TotalCount   int              `json:"total_count"`
	SuccessCount int              `json:"success_count"`
	FailedCount  int              `json:"failed_count"`
	IsDangerous  bool             `json:"is_dangerous"`
	DangerReason string           `json:"danger_reason,omitempty"`
	Results      []HostExecResult `json:"results"`
}

// 常见高危命令黑名单检测正则
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+-[rRfF]*\s+(/|\*|/\*)`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=`),
	regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`),
	regexp.MustCompile(`(?i)\bdrop\s+database\b`),
	regexp.MustCompile(`(?i)\btruncate\s+table\b`),
	regexp.MustCompile(`(?i)\biptables\s+-F\b`),
	regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`), // fork bomb
	regexp.MustCompile(`(?i)\bshutdown\b`),
	regexp.MustCompile(`(?i)\binit\s+0\b`),
	regexp.MustCompile(`(?i)\breboot\b`),
}

func checkDangerousCommand(cmd string) (bool, string) {
	trimmed := strings.TrimSpace(cmd)
	for _, p := range dangerousPatterns {
		if p.MatchString(trimmed) {
			return true, fmt.Sprintf("检测到潜在高危破坏性指令 [%s]，执行可能会导致系统崩溃或数据丢失", p.String())
		}
	}
	return false, ""
}

// BatchExec 执行主机批量命令下发
func (h *HostHandler) BatchExec(c *gin.Context) {
	var req BatchExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if len(req.HostIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请至少选择一台目标主机"})
		return
	}

	req.Command = strings.TrimSpace(req.Command)
	if req.Command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "执行命令不能为空"})
		return
	}

	// 1. 高危命令拦截检查
	isDangerous, dangerReason := checkDangerousCommand(req.Command)
	if isDangerous && !req.ForceConfirm {
		c.JSON(http.StatusOK, gin.H{
			"code": 403,
			"data": BatchExecResponse{
				TotalCount:   len(req.HostIDs),
				IsDangerous:  true,
				DangerReason: dangerReason,
				Results:      make([]HostExecResult, 0),
			},
			"message": "高危命令被系统安全策略拦截，如确认需执行请勾选「强制执行」",
		})
		return
	}

	// 2. 加载目标主机及凭据
	var hosts []Host
	if err := h.db.Preload("Credential").Where("id IN ?", req.HostIDs).Find(&hosts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询主机失败: " + err.Error()})
		return
	}

	timeout := 20 * time.Second
	if req.TimeoutSeconds > 0 && req.TimeoutSeconds <= 300 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	// 3. 并发执行 SSH 命令
	results := make([]HostExecResult, len(hosts))
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, target Host) {
			defer wg.Done()
			results[idx] = h.execSingleHost(target, req.Command, timeout)
		}(i, host)
	}

	wg.Wait()

	successCount := 0
	failedCount := 0
	for _, r := range results {
		if r.Status == "success" {
			successCount++
		} else {
			failedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": BatchExecResponse{
			TotalCount:   len(hosts),
			SuccessCount: successCount,
			FailedCount:  failedCount,
			IsDangerous:  isDangerous,
			DangerReason: dangerReason,
			Results:      results,
		},
		"message": fmt.Sprintf("批量执行完成: 成功 %d 台, 失败 %d 台", successCount, failedCount),
	})
}

// execSingleHost 单机 SSH 执行
func (h *HostHandler) execSingleHost(host Host, command string, timeout time.Duration) HostExecResult {
	start := time.Now()
	res := HostExecResult{
		HostID:   host.ID,
		Hostname: host.Name,
		IP:       host.IP,
		Port:     host.Port,
		Status:   "failed",
	}
	if res.Port <= 0 {
		res.Port = 22
	}

	if host.Credential == nil {
		res.DurationMs = time.Since(start).Milliseconds()
		res.Error = "主机未关联有效凭据"
		res.Stderr = res.Error
		return res
	}

	// 复制凭据以防并发冲突
	credCopy := *host.Credential
	if err := DecryptCredentialFields(h.secretKey, &credCopy); err != nil {
		res.DurationMs = time.Since(start).Milliseconds()
		res.Error = "解密凭据失败: " + err.Error()
		res.Stderr = res.Error
		return res
	}

	client := &core.SSHClient{
		Host:     host.IP,
		Port:     res.Port,
		Username: credCopy.Username,
		Password: credCopy.Password,
		Key:      credCopy.PrivateKey,
		Timeout:  timeout,
	}

	stdout, stderr, err := client.Execute(command)
	res.DurationMs = time.Since(start).Milliseconds()
	res.Stdout = stdout
	res.Stderr = stderr

	if err != nil {
		res.Error = err.Error()
		res.Status = "failed"
		res.ExitCode = 1
	} else {
		res.Status = "success"
		res.ExitCode = 0
	}

	return res
}
