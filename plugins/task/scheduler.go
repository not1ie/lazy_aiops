package task

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 任务调度器
type Scheduler struct {
	db       *gorm.DB
	cron     *cron.Cron
	tasks    map[string]cron.EntryID // taskID -> cronEntryID
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewScheduler 创建调度器
func NewScheduler(db *gorm.DB) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:     db,
		cron:   cron.New(cron.WithSeconds()),
		tasks:  make(map[string]cron.EntryID),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	log.Println("[Task Scheduler] Starting...")
	
	// 加载所有启用的定时任务
	if err := s.LoadTasks(); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}
	
	// 启动cron
	s.cron.Start()
	
	// 启动定期重载任务的goroutine
	go s.reloadLoop()
	
	log.Println("[Task Scheduler] Started successfully")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	log.Println("[Task Scheduler] Stopping...")
	s.cancel()
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[Task Scheduler] Stopped")
	return nil
}

// LoadTasks 加载所有定时任务
func (s *Scheduler) LoadTasks() error {
	var tasks []Task
	if err := s.db.Where("enabled = ? AND cron != ?", true, "").Find(&tasks).Error; err != nil {
		return err
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 清除现有任务
	for _, entryID := range s.tasks {
		s.cron.Remove(entryID)
	}
	s.tasks = make(map[string]cron.EntryID)
	
	// 添加新任务
	for _, task := range tasks {
		if err := s.addTask(task); err != nil {
			log.Printf("[Task Scheduler] Failed to add task %s: %v", task.Name, err)
			continue
		}
	}
	
	log.Printf("[Task Scheduler] Loaded %d tasks", len(s.tasks))
	return nil
}

// addTask 添加任务到调度器（需要持有锁）
func (s *Scheduler) addTask(task Task) error {
	entryID, err := s.cron.AddFunc(task.Cron, func() {
		s.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", task.Cron, err)
	}
	
	s.tasks[task.ID] = entryID
	
	// 更新下次执行时间
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	s.db.Model(&Task{}).Where("id = ?", task.ID).Update("next_run_at", nextRun)
	
	log.Printf("[Task Scheduler] Added task: %s (cron: %s, next: %s)", 
		task.Name, task.Cron, nextRun.Format("2006-01-02 15:04:05"))
	
	return nil
}

// AddTask 添加单个任务
func (s *Scheduler) AddTask(task Task) error {
	if task.Cron == "" || !task.Enabled {
		return nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 如果任务已存在，先移除
	if entryID, exists := s.tasks[task.ID]; exists {
		s.cron.Remove(entryID)
	}
	
	return s.addTask(task)
}

// RemoveTask 移除任务
func (s *Scheduler) RemoveTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if entryID, exists := s.tasks[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, taskID)
		log.Printf("[Task Scheduler] Removed task: %s", taskID)
	}
}

// executeTask 执行任务
func (s *Scheduler) executeTask(task Task) {
	log.Printf("[Task Scheduler] Executing task: %s", task.Name)
	
	// 创建执行记录
	execution := TaskExecution{
		TaskID:   task.ID,
		TaskName: task.Name,
		Status:   0, // 运行中
		StartAt:  time.Now(),
		Executor: "scheduler",
	}
	s.db.Create(&execution)
	
	// 更新任务最后执行时间
	now := time.Now()
	s.db.Model(&Task{}).Where("id = ?", task.ID).Update("last_run_at", now)
	
	// 执行任务
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(task.Timeout)*time.Second)
		defer cancel()
		
		output, err := s.runTask(ctx, task)
		
		endTime := time.Now()
		duration := int(endTime.Sub(execution.StartAt).Seconds())
		
		// 更新执行记录
		updates := map[string]interface{}{
			"end_at":   endTime,
			"duration": duration,
			"output":   output,
		}
		
		if err != nil {
			updates["status"] = 2 // 失败
			updates["error"] = err.Error()
			log.Printf("[Task Scheduler] Task %s failed: %v", task.Name, err)
		} else {
			updates["status"] = 1 // 成功
			log.Printf("[Task Scheduler] Task %s completed successfully", task.Name)
		}
		
		s.db.Model(&TaskExecution{}).Where("id = ?", execution.ID).Updates(updates)
	}()
}

// runTask 运行任务
func (s *Scheduler) runTask(ctx context.Context, task Task) (string, error) {
	switch task.Type {
	case "shell":
		return s.runShellTask(ctx, task)
	case "python":
		return s.runPythonTask(ctx, task)
	default:
		return "", fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

// runShellTask 执行Shell任务
func (s *Scheduler) runShellTask(ctx context.Context, task Task) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", task.Content)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runPythonTask 执行Python任务
func (s *Scheduler) runPythonTask(ctx context.Context, task Task) (string, error) {
	cmd := exec.CommandContext(ctx, "python3", "-c", task.Content)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// reloadLoop 定期重载任务
func (s *Scheduler) reloadLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			log.Println("[Task Scheduler] Reloading tasks...")
			if err := s.LoadTasks(); err != nil {
				log.Printf("[Task Scheduler] Failed to reload tasks: %v", err)
			}
		}
	}
}

// GetTaskStatus 获取任务状态
func (s *Scheduler) GetTaskStatus(taskID string) (bool, *time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entryID, exists := s.tasks[taskID]
	if !exists {
		return false, nil
	}
	
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	return true, &nextRun
}

// ValidateCron 验证Cron表达式
func ValidateCron(cronExpr string) error {
	if cronExpr == "" {
		return nil // 空表达式表示手动执行
	}
	
	// 支持标准5字段和带秒的6字段
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 && len(fields) != 6 {
		return fmt.Errorf("invalid cron expression: expected 5 or 6 fields, got %d", len(fields))
	}
	
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(cronExpr)
	return err
}
