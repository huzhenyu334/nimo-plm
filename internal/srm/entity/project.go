package entity

import "time"

// SRMProject 采购项目
type SRMProject struct {
	ID   string `json:"id" gorm:"primaryKey;size:32"`
	Code string `json:"code" gorm:"size:32;uniqueIndex;not null"` // SRMP-2026-0001
	Name string `json:"name" gorm:"size:200;not null"`
	Type string `json:"type" gorm:"size:20;not null"` // sample/production
	Phase  string `json:"phase" gorm:"size:20"`          // EVT/DVT/PVT/MP
	Status string `json:"status" gorm:"size:20;default:active"` // active/completed/cancelled

	// PLM桥接
	PLMProjectID *string `json:"plm_project_id" gorm:"size:32"`
	PLMTaskID    *string `json:"plm_task_id" gorm:"size:32"`
	PLMBOMID     *string `json:"plm_bom_id" gorm:"size:32"`

	// 进度统计
	TotalItems    int `json:"total_items" gorm:"default:0"`
	SourcingCount int `json:"sourcing_count" gorm:"default:0"`
	OrderedCount  int `json:"ordered_count" gorm:"default:0"`
	ReceivedCount int `json:"received_count" gorm:"default:0"`
	PassedCount   int `json:"passed_count" gorm:"default:0"`
	FailedCount   int `json:"failed_count" gorm:"default:0"`

	// 交期
	EstimatedDays *int       `json:"estimated_days"`
	StartDate     *time.Time `json:"start_date" gorm:"type:date"`
	TargetDate    *time.Time `json:"target_date" gorm:"type:date"`
	ActualDate    *time.Time `json:"actual_date" gorm:"type:date"`

	CreatedBy string    `json:"created_by" gorm:"size:32"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SRMProject) TableName() string {
	return "srm_projects"
}

// SRMProject 状态
const (
	SRMProjectStatusActive    = "active"
	SRMProjectStatusCompleted = "completed"
	SRMProjectStatusCancelled = "cancelled"
)

// SRMProject 类型
const (
	SRMProjectTypeSample     = "sample"
	SRMProjectTypeProduction = "production"
)
