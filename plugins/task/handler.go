package task

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskHandler struct {
	db        *gorm.DB
	scheduler *Scheduler
}

func NewTaskHandler(db *gorm.DB, scheduler *Scheduler) *TaskHandler {
	return &TaskHandler{
		db:        db,
		scheduler: scheduler,
	}
}

// List 任务列表
func (h *TaskHandler) List(c *gin.Context) {
	var tasks []Task
	if err := h.db.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tasks})
}

// Create 创建任务
func (h *TaskHandler) Create(c *gin.Context) {
	var task Task
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	
	// 验证Cron表达式
	if task.Cron != "" {
		if err := ValidateCron(task.Cron); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Cron表达式无效: " + err.Error()})
			return
		}
	}
	
	task.CreatedBy = c.GetString("username")
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	
	// 如果是定时任务且已启用，添加到调度器
	if task.Cron != "" && task.Enabled && h.scheduler != nil {
		h.scheduler.AddTask(task)
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": task})
}

// Get 获取任务详情
func (h *TaskHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := h.db.First(&task, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": task})
}

// Update 更新任务
func (h *TaskHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := h.db.First(&task, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}
	
	var updates Task
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	
	// 验证Cron表达式
	if updates.Cron != "" {
		if err := ValidateCron(updates.Cron); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Cron表达式无效: " + err.Error()})
			return
		}
	}
	
	// 更新任务
	updates.ID = task.ID
	if err := h.db.Save(&updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	
	// 更新调度器
	if h.scheduler != nil {
		if updates.Cron != "" && updates.Enabled {
			h.scheduler.AddTask(updates)
		} else {
			h.scheduler.RemoveTask(updates.ID)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": updates})
}

// Delete 删除任务
func (h *TaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	
	// 从调度器移除
	if h.scheduler != nil {
		h.scheduler.RemoveTask(id)
	}
	
	if err := h.db.Delete(&Task{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// Run 执行任务
func (h *TaskHandler) Run(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := h.db.First(&task, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}

	// 入队执行
	if h.scheduler != nil {
		exec, err := h.scheduler.EnqueueTask(task, c.GetString("username"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": exec, "message": "任务已提交执行"})
		return
	}

	// 无调度器兜底：直接创建记录
	exec := TaskExecution{
		TaskID:   task.ID,
		TaskName: task.Name,
		Status:   0,
		StartAt:  time.Now(),
		Executor: c.GetString("username"),
	}
	h.db.Create(&exec)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": exec, "message": "任务已提交执行"})
}

// ListExecutions 执行记录列表
func (h *TaskHandler) ListExecutions(c *gin.Context) {
	var execs []TaskExecution
	query := h.db.Order("created_at DESC")

	if taskID := c.Query("task_id"); taskID != "" {
		query = query.Where("task_id = ?", taskID)
	}

	if err := query.Limit(100).Find(&execs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": execs})
}

// GetExecution 获取执行详情
func (h *TaskHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	var exec TaskExecution
	if err := h.db.First(&exec, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "执行记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": exec})
}

// Enable 启用任务
func (h *TaskHandler) Enable(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := h.db.First(&task, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}
	
	task.Enabled = true
	if err := h.db.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	
	// 添加到调度器
	if task.Cron != "" && h.scheduler != nil {
		h.scheduler.AddTask(task)
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "任务已启用"})
}

// Disable 禁用任务
func (h *TaskHandler) Disable(c *gin.Context) {
	id := c.Param("id")
	var task Task
	if err := h.db.First(&task, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "任务不存在"})
		return
	}
	
	task.Enabled = false
	if err := h.db.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	
	// 从调度器移除
	if h.scheduler != nil {
		h.scheduler.RemoveTask(task.ID)
	}
	
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "任务已禁用"})
}
