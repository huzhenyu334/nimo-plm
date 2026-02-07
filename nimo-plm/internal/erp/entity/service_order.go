package entity

import (
	"time"
)

// ServiceType 服务类型
const (
	ServiceTypeRepair   = "REPAIR"   // 维修
	ServiceTypeReturn   = "RETURN"   // 退货
	ServiceTypeExchange = "EXCHANGE" // 换货
	ServiceTypeInquiry  = "INQUIRY"  // 咨询
)

// ServiceOrderStatus 服务工单状态
const (
	SvcStatusCreated      = "CREATED"
	SvcStatusAssigned     = "ASSIGNED"
	SvcStatusInProgress   = "IN_PROGRESS"
	SvcStatusWaitingParts = "WAITING_PARTS"
	SvcStatusCompleted    = "COMPLETED"
	SvcStatusClosed       = "CLOSED"
)

// ServiceOrder 服务工单
type ServiceOrder struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ServiceCode  string     `json:"service_code" gorm:"size:50;not null;uniqueIndex"`
	CustomerID   string     `json:"customer_id" gorm:"type:uuid;not null;index"`
	ProductSN    string     `json:"product_sn" gorm:"size:100;not null"`
	ServiceType  string     `json:"service_type" gorm:"size:20;not null"`
	Status       string     `json:"status" gorm:"size:20;not null;default:CREATED"`
	Priority     int        `json:"priority" gorm:"default:0"`
	Description  string     `json:"description" gorm:"type:text;not null"`
	Solution     string     `json:"solution" gorm:"type:text"`
	AssigneeID   string     `json:"assignee_id" gorm:"size:64"`
	AssigneeName string     `json:"assignee_name" gorm:"size:100"`
	Notes        string     `json:"notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:64;not null"`
	CompletedAt  *time.Time `json:"completed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`

	Customer *Customer `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
}

func (ServiceOrder) TableName() string {
	return "erp_service_orders"
}
