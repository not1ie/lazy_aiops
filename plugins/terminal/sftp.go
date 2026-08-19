package terminal

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SFTPFileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"size_human"`
	Mode      string `json:"mode"`
	ModTime   string `json:"mod_time"`
	IsDir     bool   `json:"is_dir"`
	IsSymlink bool   `json:"is_symlink"`
	Ext       string `json:"ext"`
}

type SFTPListResult struct {
	Path   string         `json:"path"`
	Parent string         `json:"parent"`
	Files  []SFTPFileInfo `json:"files"`
}

// formatBytes 人类可读文件大小格式化
func formatBytes(bytes int64) string {
	if bytes < 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getSFTPClient 根据 Session ID 获取或新建 SFTP 客户端连接
func (h *TerminalHandler) getSFTPClient(sessionID string) (*sftp.Client, func(), error) {
	// 1. 如果已存在活跃的 Web SSH 交互连接，复用其 SSH Client 获得极致速度
	if sessVal, ok := h.sessions.Load(sessionID); ok {
		sshSess := sessVal.(*SSHSession)
		if sshSess != nil && sshSess.Client != nil {
			sftpClient, err := sftp.NewClient(sshSess.Client)
			if err == nil {
				return sftpClient, func() { _ = sftpClient.Close() }, nil
			}
		}
	}

	// 2. 否则从数据库读取会话连接凭据建立专用 SFTP 客户端
	var session TerminalSession
	if err := h.db.First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, nil, fmt.Errorf("终端会话不存在: %w", err)
	}

	// 构建 SSH 认证
	authMethods := []ssh.AuthMethod{}
	if strings.TrimSpace(session.Password) != "" {
		authMethods = append(authMethods, ssh.Password(session.Password))
	}
	if strings.TrimSpace(session.PrivateKey) != "" {
		signer, err := ssh.ParsePrivateKey([]byte(session.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if len(authMethods) == 0 {
		return nil, nil, fmt.Errorf("未找到有效密码或私钥认证信息")
	}

	port := session.Port
	if port <= 0 {
		port = 22
	}

	sshConfig := &ssh.ClientConfig{
		User:            session.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         6 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", session.Host, port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("SFTP 子系统初始化失败: %w", err)
	}

	cleanup := func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}

	return sftpClient, cleanup, nil
}

// SFTPList 浏览目录列表
func (h *TerminalHandler) SFTPList(c *gin.Context) {
	sessionID := c.Param("id")
	targetPath := strings.TrimSpace(c.Query("path"))
	if targetPath == "" {
		targetPath = "/"
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	// 规范化绝对路径
	if targetPath == "~" {
		targetPath, _ = sftpClient.Getwd()
		if targetPath == "" {
			targetPath = "/root"
		}
	}
	targetPath = path.Clean(targetPath)

	entries, err := sftpClient.ReadDir(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取目录失败: " + err.Error()})
		return
	}

	var fileList []SFTPFileInfo
	for _, entry := range entries {
		name := entry.Name()
		fullPath := path.Join(targetPath, name)
		isDir := entry.IsDir()
		isSymlink := entry.Mode()&os.ModeSymlink != 0
		ext := strings.TrimPrefix(filepath.Ext(name), ".")

		fileList = append(fileList, SFTPFileInfo{
			Name:      name,
			Path:      fullPath,
			Size:      entry.Size(),
			SizeHuman: formatBytes(entry.Size()),
			Mode:      entry.Mode().String(),
			ModTime:   entry.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:     isDir,
			IsSymlink: isSymlink,
			Ext:       strings.ToLower(ext),
		})
	}

	// 排序：目录排在最前，然后按名称字母升序
	sort.Slice(fileList, func(i, j int) bool {
		if fileList[i].IsDir && !fileList[j].IsDir {
			return true
		}
		if !fileList[i].IsDir && fileList[j].IsDir {
			return false
		}
		return strings.ToLower(fileList[i].Name) < strings.ToLower(fileList[j].Name)
	})

	parentPath := path.Dir(targetPath)
	if targetPath == "/" {
		parentPath = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": SFTPListResult{
			Path:   targetPath,
			Parent: parentPath,
			Files:  fileList,
		},
	})
}

// SFTPReadFile 读取文件内容（限制 5MB 内）
func (h *TerminalHandler) SFTPReadFile(c *gin.Context) {
	sessionID := c.Param("id")
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件路径不能为空"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	stat, err := sftpClient.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在或无权访问: " + err.Error()})
		return
	}
	if stat.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标是一个目录，无法直接作为文本读取"})
		return
	}
	if stat.Size() > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("文件过大 (%s)，在线编辑仅支持 5MB 以内文本文件，请直接下载", formatBytes(stat.Size()))})
		return
	}

	file, err := sftpClient.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "打开远程文件失败: " + err.Error()})
		return
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件流失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"path":       filePath,
			"name":       path.Base(filePath),
			"size":       stat.Size(),
			"size_human": formatBytes(stat.Size()),
			"content":    string(contentBytes),
		},
	})
}

// SFTPWriteFile 写入/保存文件内容
func (h *TerminalHandler) SFTPWriteFile(c *gin.Context) {
	sessionID := c.Param("id")
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件路径和内容不能为空"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	filePath := path.Clean(strings.TrimSpace(req.Path))

	// 创建或覆盖文件
	file, err := sftpClient.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "打开/创建远程文件失败: " + err.Error()})
		return
	}
	defer file.Close()

	if _, err := file.Write([]byte(req.Content)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "写入文件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "文件保存成功",
		"data": gin.H{
			"path": filePath,
			"size": len(req.Content),
		},
	})
}

// SFTPUploadFile 上传文件到远程指定目录
func (h *TerminalHandler) SFTPUploadFile(c *gin.Context) {
	sessionID := c.Param("id")
	targetDir := strings.TrimSpace(c.PostForm("target_dir"))
	if targetDir == "" {
		targetDir = "/root"
	}
	targetDir = path.Clean(targetDir)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件上传表单: " + err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择要上传的文件"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	// 确保目标目录存在
	_ = sftpClient.MkdirAll(targetDir)

	uploadedCount := 0
	for _, fileHeader := range files {
		if err := uploadSingleFile(sftpClient, targetDir, fileHeader); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": fmt.Sprintf("上传文件 %s 失败: %v", fileHeader.Filename, err),
			})
			return
		}
		uploadedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("成功上传 %d 个文件至 %s", uploadedCount, targetDir),
	})
}

func uploadSingleFile(client *sftp.Client, targetDir string, fileHeader *multipart.FileHeader) error {
	srcFile, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destPath := path.Join(targetDir, fileHeader.Filename)
	destFile, err := client.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// SFTPDownloadFile 流式下载远程文件
func (h *TerminalHandler) SFTPDownloadFile(c *gin.Context) {
	sessionID := c.Param("id")
	filePath := strings.TrimSpace(c.Query("path"))
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件路径不能为空"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	stat, err := sftpClient.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在: " + err.Error()})
		return
	}
	if stat.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标是目录，暂不支持直接下载目录"})
		return
	}

	file, err := sftpClient.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "无法打开文件: " + err.Error()})
		return
	}
	defer file.Close()

	fileName := path.Base(filePath)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))

	_, _ = io.Copy(c.Writer, file)
}

// SFTPMkdir 创建远程目录
func (h *TerminalHandler) SFTPMkdir(c *gin.Context) {
	sessionID := c.Param("id")
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目录路径不能为空"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	targetPath := path.Clean(strings.TrimSpace(req.Path))
	if err := sftpClient.MkdirAll(targetPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建目录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "目录创建成功",
		"data":    gin.H{"path": targetPath},
	})
}

// SFTPRename 重命名或移动文件/目录
func (h *TerminalHandler) SFTPRename(c *gin.Context) {
	sessionID := c.Param("id")
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.OldPath) == "" || strings.TrimSpace(req.NewPath) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "原路径和新路径均不能为空"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	oldPath := path.Clean(strings.TrimSpace(req.OldPath))
	newPath := path.Clean(strings.TrimSpace(req.NewPath))

	if err := sftpClient.Rename(oldPath, newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "重命名/移动失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "重命名成功",
	})
}

// SFTPDelete 删除文件或目录
func (h *TerminalHandler) SFTPDelete(c *gin.Context) {
	sessionID := c.Param("id")
	targetPath := strings.TrimSpace(c.Query("path"))
	if targetPath == "" {
		var req struct {
			Path string `json:"path"`
		}
		_ = c.ShouldBindJSON(&req)
		targetPath = strings.TrimSpace(req.Path)
	}
	if targetPath == "" || targetPath == "/" || targetPath == "/root" || targetPath == "/etc" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "非法或危险的删除路径"})
		return
	}

	sftpClient, cleanup, err := h.getSFTPClient(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	defer cleanup()

	targetPath = path.Clean(targetPath)
	stat, err := sftpClient.Stat(targetPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "目标不存在: " + err.Error()})
		return
	}

	if stat.IsDir() {
		// 递归删除目录
		if err := removeSFTPDirRecursive(sftpClient, targetPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除目录失败: " + err.Error()})
			return
		}
	} else {
		if err := sftpClient.Remove(targetPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文件失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// removeSFTPDirRecursive 递归删除 SFTP 目录与子项
func removeSFTPDirRecursive(client *sftp.Client, dirPath string) error {
	entries, err := client.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		subPath := path.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if err := removeSFTPDirRecursive(client, subPath); err != nil {
				return err
			}
		} else {
			if err := client.Remove(subPath); err != nil {
				return err
			}
		}
	}
	return client.RemoveDirectory(dirPath)
}
