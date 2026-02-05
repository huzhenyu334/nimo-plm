package entity

import (
	"time"
)

// MaterialCategory 物料类别
type MaterialCategory struct {
	ID        string    `json:"id" gorm:"primaryKey;size:32"`
	Code      string    `json:"code" gorm:"size:32;not null;uniqueIndex"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	ParentID  string    `json:"parent_id" gorm:"size:32"`
	Path      string    `json:"path" gorm:"size:256"`
	Level     int       `json:"level" gorm:"not null;default:1"`
	SortOrder int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Parent   *MaterialCategory  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []MaterialCategory `json:"children,omitempty" gorm:"foreignKey:ParentID"`
}

func (MaterialCategory) TableName() string {
	return "material_categories"
}

// Material 物料实体
type Material struct {
	ID           string     `json:"id" gorm:"primaryKey;size:32"`
	Code         string     `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Name         string     `json:"name" gorm:"size:128;not null"`
	CategoryID   string     `json:"category_id" gorm:"size:32;not null"`
	Status       string     `json:"status" gorm:"size:16;not null;default:active"`
	Unit         string     `json:"unit" gorm:"size:16;not null;default:pcs"`
	Description  string     `json:"description" gorm:"type:text"`
	Specs        JSONB      `json:"specs" gorm:"type:jsonb"`
	LeadTimeDays int        `json:"lead_time_days"`
	MinOrderQty  float64    `json:"min_order_qty" gorm:"type:decimal(15,4)"`
	SafetyStock  float64    `json:"safety_stock" gorm:"type:decimal(15,4)"`
	StandardCost float64    `json:"standard_cost" gorm:"type:decimal(15,4)"`
	LastCost     float64    `json:"last_cost" gorm:"type:decimal(15,4)"`
	Currency     string     `json:"currency" gorm:"size:3;default:CNY"`
	CreatedBy    string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Category *MaterialCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Creator  *User             `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

func (Material) TableName() string {
	return "materials"
}

// MaterialStatus 物料状态
const (
	MaterialStatusActive   = "active"
	MaterialStatusInactive = "inactive"
	MaterialStatusObsolete = "obsolete"
)

// MaterialUnit 物料单位
const (
	MaterialUnitPCS = "pcs"
	MaterialUnitKG  = "kg"
	MaterialUnitM   = "m"
	MaterialUnitSet = "set"
)
