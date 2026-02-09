package entity

import "time"

// DelayRequest 延期审批单
type DelayRequest struct {
	ID            string `json:"id" gorm:"primaryKey;size:32"`
	Code          string `json:"code" gorm:"size:32;uniqueIndex;not null"` // DLY-2026-0001
	SRMProjectID  string `json:"srm_project_id" gorm:"size:32;not null;index"`
	PRItemID      string `json:"pr_item_id" gorm:"size:32"`
	MaterialName  string `json:"material_name" gorm:"size:200"`
	OriginalDays  int    `json:"original_days"`
	RequestedDays int    `json:"requested_days"`
	Reason        string `json:"reason" gorm:"type:text"`
	ReasonType    string `json:"reason_type" gorm:"size:50"` // supplier_capacity/design_change/quality_issue/other

	Status      string     `json:"status" gorm:"size:20;default:pending"` // pending/approved/rejected
	RequestedBy string     `json:"requested_by" gorm:"size:32"`
	ApprovedBy  *string    `json:"approved_by" gorm:"size:32"`
	ApprovedAt  *time.Time `json:"approved_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (DelayRequest) TableName() string {
	return "srm_delay_requests"
}

// DelayRequest 状态
const (
	DelayRequestStatusPending  = "pending"
	DelayRequestStatusApproved = "approved"
	DelayRequestStatusRejected = "rejected"
)
