package workflow

import (
	"fmt"
	"sync"

	"gorm.io/gorm"
)

var (
	defaultEngineMu sync.RWMutex
	defaultEngine   *Engine
)

// RegisterDefaultEngine registers the runtime engine used by cross-plugin callers.
func RegisterDefaultEngine(engine *Engine) {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()
	defaultEngine = engine
}

func resolveEngine(db *gorm.DB) *Engine {
	defaultEngineMu.RLock()
	engine := defaultEngine
	defaultEngineMu.RUnlock()
	if engine != nil {
		return engine
	}
	return NewEngine(db)
}

// ExecuteWorkflowByID executes a workflow by ID with optional runtime variables.
func ExecuteWorkflowByID(db *gorm.DB, workflowID string, variables map[string]interface{}, triggerBy string) (*WorkflowExecution, error) {
	if db == nil {
		return nil, fmt.Errorf("workflow db is nil")
	}
	var wf Workflow
	if err := db.First(&wf, "id = ?", workflowID).Error; err != nil {
		return nil, err
	}
	if variables == nil {
		variables = make(map[string]interface{})
	}
	return resolveEngine(db).Execute(&wf, variables, triggerBy)
}

// CancelWorkflowExecutionByID cancels a workflow execution by ID.
func CancelWorkflowExecutionByID(db *gorm.DB, executionID string) error {
	if db == nil {
		return fmt.Errorf("workflow db is nil")
	}
	return resolveEngine(db).Cancel(executionID)
}
