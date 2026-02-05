package entity

import (
	"time"
)

// ECN 工程变更通知
type ECN struct {
	ID                   string     `json:"id" gorm:"primaryKey;size:32"`
	Code                 string     `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Title                string     `json:"title" gorm:"size:256;not null"`
	ProductID            string     `json:"product_id" gorm:"size:32;not null"`
	ChangeType           string     `json:"change_type" gorm:"size:32;not null"`
	Urgency              string     `json:"urgency" gorm:"size:16;not null;default:medium"`
	Status               string     `json:"status" gorm:"size:16;not null;default:draft"`
	Reason               string     `json:"reason" gorm:"type:text;not null"`
	Description          string     `json:"description" gorm:"type:text"`
	ImpactAnalysis       string     `json:"impact_analysis" gorm:"type:text"`
	RequestedBy          string     `json:"requested_by" gorm:"size:32;not null"`
	RequestedAt          *time.Time `json:"requested_at"`
	ApprovedBy           string     `json:"approved_by" gorm:"size:32"`
	ApprovedAt           *time.Time `json:"approved_at"`
	RejectionReason      string     `json:"rejection_reason" gorm:"type:text"`
	ImplementedBy        string     `json:"implemented_by" gorm:"size:32"`
	ImplementedAt        *time.Time `json:"implemented_at"`
	FeishuApprovalCode   string     `json:"feishu_approval_code" gorm:"size:64"`
	FeishuInstanceCode   string     `json:"feishu_instance_code" gorm:"size:64"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// 关联
	Product       *Product           `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	Requester     *User              `json:"requester,omitempty" gorm:"foreignKey:RequestedBy"`
	Approver      *User              `json:"approver,omitempty" gorm:"foreignKey:ApprovedBy"`
	Implementer   *User              `json:"implementer,omitempty" gorm:"foreignKey:ImplementedBy"`
	AffectedItems []ECNAffectedItem  `json:"affected_items,omitempty" gorm:"foreignKey:ECNID"`
	Approvals     []ECNApproval      `json:"approvals,omitempty" gorm:"foreignKey:ECNID"`
}

func (ECN) TableName() string {
	return "ecns"
}

// ECNAffectedItem ECN受影响项目
type ECNAffectedItem struct {
	ID                string    `json:"id" gorm:"primaryKey;size:32"`
	ECNID             string    `json:"ecn_id" gorm:"size:32;not null"`
	ItemType          string    `json:"item_type" gorm:"size:32;not null"`
	ItemID            string    `json:"item_id" gorm:"size:32;not null"`
	BeforeValue       JSONB     `json:"before_value" gorm:"type:jsonb"`
	AfterValue        JSONB     `json:"after_value" gorm:"type:jsonb"`
	ChangeDescription string    `json:"change_description" gorm:"type:text"`
	CreatedAt         time.Time `json:"created_at"`

	// 关联
	ECN *ECN `json:"ecn,omitempty" gorm:"foreignKey:ECNID"`
}

func (ECNAffectedItem) TableName() string {
	return "ecn_affected_items"
}

// ECNApproval ECN审批记录
type ECNApproval struct {
	ID         string     `json:"id" gorm:"primaryKey;size:32"`
	ECNID      string     `json:"ecn_id" gorm:"size:32;not null"`
	ApproverID string     `json:"approver_id" gorm:"size:32;not null"`
	Sequence   int        `json:"sequence" gorm:"not null"`
	Status     string     `json:"status" gorm:"size:16;not null;default:pending"`
	Decision   string     `json:"decision" gorm:"size:16"`
	Comment    string     `json:"comment" gorm:"type:text"`
	DecidedAt  *time.Time `json:"decided_at"`
	CreatedAt  time.Time  `json:"created_at"`

	// 关联
	ECN      *ECN  `json:"ecn,omitempty" gorm:"foreignKey:ECNID"`
	Approver *User `json:"approver,omitempty" gorm:"foreignKey:ApproverID"`
}

func (ECNApproval) TableName() string {
	return "ecn_approvals"
}

// ECN状态常量
const (
	ECNStatusDraft       = "draft"
	ECNStatusPending     = "pending"
	ECNStatusApproved    = "approved"
	ECNStatusRejected    = "rejected"
	ECNStatusImplemented = "implemented"
	ECNStatusCancelled   = "cancelled"
)

// ECN变更类型
const (
	ECNChangeTypeDesign   = "design"
	ECNChangeTypeMaterial = "material"
	ECNChangeTypeProcess  = "process"
	ECNChangeTypeDocument = "document"
)

// ECN紧急程度
const (
	ECNUrgencyLow      = "low"
	ECNUrgencyMedium   = "medium"
	ECNUrgencyHigh     = "high"
	ECNUrgencyCritical = "critical"
)

// ECN受影响项目类型
const (
	ECNAffectedTypeBOMItem   = "bom_item"
	ECNAffectedTypeMaterial  = "material"
	ECNAffectedTypeDocument  = "document"
	ECNAffectedTypeDrawing   = "drawing"
)

// ECN审批决策
const (
	ECNApprovalDecisionApprove = "approve"
	ECNApprovalDecisionReject  = "reject"
)
