package entity

import "time"

// PurchaseRequest 采购需求单
type PurchaseRequest struct {
	ID           string     `json:"id" gorm:"primaryKey;size:32"`
	PRCode       string     `json:"pr_code" gorm:"size:32;uniqueIndex;not null"`
	Title        string     `json:"title" gorm:"size:200;not null"`
	Type         string     `json:"type" gorm:"size:20;not null"`     // sample/production
	Priority     string     `json:"priority" gorm:"size:20;default:normal"` // urgent/high/normal/low
	Status       string     `json:"status" gorm:"size:20;default:draft"`    // draft/pending/approved/sourcing/completed/cancelled

	// 关联
	ProjectID    *string `json:"project_id" gorm:"size:32"`       // PLM项目ID
	SRMProjectID *string `json:"srm_project_id" gorm:"size:32"`   // SRM采购项目ID
	BOMID        *string `json:"bom_id" gorm:"size:32"`
	Phase        string  `json:"phase" gorm:"size:20"` // EVT/DVT/PVT/MP

	// 需求信息
	RequiredDate *time.Time `json:"required_date"`

	// 管理
	RequestedBy string     `json:"requested_by" gorm:"size:32"`
	ApprovedBy  *string    `json:"approved_by" gorm:"size:32"`
	ApprovedAt  *time.Time `json:"approved_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Notes       string     `json:"notes" gorm:"type:text"`

	// 关联
	Items []PRItem `json:"items,omitempty" gorm:"foreignKey:PRID"`
}

func (PurchaseRequest) TableName() string {
	return "srm_purchase_requests"
}

// PR状态
const (
	PRStatusDraft     = "draft"
	PRStatusPending   = "pending"
	PRStatusApproved  = "approved"
	PRStatusSourcing  = "sourcing"
	PRStatusCompleted = "completed"
	PRStatusCancelled = "cancelled"
)

// PR类型
const (
	PRTypeSample     = "sample"
	PRTypeProduction = "production"
)

// PRItem 采购需求行项
type PRItem struct {
	ID            string   `json:"id" gorm:"primaryKey;size:32"`
	PRID          string   `json:"pr_id" gorm:"size:32;not null;index"`

	// 物料信息
	MaterialID    *string  `json:"material_id" gorm:"size:32"`
	MaterialCode  string   `json:"material_code" gorm:"size:50"`
	MaterialName  string   `json:"material_name" gorm:"size:200;not null"`
	Specification string   `json:"specification" gorm:"size:500"`
	Category      string   `json:"category" gorm:"size:100"`

	// 需求数量
	Quantity float64 `json:"quantity" gorm:"type:decimal(10,2);not null"`
	Unit     string  `json:"unit" gorm:"size:20;default:pcs"`

	// 采购进度
	Status      string   `json:"status" gorm:"size:20;default:pending"` // pending/sourcing/ordered/received/inspected/completed
	SupplierID  *string  `json:"supplier_id" gorm:"size:32"`
	UnitPrice   *float64 `json:"unit_price" gorm:"type:decimal(12,4)"`
	TotalAmount *float64 `json:"total_amount" gorm:"type:decimal(15,2)"`

	// 交期
	ExpectedDate *time.Time `json:"expected_date"`
	ActualDate   *time.Time `json:"actual_date"`

	// 检验
	InspectionResult string `json:"inspection_result" gorm:"size:20"` // passed/failed/conditional

	Round     int       `json:"round" gorm:"default:1"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	Notes     string    `json:"notes" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PRItem) TableName() string {
	return "srm_pr_items"
}

// PR行项状态
const (
	PRItemStatusPending   = "pending"
	PRItemStatusSourcing  = "sourcing"
	PRItemStatusOrdered   = "ordered"
	PRItemStatusReceived  = "received"
	PRItemStatusInspected = "inspected"
	PRItemStatusCompleted = "completed"
)
