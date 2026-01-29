package gitops

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GitOpsHandler struct {
	db       *gorm.DB
	workDir  string
}

func NewGitOpsHandler(db *gorm.DB, workDir string) *GitOpsHandler {
	if workDir == "" {
		workDir = "data/gitops"
	}
	os.MkdirAll(workDir, 0755)
	return &GitOpsHandler{db: db, workDir: workDir}
}

// ListRepos 仓库列表
func (h *GitOpsHandler) ListRepos(c *gin.Context) {
	var repos []GitRepo
	if err := h.db.Find(&repos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": repos})
}

// CreateRepo 创建仓库
func (h *GitOpsHandler) CreateRepo(c *gin.Context) {
	var repo GitRepo
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	repo.LocalPath = filepath.Join(h.workDir, repo.Name)
	if err := h.db.Create(&repo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 异步克隆仓库
	go h.cloneRepo(&repo)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": repo})
}

// GetRepo 获取仓库详情
func (h *GitOpsHandler) GetRepo(c *gin.Context) {
	id := c.Param("id")
	var repo GitRepo
	if err := h.db.First(&repo, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": repo})
}

// DeleteRepo 删除仓库
func (h *GitOpsHandler) DeleteRepo(c *gin.Context) {
	id := c.Param("id")
	var repo GitRepo
	if err := h.db.First(&repo, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	// 删除本地目录
	os.RemoveAll(repo.LocalPath)

	if err := h.db.Delete(&repo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// SyncRepo 同步仓库
func (h *GitOpsHandler) SyncRepo(c *gin.Context) {
	id := c.Param("id")
	var repo GitRepo
	if err := h.db.First(&repo, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	// 执行git pull
	if err := h.pullRepo(&repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步成功"})
}

// ListConfigs 配置列表
func (h *GitOpsHandler) ListConfigs(c *gin.Context) {
	var configs []GitConfig
	query := h.db.Preload("Repo")
	if repoID := c.Query("repo_id"); repoID != "" {
		query = query.Where("repo_id = ?", repoID)
	}
	if err := query.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// CreateConfig 创建配置
func (h *GitOpsHandler) CreateConfig(c *gin.Context) {
	var config GitConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}

// GetConfig 获取配置内容
func (h *GitOpsHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")
	var config GitConfig
	if err := h.db.Preload("Repo").First(&config, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	// 读取文件内容
	filePath := filepath.Join(config.Repo.LocalPath, config.FilePath)
	content, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取文件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"config":  config,
		"content": string(content),
	}})
}

// UpdateConfig 更新配置
func (h *GitOpsHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	var config GitConfig
	if err := h.db.Preload("Repo").First(&config, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	var req struct {
		Content       string `json:"content" binding:"required"`
		CommitMessage string `json:"commit_message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 写入文件
	filePath := filepath.Join(config.Repo.LocalPath, config.FilePath)
	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "写入文件失败"})
		return
	}

	// 提交并推送
	username := c.GetString("username")
	commitMsg := req.CommitMessage
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Update %s by %s", config.FilePath, username)
	}

	if err := h.commitAndPush(config.Repo, config.FilePath, commitMsg, username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Git操作失败: " + err.Error()})
		return
	}

	// 记录变更
	change := ConfigChange{
		ConfigID:      config.ID,
		ConfigName:    config.Name,
		ChangeType:    "update",
		Content:       req.Content,
		CommitMessage: commitMsg,
		CommitBy:      username,
	}
	h.db.Create(&change)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// ListChanges 变更历史
func (h *GitOpsHandler) ListChanges(c *gin.Context) {
	var changes []ConfigChange
	query := h.db.Order("created_at DESC")
	if configID := c.Query("config_id"); configID != "" {
		query = query.Where("config_id = ?", configID)
	}
	if err := query.Limit(100).Find(&changes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": changes})
}

// Git操作
func (h *GitOpsHandler) cloneRepo(repo *GitRepo) error {
	h.db.Model(repo).Update("status", 1) // syncing

	args := []string{"clone", "--depth", "1"}
	if repo.Branch != "" {
		args = append(args, "-b", repo.Branch)
	}
	args = append(args, repo.URL, repo.LocalPath)

	cmd := exec.Command("git", args...)
	if repo.SSHKey != "" {
		// 使用SSH key
		keyFile := filepath.Join(h.workDir, ".keys", repo.ID)
		os.MkdirAll(filepath.Dir(keyFile), 0700)
		os.WriteFile(keyFile, []byte(repo.SSHKey), 0600)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyFile))
	}

	if err := cmd.Run(); err != nil {
		h.db.Model(repo).Update("status", 2) // error
		return err
	}

	now := time.Now()
	h.db.Model(repo).Updates(map[string]interface{}{
		"status":       0,
		"last_sync_at": now,
	})
	return nil
}

func (h *GitOpsHandler) pullRepo(repo *GitRepo) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = repo.LocalPath

	if repo.SSHKey != "" {
		keyFile := filepath.Join(h.workDir, ".keys", repo.ID)
		cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyFile))
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	h.db.Model(repo).Update("last_sync_at", time.Now())
	return nil
}

func (h *GitOpsHandler) commitAndPush(repo *GitRepo, filePath, message, author string) error {
	// git add
	addCmd := exec.Command("git", "add", filePath)
	addCmd.Dir = repo.LocalPath
	if err := addCmd.Run(); err != nil {
		return err
	}

	// git commit
	commitCmd := exec.Command("git", "commit", "-m", message, "--author", fmt.Sprintf("%s <ops@lazyautoops.local>", author))
	commitCmd.Dir = repo.LocalPath
	if err := commitCmd.Run(); err != nil {
		return err
	}

	// git push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = repo.LocalPath
	if repo.SSHKey != "" {
		keyFile := filepath.Join(h.workDir, ".keys", repo.ID)
		pushCmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", keyFile))
	}

	return pushCmd.Run()
}
