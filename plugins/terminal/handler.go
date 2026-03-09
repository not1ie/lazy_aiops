package terminal

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lazyautoops/lazy-auto-ops/internal/core"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type TerminalHandler struct {
	db       *gorm.DB
	auth     *core.AuthService
	sessions sync.Map // sessionID -> *SSHSession
}

type SSHSession struct {
	ID                  string
	Client              *ssh.Client
	Session             *ssh.Session
	StdinPipe           io.WriteCloser
	StdoutPipe          io.Reader
	StderrPipe          io.Reader
	Conn                *websocket.Conn
	ConnMu              sync.Mutex
	Recording           []RecordItem
	StartTime           time.Time
	JumpSessionID       string
	JumpAuditResolved   bool
	JumpProtocol        string
	JumpAssetName       string
	PendingCommandInput string
	JumpBlocked         bool
	JumpBlockReason     string
	ClosedByUser        bool
	OutputTail          string
	OutputTailMu        sync.Mutex
}

type terminalConnectPayload struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	KeyAuth  string `json:"key_auth"`
}

type RecordItem struct {
	Time    int64  `json:"time"` // 相对于开始时间的毫秒数
	Type    string `json:"type"` // input, output
	Content string `json:"content"`
}

type jumpCommandAudit struct {
	ID            string    `gorm:"primaryKey;size:36"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	Blocked       bool      `gorm:"index" json:"blocked"`
	AlertID       string    `gorm:"size:36;index" json:"alert_id"`
	ExecutedAt    time.Time `gorm:"index" json:"executed_at"`
}

func (jumpCommandAudit) TableName() string {
	return "jump_commands"
}

type jumpCommandRule struct {
	ID        string `gorm:"primaryKey;size:36"`
	Name      string `gorm:"size:128"`
	Pattern   string `gorm:"type:text"`
	MatchType string `gorm:"size:32"`
	RuleKind  string `gorm:"size:32"`
	Protocol  string `gorm:"size:32"`
	Severity  string `gorm:"size:32"`
	Action    string `gorm:"size:32"`
	Priority  int
	Enabled   bool
}

func (jumpCommandRule) TableName() string {
	return "jump_command_rules"
}

type jumpSessionLink struct {
	ID        string    `json:"id"`
	Protocol  string    `json:"protocol"`
	AssetName string    `json:"asset_name"`
	StartedAt time.Time `json:"started_at"`
}

type generatedAlert struct {
	ID          string    `gorm:"primaryKey;size:36"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	RuleID      string    `gorm:"size:36;index" json:"rule_id"`
	RuleName    string    `gorm:"size:128" json:"rule_name"`
	Fingerprint string    `gorm:"size:64;index" json:"fingerprint"`
	Target      string    `gorm:"size:256" json:"target"`
	Metric      string    `gorm:"size:128" json:"metric"`
	Value       string    `gorm:"size:64" json:"value"`
	Threshold   string    `gorm:"size:64" json:"threshold"`
	Severity    string    `gorm:"size:32;index" json:"severity"`
	Status      int       `gorm:"index" json:"status"`
	FiredAt     time.Time `gorm:"index" json:"fired_at"`
	GroupKey    string    `gorm:"size:128;index" json:"group_key"`
	Labels      string    `gorm:"type:text" json:"labels"`
	Annotations string    `gorm:"type:text" json:"annotations"`
}

func (generatedAlert) TableName() string {
	return "alerts"
}

type terminalCommandAuditItem struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	SessionNo  string    `json:"session_no"`
	Host       string    `json:"host"`
	AssetName  string    `json:"asset_name"`
	Operator   string    `json:"operator"`
	LoginUser  string    `json:"login_user"`
	Protocol   string    `json:"protocol"`
	Command    string    `json:"command"`
	RuleName   string    `json:"rule_name"`
	RiskLevel  string    `json:"risk_level"`
	RiskAction string    `json:"risk_action"`
	RiskReason string    `json:"risk_reason"`
	Blocked    bool      `json:"blocked"`
	ExecutedAt time.Time `json:"executed_at"`
}

type jumpRiskDecision struct {
	Matched      bool
	WhitelistHit bool
	RuleID       string
	RuleName     string
	Severity     string
	Action       string
	Reason       string
	RuleIDs      []string
	RuleNames    []string
	MatchedRules []jumpRuleHit
	AlertID      string
}

type jumpRuleHit struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RuleKind string `json:"rule_kind"`
	Action   string `json:"action"`
	Severity string `json:"severity"`
	Priority int    `json:"priority"`
	Pattern  string `json:"pattern"`
}

func NewTerminalHandler(db *gorm.DB, auth *core.AuthService) *TerminalHandler {
	return &TerminalHandler{db: db, auth: auth}
}

func normalizeConnectPayload(req *terminalConnectPayload) error {
	if req == nil {
		return errors.New("参数错误")
	}
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.KeyAuth = strings.TrimSpace(req.KeyAuth)
	if req.Host == "" {
		return errors.New("主机地址不能为空")
	}
	if req.Username == "" {
		return errors.New("用户名不能为空")
	}
	if req.Password == "" && req.KeyAuth == "" {
		return errors.New("密码和私钥至少填写一个")
	}
	if req.Port <= 0 {
		req.Port = 22
	}
	return nil
}

// ListSessions 会话列表
func (h *TerminalHandler) ListSessions(c *gin.Context) {
	var sessions []TerminalSession
	query := h.db.Model(&TerminalSession{}).Order("created_at DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if host := strings.TrimSpace(c.Query("host")); host != "" {
		query = query.Where("host LIKE ?", "%"+host+"%")
	}
	if operator := strings.TrimSpace(c.Query("operator")); operator != "" {
		query = query.Where("operator LIKE ?", "%"+operator+"%")
	}
	if username := strings.TrimSpace(c.Query("username")); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("host LIKE ? OR operator LIKE ? OR username LIKE ?", like, like, like)
	}
	if err := query.Limit(200).Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessions})
}

// GetSession 会话详情
func (h *TerminalHandler) GetSession(c *gin.Context) {
	id := c.Param("id")
	var session TerminalSession
	if err := h.db.First(&session, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// CreateSession 创建会话
func (h *TerminalHandler) CreateSession(c *gin.Context) {
	var req struct {
		HostID string `json:"host_id"`
		terminalConnectPayload
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := normalizeConnectPayload(&req.terminalConnectPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 创建会话记录
	session := TerminalSession{
		HostID:     req.HostID,
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		Password:   req.Password,
		PrivateKey: req.KeyAuth,
		UserID:     c.GetString("user_id"),
		Operator:   c.GetString("username"),
		Status:     0, // 待连接
	}
	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// UpdateSession 编辑会话（在线会话需先关闭）
func (h *TerminalHandler) UpdateSession(c *gin.Context) {
	id := c.Param("id")

	var session TerminalSession
	if err := h.db.First(&session, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	if session.Status == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "在线会话请先关闭后再编辑"})
		return
	}

	var req terminalConnectPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Port <= 0 {
		req.Port = 22
	}
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	if req.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "主机地址不能为空"})
		return
	}
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名不能为空"})
		return
	}

	updates := map[string]interface{}{
		"host":       req.Host,
		"port":       req.Port,
		"username":   req.Username,
		"status":     0,
		"started_at": nil,
		"ended_at":   nil,
		"last_error": "",
	}

	if req.Password != "" || req.KeyAuth != "" {
		updates["password"] = strings.TrimSpace(req.Password)
		updates["private_key"] = strings.TrimSpace(req.KeyAuth)
	}

	if err := h.db.Model(&session).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if err := h.db.First(&session, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "会话刷新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// PrecheckConnection 在创建会话前做一次连接预检查
func (h *TerminalHandler) PrecheckConnection(c *gin.Context) {
	var req terminalConnectPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := normalizeConnectPayload(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	session := TerminalSession{
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		Password:   req.Password,
		PrivateKey: req.KeyAuth,
	}
	sshSess, err := h.connectSSH(&session)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": normalizeSSHFailureReason(err),
			"data": gin.H{
				"ok":      false,
				"host":    req.Host,
				"port":    req.Port,
				"message": normalizeSSHFailureReason(err),
			},
		})
		return
	}
	h.closeSSHSession(sshSess)
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "连接测试通过",
		"data": gin.H{
			"ok":      true,
			"host":    req.Host,
			"port":    req.Port,
			"message": "SSH 连通且认证成功",
		},
	})
}

// CloseSession 关闭会话
func (h *TerminalHandler) CloseSession(c *gin.Context) {
	id := c.Param("id")

	var exists int64
	if err := h.db.Model(&TerminalSession{}).Where("id = ?", id).Count(&exists).Error; err != nil || exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if sess, ok := h.sessions.Load(id); ok {
		sshSess := sess.(*SSHSession)
		sshSess.ClosedByUser = true
		h.closeSSHSession(sshSess)
		h.sessions.Delete(id)
	}

	h.db.Model(&TerminalSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     2,
		"ended_at":   time.Now(),
		"last_error": "",
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "会话已关闭"})
}

// DeleteSession 删除会话记录（含关联录像）
func (h *TerminalHandler) DeleteSession(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话ID不能为空"})
		return
	}

	if sess, ok := h.sessions.Load(id); ok {
		sshSess := sess.(*SSHSession)
		sshSess.ClosedByUser = true
		h.closeSSHSession(sshSess)
		h.sessions.Delete(id)
	}

	if err := h.db.Unscoped().Where("session_id = ?", id).Delete(&TerminalRecord{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除会话录像失败"})
		return
	}
	res := h.db.Unscoped().Where("id = ?", id).Delete(&TerminalSession{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除会话失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "会话已删除"})
}

// HandleWebSocket WebSocket连接
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	id := c.Param("id")

	// 验证Token
	token := ""
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if token == "" {
		if cookieToken, err := c.Cookie("token"); err == nil {
			token = strings.TrimSpace(cookieToken)
		}
	}
	// 仅在 WebSocket 升级场景兼容 query token。
	if token == "" && strings.EqualFold(strings.TrimSpace(c.GetHeader("Upgrade")), "websocket") {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
		return
	}

	claims, err := h.auth.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Token无效"})
		return
	}

	// 获取会话信息
	var session TerminalSession
	if err := h.db.First(&session, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	// 升级WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 建立SSH连接
	sshSess, err := h.connectSSH(&session)
	if err != nil {
		failReason := normalizeSSHFailureReason(err)
		h.db.Model(&session).Updates(map[string]interface{}{
			"status":     3,
			"ended_at":   time.Now(),
			"last_error": failReason,
		})
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n连接失败: %s\r\n", failReason)))
		conn.Close()
		return
	}

	sshSess.Conn = conn
	sshSess.StartTime = time.Now()
	h.sessions.Store(id, sshSess)

	// 更新会话状态
	h.db.Model(&session).Updates(map[string]interface{}{
		"status":     1,
		"started_at": time.Now(),
		"last_error": "",
	})

	// 处理WebSocket消息
	go h.handleWSMessages(sshSess, &session, claims.Username)
}

func (h *TerminalHandler) connectSSH(session *TerminalSession) (*SSHSession, error) {
	var authMethods []ssh.AuthMethod

	// 密码认证
	if session.Password != "" {
		authMethods = append(authMethods, ssh.Password(session.Password))
	}

	// 密钥认证
	if session.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(session.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	config := &ssh.ClientConfig{
		User:            session.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", session.Host, session.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := sshSession.RequestPty("xterm-256color", 40, 120, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	stderr, err := sshSession.StderrPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	return &SSHSession{
		ID:         session.ID,
		Client:     client,
		Session:    sshSession,
		StdinPipe:  stdin,
		StdoutPipe: stdout,
		StderrPipe: stderr,
		Recording:  make([]RecordItem, 0),
	}, nil
}

func (h *TerminalHandler) handleWSMessages(sshSess *SSHSession, session *TerminalSession, operator string) {
	closeReason := ""
	reasonCh := make(chan string, 1)
	notifyReason := func(reason string) {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			return
		}
		select {
		case reasonCh <- reason:
		default:
		}
	}

	defer func() {
		sshSess.PendingCommandInput = ""
		h.closeSSHSession(sshSess)
		h.sessions.Delete(sshSess.ID)
		endedAt := time.Now()
		select {
		case reason := <-reasonCh:
			if closeReason == "" {
				closeReason = reason
			}
		default:
		}
		if closeReason == "" && !sshSess.ClosedByUser {
			closeReason = h.diagnoseSessionClose(sshSess)
		}
		updates := map[string]interface{}{
			"status":   2,
			"ended_at": endedAt,
		}
		if closeReason != "" && !sshSess.ClosedByUser {
			updates["status"] = 3
			updates["last_error"] = closeReason
		} else {
			updates["last_error"] = ""
		}
		h.db.Model(session).Updates(updates)

		// 保存录像
		if len(sshSess.Recording) > 0 {
			recordData, _ := json.Marshal(sshSess.Recording)
			record := TerminalRecord{
				SessionID: session.ID,
				Host:      session.Host,
				Operator:  operator,
				Duration:  int(endedAt.Sub(sshSess.StartTime).Seconds()),
				Data:      string(recordData),
			}
			h.db.Create(&record)
		}
	}()

	// 读取SSH输出（stdout/stderr）
	go h.streamSSHOutput(sshSess, sshSess.StdoutPipe, notifyReason)
	go h.streamSSHOutput(sshSess, sshSess.StderrPipe, notifyReason)

	// 等待远端shell结束
	go func() {
		if err := sshSess.Session.Wait(); err != nil {
			reason := normalizeSSHFailureReason(err)
			notifyReason(reason)
			_ = h.writeWSMessage(sshSess, websocket.TextMessage, []byte("\r\n[系统] 连接关闭: "+reason+"\r\n"))
		} else {
			_ = h.writeWSMessage(sshSess, websocket.TextMessage, []byte("\r\n[系统] 连接已关闭\r\n"))
		}
		_ = sshSess.Conn.Close()
	}()

	// 读取WebSocket输入
	for {
		_, message, err := sshSess.Conn.ReadMessage()
		if err != nil {
			closeReason = parseWSDisconnectReason(err)
			select {
			case reason := <-reasonCh:
				if reason != "" {
					closeReason = reason
				}
			default:
			}
			return
		}

		// 处理resize消息
		var msg struct {
			Type string `json:"type"`
			Cols int    `json:"cols"`
			Rows int    `json:"rows"`
		}
		if json.Unmarshal(message, &msg) == nil && msg.Type == "resize" {
			sshSess.Session.WindowChange(msg.Rows, msg.Cols)
			continue
		}
		if json.Unmarshal(message, &msg) == nil && msg.Type == "ping" {
			_ = h.writeWSMessage(sshSess, websocket.TextMessage, []byte(`{"type":"pong"}`))
			continue
		}

		blocked, reason := h.appendJumpCommandInput(sshSess, session, operator, message)
		if blocked {
			sshSess.JumpBlocked = true
			sshSess.JumpBlockReason = reason
			closeReason = "风控阻断: " + reason
			notifyReason(closeReason)
			_ = h.writeWSMessage(sshSess, websocket.TextMessage, []byte("\r\n[风控阻断] "+reason+"\r\n"))
			return
		}

		// 写入SSH
		if _, err := sshSess.StdinPipe.Write(message); err != nil {
			closeReason = normalizeSSHFailureReason(err)
			notifyReason(closeReason)
			_ = h.writeWSMessage(sshSess, websocket.TextMessage, []byte("\r\n[系统] "+closeReason+"\r\n"))
			return
		}

		// 记录输入
		sshSess.Recording = append(sshSess.Recording, RecordItem{
			Time:    time.Since(sshSess.StartTime).Milliseconds(),
			Type:    "input",
			Content: string(message),
		})
	}
}

func (h *TerminalHandler) streamSSHOutput(sshSess *SSHSession, stream io.Reader, notifyReason func(string)) {
	if stream == nil {
		return
	}
	buf := make([]byte, 8192)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			sshSess.appendOutputTail(string(data))
			_ = h.writeWSMessage(sshSess, websocket.BinaryMessage, data)
			sshSess.Recording = append(sshSess.Recording, RecordItem{
				Time:    time.Since(sshSess.StartTime).Milliseconds(),
				Type:    "output",
				Content: string(data),
			})
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				notifyReason(normalizeSSHFailureReason(err))
			}
			return
		}
	}
}

func (h *TerminalHandler) writeWSMessage(sshSess *SSHSession, msgType int, payload []byte) error {
	if sshSess == nil || sshSess.Conn == nil {
		return errors.New("websocket 未连接")
	}
	sshSess.ConnMu.Lock()
	defer sshSess.ConnMu.Unlock()
	return sshSess.Conn.WriteMessage(msgType, payload)
}

func parseWSDisconnectReason(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(msg, "use of closed network connection") || strings.Contains(msg, "broken pipe") {
		return ""
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		switch closeErr.Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway:
			return ""
		case websocket.CloseAbnormalClosure:
			return "连接中断：网络异常或服务端意外断开"
		case websocket.ClosePolicyViolation:
			return "连接被策略拒绝"
		case websocket.CloseUnsupportedData:
			return "连接断开：终端消息格式不支持"
		default:
			if closeErr.Text != "" {
				return closeErr.Text
			}
			return fmt.Sprintf("连接关闭(code=%d)", closeErr.Code)
		}
	}
	return ""
}

func normalizeSSHFailureReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, io.EOF) {
		return "远端会话已结束"
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "unable to authenticate"), strings.Contains(lower, "permission denied"):
		return "认证失败：用户名或密码/密钥错误"
	case strings.Contains(lower, "this account is currently not available"):
		return "远端账号不可登录：登录 shell 无效"
	case strings.Contains(lower, "handshake failed"):
		return "SSH握手失败：请检查账号、密码/密钥与服务端认证策略"
	case strings.Contains(lower, "connection refused"):
		return "连接失败：目标主机端口拒绝连接"
	case strings.Contains(lower, "no route to host"):
		return "连接失败：目标主机网络不可达"
	case strings.Contains(lower, "connection reset by peer"), strings.Contains(lower, "connection closed by remote host"):
		return "连接被远端主机重置"
	case strings.Contains(lower, "i/o timeout"), strings.Contains(lower, "timeout"):
		return "连接超时：请检查网络与防火墙策略"
	case strings.Contains(lower, "exited without exit status"), strings.Contains(lower, "process exited with status"):
		return "远端会话已结束"
	default:
		return msg
	}
}

func (sshSess *SSHSession) appendOutputTail(text string) {
	trimmed := strings.TrimSpace(stripANSI(text))
	if trimmed == "" {
		return
	}
	sshSess.OutputTailMu.Lock()
	defer sshSess.OutputTailMu.Unlock()
	sshSess.OutputTail += " " + trimmed
	if len(sshSess.OutputTail) > 240 {
		sshSess.OutputTail = sshSess.OutputTail[len(sshSess.OutputTail)-240:]
	}
}

func (sshSess *SSHSession) readOutputTail() string {
	sshSess.OutputTailMu.Lock()
	defer sshSess.OutputTailMu.Unlock()
	return strings.TrimSpace(sshSess.OutputTail)
}

func (h *TerminalHandler) diagnoseSessionClose(sshSess *SSHSession) string {
	if sshSess == nil || sshSess.ClosedByUser {
		return ""
	}
	lifetime := time.Since(sshSess.StartTime)
	outputTail := sshSess.readOutputTail()
	lowerTail := strings.ToLower(outputTail)

	switch {
	case strings.Contains(lowerTail, "this account is currently not available"):
		return "远端账号不可登录：登录 shell 无效"
	case strings.Contains(lowerTail, "permission denied"):
		return "认证成功后会话被远端拒绝：请检查账号权限或登录策略"
	case strings.Contains(lowerTail, "last login"):
		return ""
	case lifetime < 3*time.Second && outputTail != "":
		return "远端会话启动后立即退出：" + outputTail
	case lifetime < 3*time.Second:
		return "远端会话启动后立即退出：请检查登录 shell、强制命令或 PAM/安全策略"
	default:
		return ""
	}
}

func (h *TerminalHandler) closeSSHSession(sshSess *SSHSession) {
	if sshSess.StdinPipe != nil {
		sshSess.StdinPipe.Close()
	}
	if sshSess.Session != nil {
		sshSess.Session.Close()
	}
	if sshSess.Client != nil {
		sshSess.Client.Close()
	}
	if sshSess.Conn != nil {
		sshSess.Conn.Close()
	}
}

func (h *TerminalHandler) appendJumpCommandInput(sshSess *SSHSession, session *TerminalSession, operator string, raw []byte) (bool, string) {
	if sshSess == nil || len(raw) == 0 {
		return false, ""
	}
	if !h.ensureJumpSession(sshSess) {
		return false, ""
	}

	clean := stripANSI(string(raw))
	if clean == "" {
		return false, ""
	}

	for _, r := range clean {
		switch r {
		case '\r', '\n':
			cmd := strings.TrimSpace(sshSess.PendingCommandInput)
			sshSess.PendingCommandInput = ""
			if cmd == "" {
				continue
			}
			decision := h.auditJumpCommand(sshSess, session, operator, cmd)
			if decision.Matched && decision.Action == "block" {
				return true, decision.Reason
			}
		case '\b', 127:
			sshSess.PendingCommandInput = trimLastRune(sshSess.PendingCommandInput)
		default:
			if r < 32 && r != '\t' {
				continue
			}
			sshSess.PendingCommandInput += string(r)
		}
	}
	return false, ""
}

func (h *TerminalHandler) auditJumpCommand(sshSess *SSHSession, session *TerminalSession, operator, cmd string) jumpRiskDecision {
	now := time.Now()
	decision := h.matchJumpCommandRule(sshSess, cmd)
	if decision.Matched && !decision.WhitelistHit && (decision.Action == "alert" || decision.Action == "block") {
		decision.AlertID = h.createJumpRiskAlert(sshSess, session, decision, cmd, now)
	}
	matchedRules, _ := json.Marshal(decision.MatchedRules)

	record := jumpCommandAudit{
		ID:         uuid.NewString(),
		CreatedAt:  now,
		UpdatedAt:  now,
		SessionID:  sshSess.JumpSessionID,
		Username:   operator,
		Command:    cmd,
		ExecutedAt: now,
	}
	if decision.Matched {
		record.RuleID = strings.Join(decision.RuleIDs, ",")
		record.RuleName = strings.Join(decision.RuleNames, ",")
		record.MatchedRules = string(matchedRules)
		record.WhitelistHit = decision.WhitelistHit
		record.RiskLevel = decision.Severity
		record.RiskAction = decision.Action
		record.RiskReason = decision.Reason
		record.Blocked = !decision.WhitelistHit && decision.Action == "block"
		record.AlertID = decision.AlertID
	}

	if err := h.db.Table("jump_commands").Create(&record).Error; err != nil {
		return jumpRiskDecision{}
	}
	_ = h.db.Table("jump_sessions").Where("id = ? AND status = ?", sshSess.JumpSessionID, "active").Updates(map[string]interface{}{
		"last_command_at": now,
		"command_count":   gorm.Expr("COALESCE(command_count, 0) + 1"),
	}).Error
	if decision.Matched && !decision.WhitelistHit && decision.Action == "block" {
		h.markJumpSessionBlocked(sshSess, decision.Reason, now)
	}
	return decision
}

func (h *TerminalHandler) matchJumpCommandRule(sshSess *SSHSession, cmd string) jumpRiskDecision {
	if sshSess == nil || strings.TrimSpace(cmd) == "" {
		return jumpRiskDecision{}
	}

	var rules []jumpCommandRule
	query := h.db.Table("jump_command_rules").Where("enabled = ?", true).Order("priority DESC, created_at ASC")
	if protocol := strings.TrimSpace(sshSess.JumpProtocol); protocol != "" {
		query = query.Where("(protocol = '' OR protocol = ?)", strings.ToLower(protocol))
	}
	if err := query.Find(&rules).Error; err != nil {
		return jumpRiskDecision{}
	}
	if len(rules) == 0 {
		return jumpRiskDecision{}
	}

	matches := make([]jumpRuleHit, 0)
	allowMatches := make([]jumpRuleHit, 0)
	riskMatches := make([]jumpRuleHit, 0)
	for i := range rules {
		rule := rules[i]
		pattern := strings.TrimSpace(rule.Pattern)
		if pattern == "" {
			continue
		}
		matched := false
		switch strings.ToLower(strings.TrimSpace(rule.MatchType)) {
		case "exact":
			matched = cmd == pattern
		case "prefix":
			matched = strings.HasPrefix(cmd, pattern)
		case "regex":
			re, err := regexp.Compile(pattern)
			if err == nil {
				matched = re.MatchString(cmd)
			}
		default:
			matched = strings.Contains(strings.ToLower(cmd), strings.ToLower(pattern))
		}
		if !matched {
			continue
		}
		hit := jumpRuleHit{
			ID:       rule.ID,
			Name:     rule.Name,
			RuleKind: normalizeJumpRuleKind(rule.RuleKind),
			Action:   normalizeJumpRuleAction(rule.Action),
			Severity: normalizeJumpRuleSeverity(rule.Severity),
			Priority: rule.Priority,
			Pattern:  pattern,
		}
		matches = append(matches, hit)
		if hit.RuleKind == "allow" {
			allowMatches = append(allowMatches, hit)
			continue
		}
		riskMatches = append(riskMatches, hit)
	}
	if len(matches) == 0 {
		return jumpRiskDecision{}
	}
	if len(allowMatches) > 0 {
		ruleIDs := make([]string, 0, len(allowMatches))
		ruleNames := make([]string, 0, len(allowMatches))
		for i := range allowMatches {
			ruleIDs = append(ruleIDs, allowMatches[i].ID)
			ruleNames = append(ruleNames, allowMatches[i].Name)
		}
		return jumpRiskDecision{
			Matched:      true,
			WhitelistHit: true,
			RuleID:       strings.Join(ruleIDs, ","),
			RuleName:     strings.Join(ruleNames, ","),
			RuleIDs:      ruleIDs,
			RuleNames:    ruleNames,
			MatchedRules: matches,
			Severity:     "info",
			Action:       "allow",
			Reason:       "命中白名单规则，已放行",
		}
	}
	if len(riskMatches) == 0 {
		return jumpRiskDecision{}
	}
	ruleIDs := make([]string, 0, len(riskMatches))
	ruleNames := make([]string, 0, len(riskMatches))
	action := "alert"
	severity := "info"
	for i := range riskMatches {
		ruleIDs = append(ruleIDs, riskMatches[i].ID)
		ruleNames = append(ruleNames, riskMatches[i].Name)
		if riskMatches[i].Action == "block" {
			action = "block"
		}
		if compareJumpSeverity(riskMatches[i].Severity, severity) > 0 {
			severity = riskMatches[i].Severity
		}
	}
	return jumpRiskDecision{
		Matched:      true,
		RuleID:       strings.Join(ruleIDs, ","),
		RuleName:     strings.Join(ruleNames, ","),
		RuleIDs:      ruleIDs,
		RuleNames:    ruleNames,
		MatchedRules: matches,
		Severity:     severity,
		Action:       action,
		Reason:       fmt.Sprintf("命中规则[%s]，命令已%s", strings.Join(ruleNames, ","), map[bool]string{true: "阻断", false: "记录告警"}[action == "block"]),
	}
}

func (h *TerminalHandler) markJumpSessionBlocked(sshSess *SSHSession, reason string, now time.Time) {
	if sshSess == nil || strings.TrimSpace(sshSess.JumpSessionID) == "" {
		return
	}
	var info jumpSessionLink
	_ = h.db.Table("jump_sessions").Select("id, started_at").Where("id = ?", sshSess.JumpSessionID).Limit(1).Scan(&info).Error
	duration := 0
	if !info.StartedAt.IsZero() {
		duration = int(now.Sub(info.StartedAt).Seconds())
	}
	_ = h.db.Table("jump_sessions").Where("id = ? AND status = ?", sshSess.JumpSessionID, "active").Updates(map[string]interface{}{
		"status":       "blocked",
		"ended_at":     now,
		"duration_sec": duration,
		"close_reason": reason,
	}).Error
}

func (h *TerminalHandler) createJumpRiskAlert(sshSess *SSHSession, session *TerminalSession, decision jumpRiskDecision, cmd string, now time.Time) string {
	if sshSess == nil || session == nil || !decision.Matched || decision.WhitelistHit || decision.Action == "allow" {
		return ""
	}
	primaryRuleID := decision.RuleID
	if len(decision.RuleIDs) > 0 && strings.TrimSpace(decision.RuleIDs[0]) != "" {
		primaryRuleID = strings.TrimSpace(decision.RuleIDs[0])
	}
	combinedRuleName := strings.Join(decision.RuleNames, ",")
	if combinedRuleName == "" {
		combinedRuleName = decision.RuleName
	}

	labels, _ := json.Marshal(map[string]string{
		"source":       "jump",
		"jump_session": sshSess.JumpSessionID,
		"protocol":     sshSess.JumpProtocol,
		"host":         session.Host,
		"operator":     session.Operator,
		"rule_id":      primaryRuleID,
		"action":       decision.Action,
	})
	annotations, _ := json.Marshal(map[string]string{
		"command": cmd,
		"reason":  decision.Reason,
	})

	hash := sha1.Sum([]byte(sshSess.JumpSessionID + "|" + strings.Join(decision.RuleIDs, ",") + "|" + cmd + "|" + now.Format(time.RFC3339Nano)))
	fingerprint := hex.EncodeToString(hash[:])
	alert := generatedAlert{
		ID:          uuid.NewString(),
		CreatedAt:   now,
		UpdatedAt:   now,
		RuleID:      primaryRuleID,
		RuleName:    "Jump命令风控/" + truncateText(combinedRuleName, 96),
		Fingerprint: fingerprint,
		Target:      session.Host,
		Metric:      "jump_command_risk",
		Value:       truncateText(cmd, 64),
		Threshold:   truncateText(combinedRuleName, 64),
		Severity:    decision.Severity,
		Status:      0,
		FiredAt:     now,
		GroupKey:    "jump-command-risk-" + primaryRuleID,
		Labels:      string(labels),
		Annotations: string(annotations),
	}
	if err := h.db.Table("alerts").Create(&alert).Error; err != nil {
		return ""
	}
	return alert.ID
}

func (h *TerminalHandler) ensureJumpSession(sshSess *SSHSession) bool {
	if sshSess == nil {
		return false
	}
	if sshSess.JumpSessionID != "" {
		return true
	}
	if sshSess.JumpAuditResolved {
		return false
	}

	var linked jumpSessionLink
	err := h.db.Table("jump_sessions").
		Select("id, protocol, asset_name").
		Where("relay_session_id = ? AND status = ?", sshSess.ID, "active").
		Limit(1).
		Scan(&linked).Error
	sshSess.JumpAuditResolved = true
	if err != nil || strings.TrimSpace(linked.ID) == "" {
		return false
	}
	sshSess.JumpSessionID = linked.ID
	sshSess.JumpProtocol = strings.TrimSpace(linked.Protocol)
	sshSess.JumpAssetName = strings.TrimSpace(linked.AssetName)
	return true
}

func stripANSI(input string) string {
	if input == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(input))
	inEsc := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inEsc {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '~' {
				inEsc = false
			}
			continue
		}
		if ch == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func trimLastRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[:len(r)-1])
}

func truncateText(in string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(in)
	if len(r) <= max {
		return in
	}
	return string(r[:max])
}

func normalizeJumpRuleKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "allow":
		return "allow"
	default:
		return "risk"
	}
}

func normalizeJumpRuleAction(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "block":
		return "block"
	default:
		return "alert"
	}
}

func normalizeJumpRuleSeverity(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "critical":
		return "critical"
	case "info":
		return "info"
	default:
		return "warning"
	}
}

func compareJumpSeverity(a, b string) int {
	rank := map[string]int{
		"info":     1,
		"warning":  2,
		"critical": 3,
	}
	return rank[normalizeJumpRuleSeverity(a)] - rank[normalizeJumpRuleSeverity(b)]
}

// ListRecords 录像列表
func (h *TerminalHandler) ListRecords(c *gin.Context) {
	var records []TerminalRecord
	query := h.db.Order("created_at DESC")
	if host := strings.TrimSpace(c.Query("host")); host != "" {
		query = query.Where("host LIKE ?", "%"+host+"%")
	}
	if operator := strings.TrimSpace(c.Query("operator")); operator != "" {
		query = query.Where("operator LIKE ?", "%"+operator+"%")
	}
	if sessionID := strings.TrimSpace(c.Query("session_id")); sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("host LIKE ? OR operator LIKE ? OR session_id LIKE ?", like, like, like)
	}
	if err := query.Limit(100).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 不返回data字段
	for i := range records {
		records[i].Data = ""
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// ListCommandAudits 命令审计列表
func (h *TerminalHandler) ListCommandAudits(c *gin.Context) {
	var audits []terminalCommandAuditItem
	query := h.db.Table("jump_commands jc").
		Select(`
			jc.id,
			jc.session_id,
			COALESCE(js.session_no, '') AS session_no,
			COALESCE(ts.host, js.asset_name, '') AS host,
			COALESCE(js.asset_name, '') AS asset_name,
			COALESCE(jc.username, '') AS operator,
			COALESCE(js.account_name, '') AS login_user,
			COALESCE(js.protocol, '') AS protocol,
			jc.command,
			COALESCE(jc.rule_name, '') AS rule_name,
			COALESCE(jc.risk_level, '') AS risk_level,
			COALESCE(jc.risk_action, '') AS risk_action,
			COALESCE(jc.risk_reason, '') AS risk_reason,
			jc.blocked,
			jc.executed_at
		`).
		Joins("LEFT JOIN jump_sessions js ON js.id = jc.session_id").
		Joins("LEFT JOIN terminal_sessions ts ON ts.id = js.relay_session_id").
		Order("jc.executed_at DESC")

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`
			jc.command LIKE ?
			OR COALESCE(ts.host, js.asset_name, '') LIKE ?
			OR COALESCE(jc.username, '') LIKE ?
			OR COALESCE(js.account_name, '') LIKE ?
			OR COALESCE(js.session_no, '') LIKE ?
			OR COALESCE(jc.risk_reason, '') LIKE ?
		`, like, like, like, like, like, like)
	}
	if riskLevel := strings.TrimSpace(c.Query("risk_level")); riskLevel != "" {
		query = query.Where("jc.risk_level = ?", riskLevel)
	}
	if protocol := strings.TrimSpace(c.Query("protocol")); protocol != "" {
		query = query.Where("js.protocol = ?", protocol)
	}
	if blocked := strings.TrimSpace(c.Query("blocked")); blocked != "" {
		switch strings.ToLower(blocked) {
		case "1", "true", "yes":
			query = query.Where("jc.blocked = ?", true)
		case "0", "false", "no":
			query = query.Where("jc.blocked = ?", false)
		}
	}

	if err := query.Limit(200).Scan(&audits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": audits})
}

// GetRecord 获取录像详情
func (h *TerminalHandler) GetRecord(c *gin.Context) {
	id := c.Param("id")
	var record TerminalRecord
	if err := h.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "录像不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

// DownloadRecord 下载录像
func (h *TerminalHandler) DownloadRecord(c *gin.Context) {
	id := c.Param("id")
	var record TerminalRecord
	if err := h.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "录像不存在"})
		return
	}

	payload := gin.H{
		"id":         record.ID,
		"session_id": record.SessionID,
		"host":       record.Host,
		"operator":   record.Operator,
		"duration":   record.Duration,
		"created_at": record.CreatedAt,
		"updated_at": record.UpdatedAt,
		"events":     json.RawMessage(record.Data),
	}

	filename := fmt.Sprintf("terminal-record-%s-%s.json", sanitizeRecordFilePart(record.Host), record.ID)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.JSON(http.StatusOK, payload)
}

// DownloadRecordAsciinema 下载 asciinema cast
func (h *TerminalHandler) DownloadRecordAsciinema(c *gin.Context) {
	id := c.Param("id")
	var record TerminalRecord
	if err := h.db.First(&record, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "录像不存在"})
		return
	}

	castData, err := buildAsciinemaCast(record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成 cast 失败"})
		return
	}

	filename := fmt.Sprintf("terminal-record-%s-%s.cast", sanitizeRecordFilePart(record.Host), record.ID)
	c.Header("Content-Type", "application/x-asciicast; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(castData)
}

// ExportRecords 批量导出录像
func (h *TerminalHandler) ExportRecords(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Format string   `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择至少一条录像"})
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "cast" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "导出格式不支持"})
		return
	}

	var records []TerminalRecord
	if err := h.db.Where("id IN ?", req.IDs).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取录像失败"})
		return
	}
	if len(records) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "未找到可导出的录像"})
		return
	}

	zipData, err := buildRecordZip(records, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成导出包失败"})
		return
	}

	filename := fmt.Sprintf("terminal-records-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(zipData)
}

// DeleteRecord 删除单条录像
func (h *TerminalHandler) DeleteRecord(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "录像ID不能为空"})
		return
	}
	res := h.db.Unscoped().Where("id = ?", id).Delete(&TerminalRecord{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除录像失败"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "录像不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "录像已删除"})
}

// CleanupRecords 清理历史录像
func (h *TerminalHandler) CleanupRecords(c *gin.Context) {
	var req struct {
		KeepDays int `json:"keep_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.KeepDays <= 0 {
		req.KeepDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -req.KeepDays)
	res := h.db.Unscoped().Where("created_at < ?", cutoff).Delete(&TerminalRecord{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "清理录像失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "历史录像清理完成",
		"data": gin.H{
			"deleted":   res.RowsAffected,
			"keep_days": req.KeepDays,
			"before_at": cutoff,
		},
	})
}

func sanitizeRecordFilePart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(":", "-", "/", "-", "\\", "-", " ", "-")
	v = replacer.Replace(v)
	v = regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(v, "")
	if v == "" {
		return "unknown"
	}
	return v
}

func buildAsciinemaCast(record TerminalRecord) ([]byte, error) {
	var items []RecordItem
	raw := strings.TrimSpace(record.Data)
	if raw == "" {
		raw = "[]"
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}

	header := map[string]interface{}{
		"version":   2,
		"width":     120,
		"height":    40,
		"timestamp": record.CreatedAt.Unix(),
		"title":     fmt.Sprintf("%s@%s", record.Operator, record.Host),
		"env": map[string]string{
			"SHELL": "/bin/bash",
			"TERM":  "xterm-256color",
		},
	}

	var buf bytes.Buffer
	head, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	buf.Write(head)
	buf.WriteByte('\n')

	for _, item := range items {
		eventType := "o"
		if item.Type == "input" {
			eventType = "i"
		}
		entry, err := json.Marshal([]interface{}{
			float64(item.Time) / 1000.0,
			eventType,
			item.Content,
		})
		if err != nil {
			return nil, err
		}
		buf.Write(entry)
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func buildRecordZip(records []TerminalRecord, format string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, record := range records {
		base := fmt.Sprintf("terminal-record-%s-%s", sanitizeRecordFilePart(record.Host), record.ID)
		var (
			name    string
			content []byte
			err     error
		)
		if format == "cast" {
			name = base + ".cast"
			content, err = buildAsciinemaCast(record)
		} else {
			name = base + ".json"
			payload := map[string]interface{}{
				"id":         record.ID,
				"session_id": record.SessionID,
				"host":       record.Host,
				"operator":   record.Operator,
				"duration":   record.Duration,
				"created_at": record.CreatedAt,
				"updated_at": record.UpdatedAt,
				"events":     json.RawMessage(record.Data),
			}
			content, err = json.MarshalIndent(payload, "", "  ")
		}
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		fw, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err = fw.Write(content); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
