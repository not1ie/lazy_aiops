package cost

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CostHandler struct {
	db *gorm.DB
}

func NewCostHandler(db *gorm.DB) *CostHandler {
	return &CostHandler{db: db}
}

// ListAccounts 账号列表
func (h *CostHandler) ListAccounts(c *gin.Context) {
	var accounts []CloudAccount
	if err := h.db.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": accounts})
}

// CreateAccount 创建账号
func (h *CostHandler) CreateAccount(c *gin.Context) {
	var account CloudAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": account})
}

// UpdateAccount 更新账号
func (h *CostHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var account CloudAccount
	if err := h.db.First(&account, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": account})
}

// DeleteAccount 删除账号
func (h *CostHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CloudAccount{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// SyncCost 同步费用
func (h *CostHandler) SyncCost(c *gin.Context) {
	id := c.Param("id")
	var account CloudAccount
	if err := h.db.First(&account, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}

	var req struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	c.ShouldBindJSON(&req)

	// TODO: 根据provider调用对应云厂商API
	// 这里模拟同步
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "同步任务已提交"})
}

// GetCostSummary 费用汇总
func (h *CostHandler) GetCostSummary(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	accountID := c.Query("account_id")

	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	query := h.db.Model(&CostRecord{}).Where("billing_date BETWEEN ? AND ?", startDate, endDate)
	if accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}

	// 总费用
	var totalCost float64
	query.Select("COALESCE(SUM(amount), 0)").Scan(&totalCost)

	// 按产品分组
	var productCosts []struct {
		ProductCode string  `json:"product_code"`
		ProductName string  `json:"product_name"`
		Amount      float64 `json:"amount"`
	}
	byProductQuery := h.db.Model(&CostRecord{}).
		Select("product_code, product_name, SUM(amount) as amount").
		Where("billing_date BETWEEN ? AND ?", startDate, endDate)
	if accountID != "" {
		byProductQuery = byProductQuery.Where("account_id = ?", accountID)
	}
	byProductQuery.
		Group("product_code, product_name").
		Order("amount DESC").
		Find(&productCosts)

	// 按日期趋势
	var dailyCosts []struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}
	dailyQuery := h.db.Model(&CostRecord{}).
		Select("DATE(billing_date) as date, SUM(amount) as amount").
		Where("billing_date BETWEEN ? AND ?", startDate, endDate)
	if accountID != "" {
		dailyQuery = dailyQuery.Where("account_id = ?", accountID)
	}
	dailyQuery.
		Group("DATE(billing_date)").
		Order("date").
		Find(&dailyCosts)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":       totalCost,
			"by_product":  productCosts,
			"daily_trend": dailyCosts,
			"start_date":  startDate,
			"end_date":    endDate,
		},
	})
}

// ListCostRecords 费用记录列表
func (h *CostHandler) ListCostRecords(c *gin.Context) {
	var records []CostRecord
	query := h.db.Order("billing_date DESC")

	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if productCode := c.Query("product_code"); productCode != "" {
		query = query.Where("product_code = ?", productCode)
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("billing_date >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("billing_date <= ?", endDate)
	}

	if err := query.Limit(500).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

// ListBudgets 预算列表
func (h *CostHandler) ListBudgets(c *gin.Context) {
	var budgets []CostBudget
	if err := h.db.Order("created_at DESC").Find(&budgets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": budgets})
}

// CreateBudget 创建预算
func (h *CostHandler) CreateBudget(c *gin.Context) {
	var budget CostBudget
	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": budget})
}

// UpdateBudget 更新预算
func (h *CostHandler) UpdateBudget(c *gin.Context) {
	id := c.Param("id")
	var budget CostBudget
	if err := h.db.First(&budget, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "预算不存在"})
		return
	}
	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Save(&budget).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": budget})
}

// DeleteBudget 删除预算
func (h *CostHandler) DeleteBudget(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CostBudget{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// GetBudgetStatus 预算执行状态
func (h *CostHandler) GetBudgetStatus(c *gin.Context) {
	var budgets []CostBudget
	h.db.Where("status = 1").Find(&budgets)

	results := make([]map[string]interface{}, 0)
	for _, budget := range budgets {
		var currentCost float64
		h.db.Model(&CostRecord{}).
			Where("account_id = ? AND billing_date BETWEEN ? AND ?", budget.AccountID, budget.StartDate, budget.EndDate).
			Select("COALESCE(SUM(amount), 0)").Scan(&currentCost)

		percentage := 0.0
		if budget.Amount > 0 {
			percentage = currentCost / budget.Amount * 100
		}

		results = append(results, map[string]interface{}{
			"budget_id":     budget.ID,
			"budget_name":   budget.Name,
			"budget_amount": budget.Amount,
			"current_cost":  currentCost,
			"percentage":    percentage,
			"alert_at":      budget.AlertAt,
			"is_alert":      percentage >= budget.AlertAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results})
}

// ListAlerts 告警列表
func (h *CostHandler) ListAlerts(c *gin.Context) {
	var alerts []CostAlert
	query := h.db.Order("alert_at DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Limit(100).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": alerts})
}

// AckAlert 确认告警
func (h *CostHandler) AckAlert(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&CostAlert{}).Where("id = ?", id).Update("status", 1)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已确认"})
}

// ListOptimizations 优化建议列表
func (h *CostHandler) ListOptimizations(c *gin.Context) {
	var opts []CostOptimization
	query := h.db.Order("save_amount DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&opts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": opts})
}

// UpdateOptimization 更新优化建议状态
func (h *CostHandler) UpdateOptimization(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	h.db.Model(&CostOptimization{}).Where("id = ?", id).Update("status", req.Status)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// AnalyzeOptimization 分析优化建议
func (h *CostHandler) AnalyzeOptimization(c *gin.Context) {
	type resourceCostStat struct {
		AccountID    string  `json:"account_id"`
		ResourceID   string  `json:"resource_id"`
		ResourceName string  `json:"resource_name"`
		ProductCode  string  `json:"product_code"`
		ProductName  string  `json:"product_name"`
		Amount       float64 `json:"amount"`
	}

	since := time.Now().AddDate(0, 0, -30)
	var stats []resourceCostStat
	if err := h.db.Model(&CostRecord{}).
		Select("account_id, resource_id, resource_name, product_code, product_name, SUM(amount) AS amount").
		Where("billing_date >= ?", since).
		Group("account_id, resource_id, resource_name, product_code, product_name").
		Having("SUM(amount) > 0").
		Order("amount DESC").
		Limit(200).
		Scan(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	if len(stats) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "暂无可分析的费用数据", "data": gin.H{"created": 0, "updated": 0}})
		return
	}

	created := 0
	updated := 0
	skipped := 0
	for _, stat := range stats {
		resourceID := strings.TrimSpace(stat.ResourceID)
		if resourceID == "" {
			resourceID = strings.TrimSpace(stat.ProductCode)
		}
		if resourceID == "" {
			skipped++
			continue
		}
		resourceName := strings.TrimSpace(stat.ResourceName)
		if resourceName == "" {
			resourceName = strings.TrimSpace(stat.ProductName)
		}
		if resourceName == "" {
			resourceName = resourceID
		}

		resourceType := inferResourceType(stat.ProductCode, stat.ProductName)
		optType, saveRatio, reason := getOptimizationPolicy(resourceType, stat.Amount)
		if saveRatio <= 0 {
			skipped++
			continue
		}

		currentCost := roundCost(stat.Amount)
		suggestCost := roundCost(currentCost * (1 - saveRatio))
		saveAmount := roundCost(currentCost - suggestCost)
		opt := CostOptimization{
			AccountID:    stat.AccountID,
			ResourceID:   resourceID,
			ResourceName: resourceName,
			ResourceType: resourceType,
			OptType:      optType,
			CurrentSpec:  fmt.Sprintf("近30天成本 %.2f", currentCost),
			SuggestSpec:  fmt.Sprintf("预计优化 %.0f%%", saveRatio*100),
			CurrentCost:  currentCost,
			SuggestCost:  suggestCost,
			SaveAmount:   saveAmount,
			Reason:       reason,
			Status:       0,
		}

		var existing CostOptimization
		err := h.db.Where("account_id = ? AND resource_id = ? AND opt_type = ?",
			stat.AccountID, resourceID, optType).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if createErr := h.db.Create(&opt).Error; createErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": createErr.Error()})
					return
				}
				created++
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}

		updates := map[string]interface{}{
			"resource_name": resourceName,
			"resource_type": resourceType,
			"current_spec":  opt.CurrentSpec,
			"suggest_spec":  opt.SuggestSpec,
			"current_cost":  currentCost,
			"suggest_cost":  suggestCost,
			"save_amount":   saveAmount,
			"reason":        reason,
		}
		if existing.Status == 0 {
			updates["status"] = 0
		}
		if updateErr := h.db.Model(&existing).Updates(updates).Error; updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": updateErr.Error()})
			return
		}
		updated++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "分析完成",
		"data": gin.H{
			"created": created,
			"updated": updated,
			"skipped": skipped,
			"window":  "30d",
		},
	})
}

func inferResourceType(productCode, productName string) string {
	value := strings.ToLower(strings.TrimSpace(productCode + " " + productName))
	switch {
	case strings.Contains(value, "ecs"),
		strings.Contains(value, "ec2"),
		strings.Contains(value, "cvm"),
		strings.Contains(value, "compute"),
		strings.Contains(value, "node"):
		return "compute"
	case strings.Contains(value, "rds"),
		strings.Contains(value, "mysql"),
		strings.Contains(value, "postgres"),
		strings.Contains(value, "mongo"),
		strings.Contains(value, "database"),
		strings.Contains(value, "db"):
		return "database"
	case strings.Contains(value, "oss"),
		strings.Contains(value, "s3"),
		strings.Contains(value, "disk"),
		strings.Contains(value, "storage"),
		strings.Contains(value, "nas"):
		return "storage"
	case strings.Contains(value, "redis"),
		strings.Contains(value, "cache"):
		return "cache"
	default:
		return "general"
	}
}

func getOptimizationPolicy(resourceType string, amount float64) (optType string, saveRatio float64, reason string) {
	switch resourceType {
	case "storage":
		optType = "release"
	case "database":
		optType = "reserved"
	default:
		optType = "downgrade"
	}

	switch {
	case amount >= 10000:
		saveRatio = 0.25
		reason = "近30天费用较高，建议优先进行规格优化或长期预留"
	case amount >= 3000:
		saveRatio = 0.18
		reason = "成本处于高位，建议评估降配或预留实例"
	case amount >= 800:
		saveRatio = 0.12
		reason = "存在持续支出，建议按利用率进行规格优化"
	case amount >= 200:
		saveRatio = 0.08
		reason = "成本可进一步压缩，建议做闲置检查与容量治理"
	default:
		saveRatio = 0
		reason = "成本较低，暂不推荐优化"
	}

	if resourceType == "storage" && saveRatio > 0 && saveRatio < 0.15 {
		saveRatio = 0.15
	}
	return optType, saveRatio, reason
}

func roundCost(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// GetCostTrend 费用趋势
func (h *CostHandler) GetCostTrend(c *gin.Context) {
	months := 6
	if v := c.Query("months"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 1 && parsed <= 24 {
			months = parsed
		}
	}
	accountID := c.Query("account_id")

	var trends []struct {
		Month  string  `json:"month"`
		Amount float64 `json:"amount"`
	}

	for i := months - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, -i, 0)
		month := date.Format("2006-01")
		startDate := date.Format("2006-01") + "-01"
		endDate := date.AddDate(0, 1, -1).Format("2006-01-02")

		var amount float64
		query := h.db.Model(&CostRecord{}).Where("billing_date BETWEEN ? AND ?", startDate, endDate)
		if accountID != "" {
			query = query.Where("account_id = ?", accountID)
		}
		query.Select("COALESCE(SUM(amount), 0)").Scan(&amount)

		trends = append(trends, struct {
			Month  string  `json:"month"`
			Amount float64 `json:"amount"`
		}{Month: month, Amount: amount})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": trends})
}

// GetTopResources 费用TOP资源
func (h *CostHandler) GetTopResources(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	accountID := c.Query("account_id")
	if startDate == "" {
		startDate = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	var resources []struct {
		ResourceID   string  `json:"resource_id"`
		ResourceName string  `json:"resource_name"`
		ProductName  string  `json:"product_name"`
		Amount       float64 `json:"amount"`
	}

	query := h.db.Model(&CostRecord{}).
		Select("resource_id, resource_name, product_name, SUM(amount) as amount").
		Where("billing_date BETWEEN ? AND ?", startDate, endDate)
	if accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	query.
		Group("resource_id, resource_name, product_name").
		Order("amount DESC").
		Limit(limit).
		Find(&resources)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resources})
}
