package cicd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"github.com/lazyautoops/lazy-auto-ops/plugins/notify"
	"github.com/lazyautoops/lazy-auto-ops/plugins/workorder"
	"gorm.io/gorm"
)

type cicdApprovalFormData struct {
	Source      string            `json:"source"`
	PipelineID  string            `json:"pipeline_id"`
	Pipeline    string            `json:"pipeline_name"`
	Provider    string            `json:"provider"`
	Parameters  map[string]string `json:"parameters"`
	Reason      string            `json:"reason,omitempty"`
	Trigger     string            `json:"trigger"`
	TriggerBy   string            `json:"trigger_by"`
	RequestedAt time.Time         `json:"requested_at"`
}

type CICDHandler struct {
	db        *gorm.DB
	secretKey string
	scheduler *Scheduler
}

func NewCICDHandler(db *gorm.DB, secretKey string) *CICDHandler {
	h := &CICDHandler{db: db, secretKey: secretKey}
	h.scheduler = NewScheduler(db, h)
	h.scheduler.Start()
	return h
}

type CICDCredentialOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Username string `json:"username"`
	Provider string `json:"provider,omitempty"`
}

func pickCredentialSecretForProvider(provider string, cred *cmdb.Credential) (string, error) {
	if cred == nil {
		return "", fmt.Errorf("凭据为空")
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	credentialType := strings.ToLower(strings.TrimSpace(cred.Type))
	candidates := []string{}

	switch provider {
	case "jenkins":
		candidates = []string{cred.Password, cred.SecretKey, cred.AccessKey, cred.Passphrase}
	case "gitlab", "github", "argocd":
		if credentialType == "api" {
			candidates = []string{cred.SecretKey, cred.Password, cred.AccessKey}
		} else {
			candidates = []string{cred.Password, cred.SecretKey, cred.AccessKey, cred.Passphrase}
		}
	default:
		candidates = []string{cred.Password, cred.SecretKey, cred.AccessKey, cred.Passphrase}
	}

	for _, item := range candidates {
		if strings.TrimSpace(item) != "" {
			return item, nil
		}
	}
	return "", fmt.Errorf("凭据缺少可用密钥字段")
}

func (h *CICDHandler) loadCredential(credentialID string) (*cmdb.Credential, error) {
	credentialID = strings.TrimSpace(credentialID)
	if credentialID == "" {
		return nil, nil
	}
	var cred cmdb.Credential
	if err := h.db.First(&cred, "id = ?", credentialID).Error; err != nil {
		return nil, fmt.Errorf("凭据不存在或不可用")
	}
	if err := cmdb.DecryptCredentialFields(h.secretKey, &cred); err != nil {
		return nil, fmt.Errorf("凭据解密失败: %w", err)
	}
	return &cred, nil
}

func providerDisplayName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "jenkins":
		return "Jenkins"
	case "gitlab":
		return "GitLab"
	case "github":
		return "GitHub"
	case "argocd":
		return "ArgoCD"
	default:
		return provider
	}
}

func (h *CICDHandler) validatePipelineAuth(pipeline *CICDPipeline) error {
	if pipeline == nil {
		return fmt.Errorf("流水线不能为空")
	}
	pipeline.Provider = strings.ToLower(strings.TrimSpace(pipeline.Provider))
	switch pipeline.Provider {
	case "jenkins", "gitlab", "argocd", "github":
	default:
		return fmt.Errorf("不支持的 Provider: %s", pipeline.Provider)
	}

	credentialID := strings.TrimSpace(pipeline.CredentialID)
	if credentialID != "" {
		cred, err := h.loadCredential(credentialID)
		if err != nil {
			return err
		}
		if _, err := pickCredentialSecretForProvider(pipeline.Provider, cred); err != nil {
			return fmt.Errorf("%s 凭据不可用于 %s: %w", cred.Name, providerDisplayName(pipeline.Provider), err)
		}
		if pipeline.Provider == "jenkins" && strings.TrimSpace(cred.Username) == "" && strings.TrimSpace(pipeline.JenkinsUser) == "" {
			return fmt.Errorf("Jenkins 统一凭据缺少用户名，请在凭据里维护 Username 或在流水线里填写 Jenkins 用户名")
		}
		return nil
	}

	switch pipeline.Provider {
	case "jenkins":
		if strings.TrimSpace(pipeline.JenkinsUser) == "" || strings.TrimSpace(pipeline.JenkinsToken) == "" {
			return fmt.Errorf("Jenkins 用户名和 Token 不能为空（或选择统一凭据）")
		}
	case "gitlab":
		if strings.TrimSpace(pipeline.GitLabToken) == "" {
			return fmt.Errorf("GitLab Token 不能为空（或选择统一凭据）")
		}
	case "argocd":
		if strings.TrimSpace(pipeline.ArgoCDToken) == "" {
			return fmt.Errorf("ArgoCD Token 不能为空（或选择统一凭据）")
		}
	case "github":
		if strings.TrimSpace(pipeline.GitHubToken) == "" {
			return fmt.Errorf("GitHub Token 不能为空（或选择统一凭据）")
		}
	}
	return nil
}

func (h *CICDHandler) validatePipelineConfig(pipeline *CICDPipeline) error {
	if pipeline == nil {
		return fmt.Errorf("流水线不能为空")
	}
	if strings.TrimSpace(pipeline.Name) == "" {
		return fmt.Errorf("流水线名称不能为空")
	}
	if err := h.validatePipelineAuth(pipeline); err != nil {
		return err
	}
	switch pipeline.Provider {
	case "jenkins":
		if strings.TrimSpace(pipeline.JenkinsURL) == "" || strings.TrimSpace(pipeline.JenkinsJob) == "" {
			return fmt.Errorf("Jenkins URL 和 Job Name 不能为空")
		}
	case "gitlab":
		if strings.TrimSpace(pipeline.GitLabURL) == "" || strings.TrimSpace(pipeline.GitLabProjectID) == "" {
			return fmt.Errorf("GitLab URL 和 Project ID 不能为空")
		}
	case "argocd":
		if strings.TrimSpace(pipeline.ArgoCDURL) == "" || strings.TrimSpace(pipeline.ArgoCDApp) == "" {
			return fmt.Errorf("ArgoCD URL 和 App Name 不能为空")
		}
	case "github":
		if strings.TrimSpace(pipeline.GitHubRepo) == "" || strings.TrimSpace(pipeline.GitHubWorkflow) == "" {
			return fmt.Errorf("GitHub Repo 和 Workflow 不能为空")
		}
	}
	if pipeline.RequireApproval && strings.TrimSpace(pipeline.WorkorderTypeID) != "" {
		var count int64
		if err := h.db.Model(&workorder.WorkOrderType{}).Where("id = ?", strings.TrimSpace(pipeline.WorkorderTypeID)).Count(&count).Error; err != nil {
			return fmt.Errorf("工单类型校验失败: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("指定的工单类型不存在")
		}
	}
	return nil
}

func (h *CICDHandler) applyCredential(pipeline *CICDPipeline) error {
	if pipeline == nil || strings.TrimSpace(pipeline.CredentialID) == "" {
		return nil
	}

	cred, err := h.loadCredential(pipeline.CredentialID)
	if err != nil {
		return err
	}

	token, err := pickCredentialSecretForProvider(pipeline.Provider, cred)
	if err != nil {
		return fmt.Errorf("凭据与当前 Provider 不匹配: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(pipeline.Provider)) {
	case "jenkins":
		if strings.TrimSpace(cred.Username) != "" {
			pipeline.JenkinsUser = cred.Username
		}
		if strings.TrimSpace(token) != "" {
			pipeline.JenkinsToken = token
		}
	case "gitlab":
		if strings.TrimSpace(token) != "" {
			pipeline.GitLabToken = token
		}
	case "argocd":
		if strings.TrimSpace(token) != "" {
			pipeline.ArgoCDToken = token
		}
	case "github":
		if strings.TrimSpace(token) != "" {
			pipeline.GitHubToken = token
		}
	}
	return nil
}

func (h *CICDHandler) attachCredentialNames(pipelines []CICDPipeline) error {
	if len(pipelines) == 0 {
		return nil
	}
	ids := make([]string, 0, len(pipelines))
	idSet := make(map[string]struct{}, len(pipelines))
	for i := range pipelines {
		id := strings.TrimSpace(pipelines[i].CredentialID)
		if id == "" {
			continue
		}
		if _, ok := idSet[id]; ok {
			continue
		}
		idSet[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}

	var creds []cmdb.Credential
	if err := h.db.Select("id", "name").Where("id IN ?", ids).Find(&creds).Error; err != nil {
		return err
	}
	nameMap := make(map[string]string, len(creds))
	for i := range creds {
		nameMap[creds[i].ID] = creds[i].Name
	}
	for i := range pipelines {
		if name, ok := nameMap[pipelines[i].CredentialID]; ok {
			pipelines[i].CredentialName = name
		}
	}
	return nil
}

func (h *CICDHandler) ListCredentials(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
	var creds []cmdb.Credential
	if err := h.db.Order("created_at DESC").Find(&creds).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	options := make([]CICDCredentialOption, 0, len(creds))
	for i := range creds {
		if provider != "" {
			fullCred, err := h.loadCredential(creds[i].ID)
			if err != nil {
				continue
			}
			if _, err := pickCredentialSecretForProvider(provider, fullCred); err != nil {
				continue
			}
			if provider == "jenkins" && strings.TrimSpace(fullCred.Username) == "" {
				continue
			}
		}
		options = append(options, CICDCredentialOption{
			ID:       creds[i].ID,
			Name:     creds[i].Name,
			Type:     creds[i].Type,
			Username: creds[i].Username,
			Provider: provider,
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": options})
}

// ListPipelines 流水线列表
func (h *CICDHandler) ListPipelines(c *gin.Context) {
	var pipelines []CICDPipeline
	query := h.db.Order("created_at DESC")
	if provider := c.Query("provider"); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if err := h.attachCredentialNames(pipelines); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipelines})
}

// CreatePipeline 创建流水线
func (h *CICDHandler) CreatePipeline(c *gin.Context) {
	var pipeline CICDPipeline
	if err := c.ShouldBindJSON(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.validatePipelineConfig(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.db.Create(&pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if strings.TrimSpace(pipeline.CredentialID) != "" {
		var cred cmdb.Credential
		if err := h.db.Select("id", "name").First(&cred, "id = ?", pipeline.CredentialID).Error; err == nil {
			pipeline.CredentialName = cred.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// GetPipeline 获取流水线详情
func (h *CICDHandler) GetPipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	if strings.TrimSpace(pipeline.CredentialID) != "" {
		var cred cmdb.Credential
		if err := h.db.Select("id", "name").First(&cred, "id = ?", pipeline.CredentialID).Error; err == nil {
			pipeline.CredentialName = cred.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// UpdatePipeline 更新流水线
func (h *CICDHandler) UpdatePipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	var req CICDPipeline
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.validatePipelineConfig(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	updates := map[string]interface{}{
		"name":               req.Name,
		"description":        req.Description,
		"provider":           strings.ToLower(strings.TrimSpace(req.Provider)),
		"credential_id":      req.CredentialID,
		"require_approval":   req.RequireApproval,
		"workorder_type_id":  req.WorkorderTypeID,
		"notify_target_id":   req.NotifyTargetID,
		"notify_receiver":    req.NotifyReceiver,
		"jenkins_url":        req.JenkinsURL,
		"jenkins_job":        req.JenkinsJob,
		"jenkins_user":       req.JenkinsUser,
		"jenkins_token":      req.JenkinsToken,
		"git_lab_url":        req.GitLabURL,
		"git_lab_project_id": req.GitLabProjectID,
		"git_lab_token":      req.GitLabToken,
		"git_lab_ref":        req.GitLabRef,
		"argo_cd_url":        req.ArgoCDURL,
		"argo_cd_app":        req.ArgoCDApp,
		"argo_cd_token":      req.ArgoCDToken,
		"git_hub_repo":       req.GitHubRepo,
		"git_hub_token":      req.GitHubToken,
		"git_hub_workflow":   req.GitHubWorkflow,
		"parameters":         req.Parameters,
		"env_vars":           req.EnvVars,
		"status":             req.Status,
	}
	if err := h.db.Model(&pipeline).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&pipeline, "id = ?", id)
	if strings.TrimSpace(pipeline.CredentialID) != "" {
		var cred cmdb.Credential
		if err := h.db.Select("id", "name").First(&cred, "id = ?", pipeline.CredentialID).Error; err == nil {
			pipeline.CredentialName = cred.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline})
}

// DeletePipeline 删除流水线
func (h *CICDHandler) DeletePipeline(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDPipeline{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TriggerPipeline 触发流水线
func (h *CICDHandler) TriggerPipeline(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	var req struct {
		Parameters map[string]string `json:"parameters"`
		Reason     string            `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	triggerBy := c.GetString("username")
	if strings.TrimSpace(triggerBy) == "" {
		triggerBy = "system"
	}
	triggerByID := c.GetString("user_id")

	if pipeline.RequireApproval {
		order, err := h.createApprovalWorkOrder(&pipeline, req.Parameters, req.Reason, "manual", triggerBy, triggerByID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"mode":         "approval_required",
				"workorder_id": order.ID,
				"status":       "pending",
			},
		})
		return
	}

	execution, err := h.triggerBuild(&pipeline, req.Parameters, "manual", triggerBy, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.sendExecutionNotify(&pipeline, execution, "流水线已触发", "流水线已开始执行")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// ExecuteByWorkOrder 根据工单触发流水线（审批通过后执行）
func (h *CICDHandler) ExecuteByWorkOrder(c *gin.Context) {
	orderID := strings.TrimSpace(c.Param("orderID"))
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单ID不能为空"})
		return
	}

	var order workorder.WorkOrder
	if err := h.db.First(&order, "id = ?", orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "工单不存在"})
		return
	}
	if order.Status != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单未审批通过，无法执行"})
		return
	}

	var form cicdApprovalFormData
	if err := json.Unmarshal([]byte(order.FormData), &form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "工单表单数据解析失败"})
		return
	}
	if strings.ToLower(strings.TrimSpace(form.Source)) != "cicd" || strings.TrimSpace(form.PipelineID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该工单不是CI/CD触发工单"})
		return
	}

	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", form.PipelineID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "关联流水线不存在"})
		return
	}

	operator := c.GetString("username")
	if strings.TrimSpace(operator) == "" {
		operator = "system"
	}
	operatorID := c.GetString("user_id")
	updates := map[string]interface{}{
		"status":      4,
		"assignee":    operator,
		"assignee_id": operatorID,
	}
	_ = h.db.Model(&order).Updates(updates).Error
	h.addWorkOrderComment(h.db, order.ID, operator, "system", "已从CI/CD模块触发执行")

	execution, err := h.triggerBuild(&pipeline, form.Parameters, "workorder", operator, order.ID)
	if err != nil {
		_ = h.db.Model(&order).Updates(map[string]interface{}{"status": 2}).Error
		h.addWorkOrderComment(h.db, order.ID, operator, "system", "触发失败: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.sendExecutionNotify(&pipeline, execution, "工单执行已触发", "工单已进入流水线执行阶段")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

func (h *CICDHandler) resolveWorkOrderType(typeID string) (*workorder.WorkOrderType, error) {
	if strings.TrimSpace(typeID) != "" {
		var t workorder.WorkOrderType
		if err := h.db.First(&t, "id = ?", strings.TrimSpace(typeID)).Error; err != nil {
			return nil, fmt.Errorf("工单类型不存在")
		}
		return &t, nil
	}

	var preferred workorder.WorkOrderType
	if err := h.db.Where("code = ? AND enabled = ?", "change_apply", true).First(&preferred).Error; err == nil {
		return &preferred, nil
	}

	var fallback workorder.WorkOrderType
	if err := h.db.Where("enabled = ?", true).Order("created_at ASC").First(&fallback).Error; err == nil {
		return &fallback, nil
	}

	created := workorder.WorkOrderType{
		Name:        "CI/CD发布",
		Code:        "cicd_release",
		Icon:        "upload",
		Enabled:     true,
		Description: "CI/CD流水线触发审批工单",
	}
	if err := h.db.Create(&created).Error; err != nil {
		return nil, err
	}
	return &created, nil
}

func (h *CICDHandler) createWorkOrderSteps(db *gorm.DB, order *workorder.WorkOrder, orderType *workorder.WorkOrderType) error {
	if db == nil {
		db = h.db
	}
	if order == nil {
		return fmt.Errorf("工单为空")
	}
	if orderType == nil || strings.TrimSpace(orderType.FlowID) == "" {
		order.TotalSteps = 1
		order.CurrentStep = 1
		if err := db.Model(order).Updates(map[string]interface{}{"total_steps": 1, "current_step": 1}).Error; err != nil {
			return err
		}
		defaultStep := workorder.WorkOrderStep{
			OrderID: order.ID,
			Step:    1,
			Name:    "默认审批",
			Status:  0,
		}
		return db.Create(&defaultStep).Error
	}

	var flow workorder.WorkOrderFlow
	if err := db.First(&flow, "id = ?", orderType.FlowID).Error; err != nil {
		return err
	}
	var steps []map[string]interface{}
	if err := json.Unmarshal([]byte(flow.Steps), &steps); err != nil {
		return err
	}
	if len(steps) == 0 {
		return h.createWorkOrderSteps(db, order, nil)
	}

	order.TotalSteps = len(steps)
	order.CurrentStep = 1
	if err := db.Model(order).Updates(map[string]interface{}{
		"total_steps":  len(steps),
		"current_step": 1,
	}).Error; err != nil {
		return err
	}
	for i := range steps {
		step := workorder.WorkOrderStep{
			OrderID: order.ID,
			Step:    i + 1,
			Status:  0,
		}
		if name, ok := steps[i]["name"].(string); ok && strings.TrimSpace(name) != "" {
			step.Name = name
		} else {
			step.Name = fmt.Sprintf("审批步骤%d", i+1)
		}
		if approver, ok := steps[i]["approver"].(string); ok {
			step.Approver = approver
		}
		if err := db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *CICDHandler) addWorkOrderComment(db *gorm.DB, orderID, username, commentType, content string) {
	if db == nil {
		db = h.db
	}
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(content) == "" {
		return
	}
	comment := workorder.WorkOrderComment{
		OrderID:  orderID,
		Username: username,
		Type:     commentType,
		Content:  content,
	}
	_ = db.Create(&comment).Error
}

func (h *CICDHandler) createApprovalWorkOrder(pipeline *CICDPipeline, params map[string]string, reason, trigger, triggerBy, triggerByID string) (*workorder.WorkOrder, error) {
	if pipeline == nil {
		return nil, fmt.Errorf("流水线不存在")
	}
	if params == nil {
		params = map[string]string{}
	}
	orderType, err := h.resolveWorkOrderType(pipeline.WorkorderTypeID)
	if err != nil {
		return nil, err
	}

	form := cicdApprovalFormData{
		Source:      "cicd",
		PipelineID:  pipeline.ID,
		Pipeline:    pipeline.Name,
		Provider:    pipeline.Provider,
		Parameters:  params,
		Reason:      strings.TrimSpace(reason),
		Trigger:     trigger,
		TriggerBy:   triggerBy,
		RequestedAt: time.Now(),
	}
	formJSON, _ := json.Marshal(form)

	content := fmt.Sprintf("流水线「%s」触发审批申请\nProvider: %s", pipeline.Name, providerDisplayName(pipeline.Provider))
	if form.Reason != "" {
		content += "\n触发原因: " + form.Reason
	}

	order := &workorder.WorkOrder{
		Title:       fmt.Sprintf("CI/CD发布审批 - %s", pipeline.Name),
		TypeID:      orderType.ID,
		TypeName:    orderType.Name,
		Content:     content,
		FormData:    string(formJSON),
		Priority:    2,
		Status:      0,
		Submitter:   triggerBy,
		SubmitterID: triggerByID,
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if err := h.createWorkOrderSteps(tx, order, orderType); err != nil {
			return err
		}
		h.addWorkOrderComment(tx, order.ID, triggerBy, "system", "CI/CD触发申请已创建，等待审批")
		return nil
	}); err != nil {
		return nil, err
	}

	if strings.TrimSpace(pipeline.NotifyTargetID) != "" {
		title := fmt.Sprintf("【待审批】CI/CD发布申请：%s", pipeline.Name)
		body := fmt.Sprintf("工单ID: %s\n提交人: %s\n触发方式: %s", order.ID, triggerBy, trigger)
		if form.Reason != "" {
			body += "\n原因: " + form.Reason
		}
		if err := notify.SendByTarget(h.db, pipeline.NotifyTargetID, title, body, strings.TrimSpace(pipeline.NotifyReceiver), "workorder", order.ID); err != nil {
			h.addWorkOrderComment(h.db, order.ID, "system", "system", "通知发送失败: "+err.Error())
		}
	}
	return order, nil
}

func (h *CICDHandler) sendExecutionNotify(pipeline *CICDPipeline, execution *CICDExecution, titlePrefix, content string) {
	if pipeline == nil || execution == nil || strings.TrimSpace(pipeline.NotifyTargetID) == "" {
		return
	}
	title := fmt.Sprintf("【%s】%s", titlePrefix, pipeline.Name)
	body := fmt.Sprintf("%s\n执行ID: %s\n触发方式: %s\n触发人: %s", content, execution.ID, execution.Trigger, execution.TriggerBy)
	if strings.TrimSpace(execution.WorkOrderID) != "" {
		body += "\n关联工单: " + execution.WorkOrderID
	}
	_ = notify.SendByTarget(h.db, pipeline.NotifyTargetID, title, body, strings.TrimSpace(pipeline.NotifyReceiver), "cicd", execution.ID)
}

// triggerBuild 触发构建
func (h *CICDHandler) triggerBuild(pipeline *CICDPipeline, params map[string]string, trigger, triggerBy, workOrderID string) (*CICDExecution, error) {
	resolvedPipeline := *pipeline
	if err := h.applyCredential(&resolvedPipeline); err != nil {
		return nil, err
	}

	paramsJSON, _ := json.Marshal(params)
	execution := &CICDExecution{
		PipelineID:   resolvedPipeline.ID,
		PipelineName: resolvedPipeline.Name,
		Provider:     resolvedPipeline.Provider,
		WorkOrderID:  strings.TrimSpace(workOrderID),
		Status:       0,
		Trigger:      trigger,
		TriggerBy:    triggerBy,
		Parameters:   string(paramsJSON),
		StartedAt:    time.Now(),
	}

	var remoteBuildID string
	var err error

	switch resolvedPipeline.Provider {
	case "jenkins":
		remoteBuildID, err = h.triggerJenkins(&resolvedPipeline, params)
	case "gitlab":
		remoteBuildID, err = h.triggerGitLab(&resolvedPipeline, params)
	case "argocd":
		remoteBuildID, err = h.triggerArgoCD(&resolvedPipeline)
	case "github":
		remoteBuildID, err = h.triggerGitHub(&resolvedPipeline, params)
	default:
		err = fmt.Errorf("不支持的Provider: %s", resolvedPipeline.Provider)
	}

	if err != nil {
		execution.Status = 2
		execution.Logs = err.Error()
	} else {
		execution.RemoteBuildID = remoteBuildID
	}

	h.db.Create(execution)

	// 异步轮询状态
	if err == nil {
		go h.pollBuildStatus(execution, &resolvedPipeline)
	}

	return execution, err
}

// triggerJenkins 触发Jenkins构建
func (h *CICDHandler) triggerJenkins(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("%s/job/%s/build", strings.TrimSuffix(pipeline.JenkinsURL, "/"), pipeline.JenkinsJob)
	if len(params) > 0 {
		url = fmt.Sprintf("%s/job/%s/buildWithParameters", strings.TrimSuffix(pipeline.JenkinsURL, "/"), pipeline.JenkinsJob)
	}

	req, _ := http.NewRequest("POST", url, nil)
	req.SetBasicAuth(pipeline.JenkinsUser, pipeline.JenkinsToken)

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Jenkins返回错误: %d, %s", resp.StatusCode, string(body))
	}

	// 获取队列ID
	queueURL := resp.Header.Get("Location")
	if queueURL != "" {
		return queueURL, nil
	}
	return "triggered", nil
}

// triggerGitLab 触发GitLab Pipeline
func (h *CICDHandler) triggerGitLab(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipeline",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), pipeline.GitLabProjectID)

	payload := map[string]interface{}{
		"ref": pipeline.GitLabRef,
	}
	if len(params) > 0 {
		variables := make([]map[string]string, 0)
		for k, v := range params {
			variables = append(variables, map[string]string{"key": k, "value": v})
		}
		payload["variables"] = variables
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID int `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ID == 0 {
		return "", fmt.Errorf("GitLab触发失败")
	}
	return fmt.Sprintf("%d", result.ID), nil
}

// triggerArgoCD 触发ArgoCD同步
func (h *CICDHandler) triggerArgoCD(pipeline *CICDPipeline) (string, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync",
		strings.TrimSuffix(pipeline.ArgoCDURL, "/"), pipeline.ArgoCDApp)

	req, _ := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+pipeline.ArgoCDToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ArgoCD同步失败: %d", resp.StatusCode)
	}
	return "sync-" + time.Now().Format("20060102150405"), nil
}

// triggerGitHub 触发GitHub Actions
func (h *CICDHandler) triggerGitHub(pipeline *CICDPipeline, params map[string]string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches",
		pipeline.GitHubRepo, pipeline.GitHubWorkflow)

	payload := map[string]interface{}{
		"ref": "main",
	}
	if len(params) > 0 {
		payload["inputs"] = params
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "token "+pipeline.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GitHub Actions触发失败: %d", resp.StatusCode)
	}
	return "dispatch-" + time.Now().Format("20060102150405"), nil
}

// pollBuildStatus 轮询构建状态
func (h *CICDHandler) pollBuildStatus(execution *CICDExecution, pipeline *CICDPipeline) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 360; i++ { // 最多1小时
		<-ticker.C

		var status int
		var logs string

		switch pipeline.Provider {
		case "jenkins":
			status, logs = h.getJenkinsStatus(pipeline, execution.RemoteBuildID)
		case "gitlab":
			status, logs = h.getGitLabStatus(pipeline, execution.RemoteBuildID)
		default:
			status = 1
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status": status,
			"logs":   logs,
		}
		if status != 0 {
			updates["finished_at"] = now
			updates["duration"] = int(now.Sub(execution.StartedAt).Seconds())
		}
		h.db.Model(execution).Updates(updates)

		if status != 0 {
			execution.Status = status
			execution.Logs = logs
			execution.FinishedAt = &now
			execution.Duration = int(now.Sub(execution.StartedAt).Seconds())
			h.handleExecutionFinished(execution, pipeline)
			return
		}
	}
}

func (h *CICDHandler) handleExecutionFinished(execution *CICDExecution, pipeline *CICDPipeline) {
	if execution == nil {
		return
	}

	switch execution.Status {
	case 1:
		h.sendExecutionNotify(pipeline, execution, "执行成功", "流水线执行成功")
	case 2:
		h.sendExecutionNotify(pipeline, execution, "执行失败", "流水线执行失败")
	case 3:
		h.sendExecutionNotify(pipeline, execution, "执行取消", "流水线执行已取消")
	}

	if strings.TrimSpace(execution.WorkOrderID) == "" {
		return
	}
	var order workorder.WorkOrder
	if err := h.db.First(&order, "id = ?", execution.WorkOrderID).Error; err != nil {
		return
	}

	switch execution.Status {
	case 1:
		now := time.Now()
		_ = h.db.Model(&order).Updates(map[string]interface{}{
			"status":       5,
			"completed_at": now,
		}).Error
		h.addWorkOrderComment(h.db, order.ID, "system", "system", "流水线执行成功，工单自动完成")
	case 2:
		_ = h.db.Model(&order).Updates(map[string]interface{}{"status": 2}).Error
		h.addWorkOrderComment(h.db, order.ID, "system", "system", "流水线执行失败，请检查执行日志后重新触发")
	case 3:
		_ = h.db.Model(&order).Updates(map[string]interface{}{"status": 6}).Error
		h.addWorkOrderComment(h.db, order.ID, "system", "system", "流水线执行已取消，工单已取消")
	}
}

func (h *CICDHandler) getJenkinsStatus(pipeline *CICDPipeline, queueURL string) (int, string) {
	// 简化实现
	return 1, "Build completed"
}

func (h *CICDHandler) getGitLabStatus(pipeline *CICDPipeline, pipelineID string) (int, string) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/pipelines/%s",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), pipeline.GitLabProjectID, pipelineID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	switch result.Status {
	case "success":
		return 1, "Pipeline success"
	case "failed":
		return 2, "Pipeline failed"
	case "canceled":
		return 3, "Pipeline canceled"
	default:
		return 0, "Running: " + result.Status
	}
}

func (h *CICDHandler) upsertCICDJob(job *CICDJob) error {
	var existing CICDJob
	err := h.db.Where("pipeline_id = ? AND remote_id = ?", job.PipelineID, job.RemoteID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.db.Create(job).Error
		}
		return err
	}

	existing.Name = job.Name
	existing.LastBuildNo = job.LastBuildNo
	existing.LastStatus = job.LastStatus
	existing.LastBuildAt = job.LastBuildAt
	return h.db.Save(&existing).Error
}

func (h *CICDHandler) syncJenkinsJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.JenkinsURL) == "" || strings.TrimSpace(pipeline.JenkinsJob) == "" {
		return nil, fmt.Errorf("Jenkins配置不完整")
	}

	apiURL := fmt.Sprintf("%s/job/%s/api/json",
		strings.TrimSuffix(pipeline.JenkinsURL, "/"),
		url.PathEscape(pipeline.JenkinsJob))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.JenkinsUser != "" || pipeline.JenkinsToken != "" {
		req.SetBasicAuth(pipeline.JenkinsUser, pipeline.JenkinsToken)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jenkins同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		Name      string `json:"name"`
		Color     string `json:"color"`
		LastBuild struct {
			Number    int   `json:"number"`
			Timestamp int64 `json:"timestamp"`
		} `json:"lastBuild"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	lastStatus := "unknown"
	color := strings.ToLower(result.Color)
	switch {
	case strings.Contains(color, "blue") || strings.Contains(color, "green"):
		lastStatus = "success"
	case strings.Contains(color, "red"):
		lastStatus = "failed"
	case strings.Contains(color, "disabled"):
		lastStatus = "disabled"
	case strings.Contains(color, "anime"):
		lastStatus = "running"
	}

	var lastBuildAt *time.Time
	if result.LastBuild.Timestamp > 0 {
		t := time.UnixMilli(result.LastBuild.Timestamp)
		lastBuildAt = &t
	}

	name := strings.TrimSpace(result.Name)
	if name == "" {
		name = pipeline.JenkinsJob
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.JenkinsJob,
		LastBuildNo: result.LastBuild.Number,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncGitLabJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.GitLabURL) == "" || strings.TrimSpace(pipeline.GitLabProjectID) == "" {
		return nil, fmt.Errorf("GitLab配置不完整")
	}

	projectID := url.PathEscape(pipeline.GitLabProjectID)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/pipelines?per_page=1",
		strings.TrimSuffix(pipeline.GitLabURL, "/"), projectID)
	if ref := strings.TrimSpace(pipeline.GitLabRef); ref != "" {
		apiURL += "&ref=" + url.QueryEscape(ref)
	}

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("PRIVATE-TOKEN", pipeline.GitLabToken)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var runs []struct {
		ID        int    `json:"id"`
		Status    string `json:"status"`
		Ref       string `json:"ref"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&runs); err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return &CICDJob{
			PipelineID: pipeline.ID,
			Name:       pipeline.Name,
			RemoteID:   pipeline.GitLabProjectID,
			LastStatus: "unknown",
		}, nil
	}

	run := runs[0]
	lastStatus := strings.TrimSpace(run.Status)
	if lastStatus == "" {
		lastStatus = "unknown"
	}
	var lastBuildAt *time.Time
	for _, raw := range []string{run.UpdatedAt, run.CreatedAt} {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			lastBuildAt = &t
			break
		}
	}

	name := pipeline.Name
	if strings.TrimSpace(run.Ref) != "" {
		name = fmt.Sprintf("%s (%s)", pipeline.Name, run.Ref)
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.GitLabProjectID,
		LastBuildNo: run.ID,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncGitHubJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.GitHubRepo) == "" || strings.TrimSpace(pipeline.GitHubWorkflow) == "" {
		return nil, fmt.Errorf("GitHub配置不完整")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/runs?per_page=1",
		strings.TrimSpace(pipeline.GitHubRepo), url.PathEscape(strings.TrimSpace(pipeline.GitHubWorkflow)))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+pipeline.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		WorkflowRuns []struct {
			ID         int64  `json:"id"`
			RunNumber  int    `json:"run_number"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			UpdatedAt  string `json:"updated_at"`
			CreatedAt  string `json:"created_at"`
		} `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.WorkflowRuns) == 0 {
		return &CICDJob{
			PipelineID: pipeline.ID,
			Name:       pipeline.Name,
			RemoteID:   pipeline.GitHubWorkflow,
			LastStatus: "unknown",
		}, nil
	}

	run := result.WorkflowRuns[0]
	status := strings.TrimSpace(run.Conclusion)
	if status == "" {
		status = strings.TrimSpace(run.Status)
	}
	if status == "" {
		status = "unknown"
	}

	var lastBuildAt *time.Time
	for _, raw := range []string{run.UpdatedAt, run.CreatedAt} {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			lastBuildAt = &t
			break
		}
	}

	name := strings.TrimSpace(run.Name)
	if name == "" {
		name = pipeline.Name
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.GitHubWorkflow,
		LastBuildNo: run.RunNumber,
		LastStatus:  status,
		LastBuildAt: lastBuildAt,
	}, nil
}

func (h *CICDHandler) syncArgoCDJob(pipeline *CICDPipeline) (*CICDJob, error) {
	if strings.TrimSpace(pipeline.ArgoCDURL) == "" || strings.TrimSpace(pipeline.ArgoCDApp) == "" {
		return nil, fmt.Errorf("ArgoCD配置不完整")
	}

	apiURL := fmt.Sprintf("%s/api/v1/applications/%s",
		strings.TrimSuffix(pipeline.ArgoCDURL, "/"), url.PathEscape(strings.TrimSpace(pipeline.ArgoCDApp)))
	req, _ := http.NewRequest("GET", apiURL, nil)
	if pipeline.ArgoCDToken != "" {
		req.Header.Set("Authorization", "Bearer "+pipeline.ArgoCDToken)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ArgoCD同步失败: %d, %s", resp.StatusCode, string(body))
	}

	var result struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			OperationState struct {
				Phase      string `json:"phase"`
				FinishedAt string `json:"finishedAt"`
			} `json:"operationState"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	lastStatus := strings.TrimSpace(result.Status.OperationState.Phase)
	if lastStatus == "" {
		lastStatus = strings.TrimSpace(result.Status.Sync.Status)
	}
	if lastStatus == "" {
		lastStatus = strings.TrimSpace(result.Status.Health.Status)
	}
	if lastStatus == "" {
		lastStatus = "unknown"
	}

	var lastBuildAt *time.Time
	if t, err := time.Parse(time.RFC3339, result.Status.OperationState.FinishedAt); err == nil {
		lastBuildAt = &t
	}

	name := strings.TrimSpace(result.Metadata.Name)
	if name == "" {
		name = pipeline.ArgoCDApp
	}

	return &CICDJob{
		PipelineID:  pipeline.ID,
		Name:        name,
		RemoteID:    pipeline.ArgoCDApp,
		LastBuildNo: 0,
		LastStatus:  lastStatus,
		LastBuildAt: lastBuildAt,
	}, nil
}

// SyncFromRemote 从远程同步Job
func (h *CICDHandler) SyncFromRemote(c *gin.Context) {
	id := c.Param("id")
	var pipeline CICDPipeline
	if err := h.db.First(&pipeline, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}
	if err := h.applyCredential(&pipeline); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var (
		job *CICDJob
		err error
	)

	switch strings.ToLower(strings.TrimSpace(pipeline.Provider)) {
	case "jenkins":
		job, err = h.syncJenkinsJob(&pipeline)
	case "gitlab":
		job, err = h.syncGitLabJob(&pipeline)
	case "github":
		job, err = h.syncGitHubJob(&pipeline)
	case "argocd":
		job, err = h.syncArgoCDJob(&pipeline)
	default:
		err = fmt.Errorf("不支持的Provider: %s", pipeline.Provider)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "同步失败：未获取到Job信息"})
		return
	}
	if err := h.upsertCICDJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步成功", "data": job})
}

// ListExecutions 执行记录列表
func (h *CICDHandler) ListExecutions(c *gin.Context) {
	var executions []CICDExecution
	query := h.db.Order("started_at DESC")
	if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
		query = query.Where("pipeline_id = ?", pipelineID)
	}
	if err := query.Limit(100).Find(&executions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": executions})
}

// GetExecution 获取执行详情
func (h *CICDHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var execution CICDExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execution})
}

// GetExecutionLogs 获取执行日志
func (h *CICDHandler) GetExecutionLogs(c *gin.Context) {
	id := c.Param("id")
	var execution CICDExecution
	if err := h.db.First(&execution, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"logs": execution.Logs}})
}

// CancelExecution 取消执行
func (h *CICDHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&CICDExecution{}).Where("id = ? AND status = 0", id).Update("status", 3)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已取消"})
}

// ListSchedules 定时发布列表
func (h *CICDHandler) ListSchedules(c *gin.Context) {
	var schedules []CICDSchedule
	if err := h.db.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedules})
}

// CreateSchedule 创建定时发布
func (h *CICDHandler) CreateSchedule(c *gin.Context) {
	var schedule CICDSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// UpdateSchedule 更新定时发布
func (h *CICDHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule CICDSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "定时任务不存在"})
		return
	}
	var req CICDSchedule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	updates := map[string]interface{}{
		"name":        req.Name,
		"pipeline_id": req.PipelineID,
		"cron":        req.Cron,
		"parameters":  req.Parameters,
		"enabled":     req.Enabled,
		"last_run_at": req.LastRunAt,
		"next_run_at": req.NextRunAt,
		"description": req.Description,
	}
	if err := h.db.Model(&schedule).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&schedule, "id = ?", id)
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// DeleteSchedule 删除定时发布
func (h *CICDHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDSchedule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ToggleSchedule 启用/禁用定时发布
func (h *CICDHandler) ToggleSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule CICDSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "定时任务不存在"})
		return
	}
	schedule.Enabled = !schedule.Enabled
	h.db.Save(&schedule)
	h.scheduler.Reload()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// ListReleases 发布记录列表
func (h *CICDHandler) ListReleases(c *gin.Context) {
	var releases []CICDRelease
	query := h.db.Order("created_at DESC")
	if pipelineID := c.Query("pipeline_id"); pipelineID != "" {
		query = query.Where("pipeline_id = ?", pipelineID)
	}
	if err := query.Find(&releases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": releases})
}

// CreateRelease 创建发布记录
func (h *CICDHandler) CreateRelease(c *gin.Context) {
	var release CICDRelease
	if err := c.ShouldBindJSON(&release); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	release.Operator = c.GetString("username")
	if release.Status == 1 {
		now := time.Now()
		release.ReleaseAt = &now
	}
	if err := h.db.Create(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": release})
}

// UpdateRelease 更新发布记录
func (h *CICDHandler) UpdateRelease(c *gin.Context) {
	id := c.Param("id")
	var release CICDRelease
	if err := h.db.First(&release, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布记录不存在"})
		return
	}
	var req CICDRelease
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Status == 1 && req.ReleaseAt == nil {
		now := time.Now()
		req.ReleaseAt = &now
	}
	updates := map[string]interface{}{
		"name":          req.Name,
		"pipeline_id":   req.PipelineID,
		"pipeline_name": req.PipelineName,
		"version":       req.Version,
		"environment":   req.Environment,
		"status":        req.Status,
		"notes":         req.Notes,
		"release_at":    req.ReleaseAt,
		"operator":      c.GetString("username"),
	}
	if err := h.db.Model(&release).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	_ = h.db.First(&release, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": release})
}

// DeleteRelease 删除发布记录
func (h *CICDHandler) DeleteRelease(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CICDRelease{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// HandleWebhook 处理Webhook
func (h *CICDHandler) HandleWebhook(c *gin.Context) {
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	body, _ := io.ReadAll(c.Request.Body)

	event := parseWebhookEvent(provider, body, c.GetHeader("X-GitHub-Event"))
	if event.Actor == "" {
		event.Actor = "webhook:" + provider
	}

	var pipelines []CICDPipeline
	if err := h.db.Where("provider = ? AND status = ?", provider, 1).Find(&pipelines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	triggered := make([]string, 0, len(pipelines))
	failed := make([]string, 0)
	skipped := 0
	for i := range pipelines {
		pipeline := &pipelines[i]
		if !event.matchesPipeline(pipeline) {
			skipped++
			continue
		}
		if pipeline.RequireApproval {
			order, err := h.createApprovalWorkOrder(pipeline, event.Params, "Webhook触发审批", "webhook", event.Actor, "")
			if err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", pipeline.Name, err))
				continue
			}
			triggered = append(triggered, "workorder:"+order.ID)
			continue
		}
		execution, err := h.triggerBuild(pipeline, event.Params, "webhook", event.Actor, "")
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", pipeline.Name, err))
			continue
		}
		h.sendExecutionNotify(pipeline, execution, "流水线已触发", "Webhook 已触发流水线执行")
		triggered = append(triggered, execution.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("Webhook received from %s", provider),
		"data": gin.H{
			"provider":   provider,
			"event":      event.Name,
			"ref":        event.Ref,
			"repo":       event.Repo,
			"actor":      event.Actor,
			"matched":    len(triggered),
			"failed":     len(failed),
			"skipped":    skipped,
			"executions": triggered,
			"errors":     failed,
		},
	})
}

type cicdWebhookEvent struct {
	Name   string
	Repo   string
	Ref    string
	Actor  string
	Params map[string]string
}

func parseWebhookEvent(provider string, body []byte, githubEvent string) cicdWebhookEvent {
	event := cicdWebhookEvent{
		Name:   strings.TrimSpace(githubEvent),
		Params: map[string]string{},
	}

	switch provider {
	case "github":
		var payload struct {
			Ref        string `json:"ref"`
			Workflow   string `json:"workflow"`
			Action     string `json:"action"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Sender struct {
				Login string `json:"login"`
			} `json:"sender"`
		}
		_ = json.Unmarshal(body, &payload)
		event.Ref = normalizeGitRef(payload.Ref)
		event.Repo = strings.TrimSpace(payload.Repository.FullName)
		event.Actor = strings.TrimSpace(payload.Sender.Login)
		if event.Name == "" {
			event.Name = strings.TrimSpace(payload.Action)
		}
		if wf := strings.TrimSpace(payload.Workflow); wf != "" {
			event.Params["workflow"] = wf
		}
	case "gitlab":
		var payload struct {
			ObjectKind string `json:"object_kind"`
			Ref        string `json:"ref"`
			Project    struct {
				PathWithNamespace string `json:"path_with_namespace"`
			} `json:"project"`
			UserUsername string `json:"user_username"`
		}
		_ = json.Unmarshal(body, &payload)
		event.Name = strings.TrimSpace(payload.ObjectKind)
		event.Ref = normalizeGitRef(payload.Ref)
		event.Repo = strings.TrimSpace(payload.Project.PathWithNamespace)
		event.Actor = strings.TrimSpace(payload.UserUsername)
	default:
		event.Name = provider
	}

	if event.Ref != "" {
		event.Params["ref"] = event.Ref
	}
	if event.Name != "" {
		event.Params["event"] = event.Name
	}
	return event
}

func (e cicdWebhookEvent) matchesPipeline(p *CICDPipeline) bool {
	if p == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(p.Provider)) {
	case "github":
		if e.Repo != "" && strings.TrimSpace(p.GitHubRepo) != "" && !strings.EqualFold(strings.TrimSpace(e.Repo), strings.TrimSpace(p.GitHubRepo)) {
			return false
		}
		if e.Ref != "" && strings.TrimSpace(p.GitHubWorkflow) != "" {
			// workflow 字段用于 API dispatch，需要时再按 workflow 过滤。
			if wf := strings.TrimSpace(e.Params["workflow"]); wf != "" && !strings.EqualFold(wf, strings.TrimSpace(p.GitHubWorkflow)) {
				return false
			}
		}
	case "gitlab":
		project := strings.TrimSpace(p.GitLabProjectID)
		if e.Repo != "" && project != "" && strings.Contains(project, "/") &&
			!strings.EqualFold(strings.TrimSpace(e.Repo), project) {
			return false
		}
		if e.Ref != "" && strings.TrimSpace(p.GitLabRef) != "" &&
			!strings.EqualFold(strings.TrimSpace(e.Ref), strings.TrimSpace(p.GitLabRef)) {
			return false
		}
	}
	return true
}

func normalizeGitRef(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "refs/heads/")
	raw = strings.TrimPrefix(raw, "refs/tags/")
	return raw
}
