package entity

import (
	"encoding/json"
	"time"
)

// Inspection 检验任务
type Inspection struct {
	ID             string    `json:"id" gorm:"primaryKey;size:32"`
	InspectionCode string   `json:"inspection_code" gorm:"size:32;uniqueIndex;not null"`
	POID           *string   `json:"po_id" gorm:"size:32"`
	POItemID       *string   `json:"po_item_id" gorm:"size:32"`
	SupplierID     *string   `json:"supplier_id" gorm:"size:32"`

	MaterialID   *string `json:"material_id" gorm:"size:32"`
	MaterialCode string  `json:"material_code" gorm:"size:50"`
	MaterialName string  `json:"material_name" gorm:"size:200"`

	// 检验信息
	Quantity  *float64 `json:"quantity" gorm:"type:decimal(10,2)"`
	SampleQty *int     `json:"sample_qty"`
	Status    string   `json:"status" gorm:"size:20;default:pending"` // pending/in_progress/completed
	Result    string   `json:"result" gorm:"size:20"`                 // passed/failed/conditional

	// 检验详情
	InspectionItems json.RawMessage `json:"inspection_items" gorm:"type:jsonb"`
	ReportURL       string          `json:"report_url" gorm:"size:500"`

	// 人员
	InspectorID *string    `json:"inspector_id" gorm:"size:32"`
	InspectedAt *time.Time `json:"inspected_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Notes     string    `json:"notes" gorm:"type:text"`
}

func (Inspection) TableName() string {
	return "srm_inspections"
}

// 检验状态
const (
	InspectionStatusPending    = "pending"
	InspectionStatusInProgress = "in_progress"
	InspectionStatusCompleted  = "completed"
)

// 检验结果
const (
	InspectionResultPassed      = "passed"
	InspectionResultFailed      = "failed"
	InspectionResultConditional = "conditional"
)

// CorrectiveAction 8D改进单
type CorrectiveAction struct {
	ID           string  `json:"id" gorm:"primaryKey;size:32"`
	CACode       string  `json:"ca_code" gorm:"size:32;uniqueIndex;not null"`
	InspectionID string  `json:"inspection_id" gorm:"size:32;not null"`
	SupplierID   string  `json:"supplier_id" gorm:"size:32;not null"`

	// 问题描述
	ProblemDesc string `json:"problem_desc" gorm:"type:text;not null"`
	Severity    string `json:"severity" gorm:"size:20"` // critical/major/minor

	// 8D流程
	Status           string `json:"status" gorm:"size:20;default:open"` // open/responded/verified/closed
	RootCause        string `json:"root_cause" gorm:"type:text"`
	CorrectiveAction string `json:"corrective_action" gorm:"type:text"`
	PreventiveAction string `json:"preventive_action" gorm:"type:text"`

	// 时间
	ResponseDeadline *time.Time `json:"response_deadline"`
	RespondedAt      *time.Time `json:"responded_at"`
	VerifiedAt       *time.Time `json:"verified_at"`
	ClosedAt         *time.Time `json:"closed_at"`

	CreatedBy string    `json:"created_by" gorm:"size:32"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CorrectiveAction) TableName() string {
	return "srm_corrective_actions"
}
