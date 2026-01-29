package cicd

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler 定时发布调度器
type Scheduler struct {
	db      *gorm.DB
	handler *CICDHandler
	cron    *cron.Cron
	jobs    map[string]cron.EntryID
	mu      sync.RWMutex
}

func NewScheduler(db *gorm.DB, handler *CICDHandler) *Scheduler {
	return &Scheduler{
		db:      db,
		handler: handler,
		cron:    cron.New(cron.WithSeconds()),
		jobs:    make(map[string]cron.EntryID),
	}
}

func (s *Scheduler) Start() {
	s.loadSchedules()
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 移除所有现有任务
	for id, entryID := range s.jobs {
		s.cron.Remove(entryID)
		delete(s.jobs, id)
	}

	// 重新加载
	s.loadSchedules()
}

func (s *Scheduler) loadSchedules() {
	var schedules []CICDSchedule
	s.db.Where("enabled = ?", true).Find(&schedules)

	for _, schedule := range schedules {
		s.addJob(schedule)
	}
}

func (s *Scheduler) addJob(schedule CICDSchedule) {
	entryID, err := s.cron.AddFunc(schedule.Cron, func() {
		s.runSchedule(schedule.ID)
	})
	if err != nil {
		return
	}
	s.jobs[schedule.ID] = entryID

	// 更新下次执行时间
	entry := s.cron.Entry(entryID)
	s.db.Model(&schedule).Update("next_run_at", entry.Next)
}

func (s *Scheduler) runSchedule(scheduleID string) {
	var schedule CICDSchedule
	if err := s.db.First(&schedule, "id = ?", scheduleID).Error; err != nil {
		return
	}

	var pipeline CICDPipeline
	if err := s.db.First(&pipeline, "id = ?", schedule.PipelineID).Error; err != nil {
		return
	}

	// 解析参数
	var params map[string]string
	if schedule.Parameters != "" {
		json.Unmarshal([]byte(schedule.Parameters), &params)
	}

	// 触发构建
	s.handler.triggerBuild(&pipeline, params, "schedule", "scheduler")

	// 更新执行时间
	now := time.Now()
	s.db.Model(&schedule).Update("last_run_at", now)

	// 更新下次执行时间
	if entryID, ok := s.jobs[scheduleID]; ok {
		entry := s.cron.Entry(entryID)
		s.db.Model(&schedule).Update("next_run_at", entry.Next)
	}
}
