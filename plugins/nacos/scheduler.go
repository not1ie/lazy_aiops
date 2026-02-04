package nacos

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler Nacos配置同步调度器
// 只负责定时调用配置同步

type Scheduler struct {
	db       *gorm.DB
	cron     *cron.Cron
	schedules map[string]cron.EntryID
	handler  *NacosHandler
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewScheduler(db *gorm.DB, handler *NacosHandler) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		db:       db,
		cron:     cron.New(cron.WithSeconds()),
		schedules: make(map[string]cron.EntryID),
		handler:  handler,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *Scheduler) Start() error {
	if s == nil {
		return nil
	}
	log.Println("[Nacos Scheduler] Starting...")
	if err := s.LoadSchedules(); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}
	s.cron.Start()
	log.Println("[Nacos Scheduler] Started")
	return nil
}

func (s *Scheduler) Stop() error {
	if s == nil {
		return nil
	}
	log.Println("[Nacos Scheduler] Stopping...")
	s.cancel()
	ctx := s.cron.Stop()
	<-ctx.Done()
	log.Println("[Nacos Scheduler] Stopped")
	return nil
}

func (s *Scheduler) LoadSchedules() error {
	var schedules []NacosSyncSchedule
	if err := s.db.Where("enabled = ? AND cron != ?", true, "").Find(&schedules).Error; err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entryID := range s.schedules {
		s.cron.Remove(entryID)
	}
	s.schedules = make(map[string]cron.EntryID)

	for _, schedule := range schedules {
		if err := s.addSchedule(schedule); err != nil {
			log.Printf("[Nacos Scheduler] Failed to add schedule %s: %v", schedule.Name, err)
			continue
		}
	}

	log.Printf("[Nacos Scheduler] Loaded %d schedules", len(s.schedules))
	return nil
}

func (s *Scheduler) Reload() error {
	return s.LoadSchedules()
}

func (s *Scheduler) AddSchedule(schedule NacosSyncSchedule) error {
	if schedule.Cron == "" || !schedule.Enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.schedules[schedule.ID]; exists {
		s.cron.Remove(entryID)
	}

	return s.addSchedule(schedule)
}

func (s *Scheduler) RemoveSchedule(scheduleID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.schedules[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.schedules, scheduleID)
	}
}

func (s *Scheduler) addSchedule(schedule NacosSyncSchedule) error {
	entryID, err := s.cron.AddFunc(schedule.Cron, func() {
		s.execute(schedule)
	})
	if err != nil {
		return err
	}

	s.schedules[schedule.ID] = entryID
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	s.db.Model(&NacosSyncSchedule{}).Where("id = ?", schedule.ID).Update("next_run_at", nextRun)
	return nil
}

func (s *Scheduler) execute(schedule NacosSyncSchedule) {
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	if s.handler == nil {
		return
	}

	_, _, err := s.handler.syncConfigsForServer(schedule.ServerID)
	if err != nil {
		log.Printf("[Nacos Scheduler] Sync failed: %v", err)
	}

	now := time.Now()
	s.db.Model(&NacosSyncSchedule{}).Where("id = ?", schedule.ID).Update("last_run_at", now)
}
