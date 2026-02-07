package entity

import (
	"time"
)

// MRPRunStatus MRP运行状态
const (
	MRPStatusRunning   = "RUNNING"
	MRPStatusCompleted = "COMPLETED"
	MRPStatusFailed    = "FAILED"
	MRPStatusApplied   = "APPLIED"
)

// MRPRun MRP运行记录
type MRPRun struct {
	ID            string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RunCode       string     `json:"run_code" gorm:"size:50;not null;uniqueIndex"`
	Status        string     `json:"status" gorm:"size:20;not null;default:RUNNING"`
	ProductID     string     `json:"product_id" gorm:"size:32"` // 空表示全部产品
	PlanningHorizon int     `json:"planning_horizon" gorm:"default:30"` // 计划范围（天）
	TotalItems    int        `json:"total_items" gorm:"default:0"`
	PRsGenerated  int        `json:"prs_generated" gorm:"default:0"`
	WOsGenerated  int        `json:"wos_generated" gorm:"default:0"`
	ErrorMessage  string     `json:"error_message" gorm:"type:text"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	AppliedAt     *time.Time `json:"applied_at"`
	CreatedBy     string     `json:"created_by" gorm:"size:64;not null"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (MRPRun) TableName() string {
	return "erp_mrp_runs"
}

// MRPResult MRP计算结果明细
type MRPResult struct {
	ID              string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	MRPRunID        string    `json:"mrp_run_id" gorm:"type:uuid;not null;index"`
	MaterialID      string    `json:"material_id" gorm:"size:32;not null"`
	MaterialCode    string    `json:"material_code" gorm:"size:64"`
	MaterialName    string    `json:"material_name" gorm:"size:128"`
	GrossRequirement float64  `json:"gross_requirement" gorm:"type:decimal(12,4);default:0"`
	OnHandStock     float64   `json:"on_hand_stock" gorm:"type:decimal(12,4);default:0"`
	InTransitQty    float64   `json:"in_transit_qty" gorm:"type:decimal(12,4);default:0"` // 采购在途
	InProductionQty float64   `json:"in_production_qty" gorm:"type:decimal(12,4);default:0"` // 工单在制
	SafetyStock     float64   `json:"safety_stock" gorm:"type:decimal(12,4);default:0"`
	NetRequirement  float64   `json:"net_requirement" gorm:"type:decimal(12,4);default:0"`
	PlannedOrderQty float64   `json:"planned_order_qty" gorm:"type:decimal(12,4);default:0"`
	ActionType      string    `json:"action_type" gorm:"size:20"` // PURCHASE, PRODUCE
	RequiredDate    *time.Time `json:"required_date"`
	LeadTimeDays    int       `json:"lead_time_days" gorm:"default:0"`
	OrderDate       *time.Time `json:"order_date"` // RequiredDate - LeadTime
	Unit            string    `json:"unit" gorm:"size:20;default:pcs"`
	Applied         bool      `json:"applied" gorm:"default:false"`
	CreatedAt       time.Time `json:"created_at"`
}

func (MRPResult) TableName() string {
	return "erp_mrp_results"
}

// FinanceRecord 财务记录（应付/应收）
type FinanceRecord struct {
	ID            string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	RecordCode    string     `json:"record_code" gorm:"size:50;not null;uniqueIndex"`
	RecordType    string     `json:"record_type" gorm:"size:20;not null"` // PAYABLE, RECEIVABLE
	ReferenceType string     `json:"reference_type" gorm:"size:20;not null"` // PO, SO
	ReferenceID   string     `json:"reference_id" gorm:"size:64;not null"`
	ReferenceCode string     `json:"reference_code" gorm:"size:50"`
	CounterpartyID string    `json:"counterparty_id" gorm:"size:64;not null"` // 供应商或客户ID
	CounterpartyName string  `json:"counterparty_name" gorm:"size:200"`
	Amount        float64    `json:"amount" gorm:"type:decimal(12,2);not null"`
	PaidAmount    float64    `json:"paid_amount" gorm:"type:decimal(12,2);default:0"`
	Currency      string     `json:"currency" gorm:"size:10;not null;default:CNY"`
	DueDate       *time.Time `json:"due_date"`
	PaidDate      *time.Time `json:"paid_date"`
	Status        string     `json:"status" gorm:"size:20;not null;default:PENDING"` // PENDING, PARTIAL, PAID, OVERDUE
	Notes         string     `json:"notes" gorm:"type:text"`
	CreatedBy     string     `json:"created_by" gorm:"size:64"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (FinanceRecord) TableName() string {
	return "erp_finance_records"
}
