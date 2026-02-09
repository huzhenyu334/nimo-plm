package entity

import "time"

// ActivityLog SRM操作日志
type ActivityLog struct {
	ID         string `json:"id" gorm:"primaryKey;size:32"`
	EntityType string `json:"entity_type" gorm:"size:50;not null;index:idx_activity_entity"` // project/pr/po/inspection/supplier/8d
	EntityID   string `json:"entity_id" gorm:"size:32;not null;index:idx_activity_entity"`
	EntityCode string `json:"entity_code" gorm:"size:50"`

	Action     string `json:"action" gorm:"size:50;not null"` // create/status_change/assign_supplier/receive/comment等
	FromStatus string `json:"from_status" gorm:"size:20"`
	ToStatus   string `json:"to_status" gorm:"size:20"`

	Content     string `json:"content" gorm:"type:text"`
	Attachments JSONB  `json:"attachments" gorm:"type:jsonb"` // [{name, url, size}]
	Metadata    JSONB  `json:"metadata" gorm:"type:jsonb"`

	OperatorID   string    `json:"operator_id" gorm:"size:32"`
	OperatorName string    `json:"operator_name" gorm:"size:100"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ActivityLog) TableName() string {
	return "srm_activity_logs"
}
