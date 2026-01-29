package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	ID        string
	Client    *ssh.Client
	Session   *ssh.Session
	StdinPipe io.WriteCloser
	Conn      *websocket.Conn
	Recording []RecordItem
	StartTime time.Time
}

type RecordItem struct {
	Time    int64  `json:"time"` // 相对于开始时间的毫秒数
	Type    string `json:"type"` // input, output
	Content string `json:"content"`
}

func NewTerminalHandler(db *gorm.DB, auth *core.AuthService) *TerminalHandler {
	return &TerminalHandler{db: db, auth: auth}
}

// ListSessions 会话列表
func (h *TerminalHandler) ListSessions(c *gin.Context) {
	var sessions []TerminalSession
	if err := h.db.Where("status = ?", 1).Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessions})
}

// CreateSession 创建会话
func (h *TerminalHandler) CreateSession(c *gin.Context) {
	var req struct {
		HostID   string `json:"host_id" binding:"required"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		KeyAuth  string `json:"key_auth"` // 私钥内容
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 创建会话记录
	session := TerminalSession{
		HostID:   req.HostID,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		UserID:   c.GetString("user_id"),
		Operator: c.GetString("username"),
		Status:   0, // 待连接
	}
	if err := h.db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": session})
}

// CloseSession 关闭会话
func (h *TerminalHandler) CloseSession(c *gin.Context) {
	id := c.Param("id")

	if sess, ok := h.sessions.Load(id); ok {
		sshSess := sess.(*SSHSession)
		h.closeSSHSession(sshSess)
		h.sessions.Delete(id)
	}

	h.db.Model(&TerminalSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   2,
		"ended_at": time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "会话已关闭"})
}

// HandleWebSocket WebSocket连接
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	id := c.Param("id")

	// 验证Token
	token := c.Query("token")
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
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\n连接失败: %s\r\n", err.Error())))
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

	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, err
	}

	return &SSHSession{
		ID:        session.ID,
		Client:    client,
		Session:   sshSession,
		StdinPipe: stdin,
		Recording: make([]RecordItem, 0),
	}, nil
}

func (h *TerminalHandler) handleWSMessages(sshSess *SSHSession, session *TerminalSession, operator string) {
	defer func() {
		h.closeSSHSession(sshSess)
		h.sessions.Delete(sshSess.ID)
		h.db.Model(session).Updates(map[string]interface{}{
			"status":   2,
			"ended_at": time.Now(),
		})

		// 保存录像
		if len(sshSess.Recording) > 0 {
			recordData, _ := json.Marshal(sshSess.Recording)
			record := TerminalRecord{
				SessionID: session.ID,
				Host:      session.Host,
				Operator:  operator,
				Duration:  int(time.Since(sshSess.StartTime).Seconds()),
				Data:      string(recordData),
			}
			h.db.Create(&record)
		}
	}()

	// 读取SSH输出
	stdout, _ := sshSess.Session.StdoutPipe()
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				sshSess.Conn.Close()
				return
			}
			if n > 0 {
				data := buf[:n]
				sshSess.Conn.WriteMessage(websocket.BinaryMessage, data)

				// 记录输出
				sshSess.Recording = append(sshSess.Recording, RecordItem{
					Time:    time.Since(sshSess.StartTime).Milliseconds(),
					Type:    "output",
					Content: string(data),
				})
			}
		}
	}()

	// 读取WebSocket输入
	for {
		_, message, err := sshSess.Conn.ReadMessage()
		if err != nil {
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

		// 写入SSH
		sshSess.StdinPipe.Write(message)

		// 记录输入
		sshSess.Recording = append(sshSess.Recording, RecordItem{
			Time:    time.Since(sshSess.StartTime).Milliseconds(),
			Type:    "input",
			Content: string(message),
		})
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

// ListRecords 录像列表
func (h *TerminalHandler) ListRecords(c *gin.Context) {
	var records []TerminalRecord
	query := h.db.Order("created_at DESC")
	if host := c.Query("host"); host != "" {
		query = query.Where("host LIKE ?", "%"+host+"%")
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
