package oncall

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OnCallHandler struct {
	db *gorm.DB
}

func NewOnCallHandler(db *gorm.DB) *OnCallHandler {
	return &OnCallHandler{db: db}
}

// ListSchedules 排班列表
func (h *OnCallHandler) ListSchedules(c *gin.Context) {
	var schedules []OnCallSchedule
	if err := h.db.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedules})
}

// CreateSchedule 创建排班
func (h *OnCallHandler) CreateSchedule(c *gin.Context) {
	var schedule OnCallSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// GetSchedule 获取排班详情
func (h *OnCallHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule OnCallSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "排班不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// UpdateSchedule 更新排班
func (h *OnCallHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	var schedule OnCallSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "排班不存在"})
		return
	}
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": schedule})
}

// DeleteSchedule 删除排班
func (h *OnCallHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&OnCallSchedule{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// GenerateShifts 生成班次
func (h *OnCallHandler) GenerateShifts(c *gin.Context) {
	id := c.Param("id")
	var schedule OnCallSchedule
	if err := h.db.First(&schedule, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "排班不存在"})
		return
	}

	var req struct {
		StartDate string `json:"start_date" binding:"required"` // 2024-01-01
		Days      int    `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)

	// 解析轮换规则
	var rotation []map[string]interface{}
	json.Unmarshal([]byte(schedule.Rotation), &rotation)

	if len(rotation) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "未配置轮换规则"})
		return
	}

	// 生成班次
	shifts := make([]OnCallShift, 0)
	for i := 0; i < req.Days; i++ {
		date := startDate.AddDate(0, 0, i)
		memberIdx := i % len(rotation)
		member := rotation[memberIdx]

		shift := OnCallShift{
			ScheduleID: schedule.ID,
			UserID:     member["user_id"].(string),
			Username:   member["username"].(string),
			StartAt:    date,
			EndAt:      date.AddDate(0, 0, 1),
			Type:       "primary",
			Status:     1,
		}
		if phone, ok := member["phone"].(string); ok {
			shift.Phone = phone
		}
		if email, ok := member["email"].(string); ok {
			shift.Email = email
		}

		shifts = append(shifts, shift)
	}

	// 批量创建
	for _, shift := range shifts {
		h.db.Create(&shift)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": len(shifts)}})
}

// ListShifts 班次列表
func (h *OnCallHandler) ListShifts(c *gin.Context) {
	id := c.Param("id")
	var shifts []OnCallShift
	query := h.db.Where("schedule_id = ?", id).Order("start_at")

	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("start_at >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("end_at <= ?", endDate)
	}

	if err := query.Find(&shifts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": shifts})
}

// GetCurrentOnCall 获取当前值班人
func (h *OnCallHandler) GetCurrentOnCall(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()

	var shift OnCallShift
	if err := h.db.Where("schedule_id = ? AND start_at <= ? AND end_at > ? AND status = 1", id, now, now).First(&shift).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "当前无值班人"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": shift})
}

// SwapShift 换班
func (h *OnCallHandler) SwapShift(c *gin.Context) {
	shiftID := c.Param("shift_id")
	var shift OnCallShift
	if err := h.db.First(&shift, "id = ?", shiftID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "班次不存在"})
		return
	}

	var req struct {
		OverrideUserID string `json:"override_user_id" binding:"required"`
		OverrideUser   string `json:"override_user" binding:"required"`
		Reason         string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 创建换班记录
	override := OnCallOverride{
		ScheduleID:   shift.ScheduleID,
		OriginalUser: shift.UserID,
		OverrideUser: req.OverrideUserID,
		StartAt:      shift.StartAt,
		EndAt:        shift.EndAt,
		Reason:       req.Reason,
		CreatedBy:    c.GetString("username"),
	}
	h.db.Create(&override)

	// 更新班次
	h.db.Model(&shift).Updates(map[string]interface{}{
		"user_id":    req.OverrideUserID,
		"username":   req.OverrideUser,
		"swapped_by": shift.UserID,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "换班成功"})
}

// ListTeams 团队列表
func (h *OnCallHandler) ListTeams(c *gin.Context) {
	var teams []OnCallTeam
	if err := h.db.Find(&teams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": teams})
}

// CreateTeam 创建团队
func (h *OnCallHandler) CreateTeam(c *gin.Context) {
	var team OnCallTeam
	if err := c.ShouldBindJSON(&team); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": team})
}

// ListEscalations 升级策略列表
func (h *OnCallHandler) ListEscalations(c *gin.Context) {
	var escalations []OnCallEscalation
	query := h.db.Order("created_at DESC")
	if scheduleID := c.Query("schedule_id"); scheduleID != "" {
		query = query.Where("schedule_id = ?", scheduleID)
	}
	if err := query.Find(&escalations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": escalations})
}

// CreateEscalation 创建升级策略
func (h *OnCallHandler) CreateEscalation(c *gin.Context) {
	var escalation OnCallEscalation
	if err := c.ShouldBindJSON(&escalation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&escalation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": escalation})
}

// UpdateEscalation 更新升级策略
func (h *OnCallHandler) UpdateEscalation(c *gin.Context) {
	id := c.Param("id")
	var escalation OnCallEscalation
	if err := h.db.First(&escalation, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "升级策略不存在"})
		return
	}
	if err := c.ShouldBindJSON(&escalation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&escalation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": escalation})
}

// DeleteEscalation 删除升级策略
func (h *OnCallHandler) DeleteEscalation(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&OnCallEscalation{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// WhoIsOnCall 查询谁在值班
func (h *OnCallHandler) WhoIsOnCall(c *gin.Context) {
	now := time.Now()
	var shifts []OnCallShift
	if err := h.db.Where("start_at <= ? AND end_at > ? AND status = 1", now, now).Find(&shifts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": shifts})
}
