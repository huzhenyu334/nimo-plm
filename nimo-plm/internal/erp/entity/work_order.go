package entity

import (
	"time"
)

// WorkOrderStatus 工单状态
const (
	WOStatusCreated    = "CREATED"
	WOStatusPlanned    = "PLANNED"
	WOStatusReleased   = "RELEASED"
	WOStatusInProgress = "IN_PROGRESS"
	WOStatusCompleted  = "COMPLETED"
	WOStatusClosed     = "CLOSED"
)

// WorkOrder 生产工单
type WorkOrder struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	WOCode       string     `json:"wo_code" gorm:"size:50;not null;uniqueIndex"`
	ProductID    string     `json:"product_id" gorm:"size:32;not null;index"`
	ProductCode  string     `json:"product_code" gorm:"size:64"`
	ProductName  string     `json:"product_name" gorm:"size:128"`
	BOMID        string     `json:"bom_id" gorm:"size:32;not null"`
	BOMVersion   string     `json:"bom_version" gorm:"size:16"`
	PlannedQty   float64    `json:"planned_qty" gorm:"type:decimal(12,4);not null"`
	CompletedQty float64    `json:"completed_qty" gorm:"type:decimal(12,4);default:0"`
	ScrapQty     float64    `json:"scrap_qty" gorm:"type:decimal(12,4);default:0"`
	Status       string     `json:"status" gorm:"size:20;not null;default:CREATED"`
	Priority     int        `json:"priority" gorm:"default:0"` // 0=普通, 1=紧急, 2=特急
	PlannedStart *time.Time `json:"planned_start"`
	PlannedEnd   *time.Time `json:"planned_end"`
	ActualStart  *time.Time `json:"actual_start"`
	ActualEnd    *time.Time `json:"actual_end"`
	WarehouseID  string     `json:"warehouse_id" gorm:"type:uuid"` // 成品入库仓库
	SourceType   string     `json:"source_type" gorm:"size:20"`    // MRP, MANUAL
	SourceID     string     `json:"source_id" gorm:"size:64"`
	Notes        string     `json:"notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:64;not null"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`

	Materials []WorkOrderMaterial `json:"materials,omitempty" gorm:"foreignKey:WorkOrderID"`
	Reports   []WorkOrderReport   `json:"reports,omitempty" gorm:"foreignKey:WorkOrderID"`
}

func (WorkOrder) TableName() string {
	return "erp_work_orders"
}

// WorkOrderMaterial 工单物料需求
type WorkOrderMaterial struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	WorkOrderID  string    `json:"work_order_id" gorm:"type:uuid;not null;index"`
	MaterialID   string    `json:"material_id" gorm:"size:32;not null"`
	MaterialCode string    `json:"material_code" gorm:"size:64"`
	MaterialName string    `json:"material_name" gorm:"size:128"`
	RequiredQty  float64   `json:"required_qty" gorm:"type:decimal(12,4);not null"`
	IssuedQty    float64   `json:"issued_qty" gorm:"type:decimal(12,4);default:0"`
	Unit         string    `json:"unit" gorm:"size:20;not null;default:pcs"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (WorkOrderMaterial) TableName() string {
	return "erp_work_order_materials"
}

// WorkOrderReport 工单报工记录
type WorkOrderReport struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	WorkOrderID string    `json:"work_order_id" gorm:"type:uuid;not null;index"`
	Quantity    float64   `json:"quantity" gorm:"type:decimal(12,4);not null"`
	ScrapQty    float64   `json:"scrap_qty" gorm:"type:decimal(12,4);default:0"`
	Notes       string    `json:"notes" gorm:"type:text"`
	ReportedBy  string    `json:"reported_by" gorm:"size:64;not null"`
	ReportedAt  time.Time `json:"reported_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (WorkOrderReport) TableName() string {
	return "erp_work_order_reports"
}
