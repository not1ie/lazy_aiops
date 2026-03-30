package workorder

import "gorm.io/gorm"

type CreateOrderInput struct {
	TypeCode     string
	Title        string
	Content      string
	FormData     string
	Priority     int
	Submitter    string
	SubmitterID  string
	Assignee     string
	AssigneeID   string
	AISuggestion string
	AIRisk       string
}

func CreateOrderWithDefaults(db *gorm.DB, input CreateOrderInput) (*WorkOrder, error) {
	orderType := resolveTypeByCode(db, input.TypeCode)
	order := &WorkOrder{
		Title:        input.Title,
		Content:      input.Content,
		FormData:     input.FormData,
		Priority:     normalizedPriority(input.Priority),
		Status:       0,
		Submitter:    input.Submitter,
		SubmitterID:  input.SubmitterID,
		Assignee:     input.Assignee,
		AssigneeID:   input.AssigneeID,
		AISuggestion: input.AISuggestion,
		AIRisk:       input.AIRisk,
	}
	if orderType != nil {
		order.TypeID = orderType.ID
		order.TypeName = orderType.Name
	}
	if err := db.Create(order).Error; err != nil {
		return nil, err
	}
	createApprovalSteps(db, order, orderType)
	addComment(db, order.ID, "", "system", "工单已创建，等待审批")
	return order, nil
}

func resolveTypeByCode(db *gorm.DB, code string) *WorkOrderType {
	targetCode := code
	if targetCode == "" {
		targetCode = "change_apply"
	}

	var orderType WorkOrderType
	if err := db.Where("code = ?", targetCode).First(&orderType).Error; err == nil {
		return &orderType
	}
	if err := db.Where("code = ?", "change_apply").First(&orderType).Error; err == nil {
		return &orderType
	}
	if err := db.Where("enabled = ?", true).Order("created_at asc").First(&orderType).Error; err == nil {
		return &orderType
	}
	return nil
}

func normalizedPriority(priority int) int {
	if priority < 1 || priority > 4 {
		return 2
	}
	return priority
}
