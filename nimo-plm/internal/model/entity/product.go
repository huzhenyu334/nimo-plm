package entity

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONB 用于PostgreSQL JSONB类型
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// ProductCategory 产品类别
type ProductCategory struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	Code        string    `json:"code" gorm:"size:32;not null;uniqueIndex"`
	Name        string    `json:"name" gorm:"size:64;not null"`
	ParentID    string    `json:"parent_id" gorm:"size:32"`
	Path        string    `json:"path" gorm:"size:256"`
	Level       int       `json:"level" gorm:"not null;default:1"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联
	Parent   *ProductCategory  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []ProductCategory `json:"children,omitempty" gorm:"foreignKey:ParentID"`
}

func (ProductCategory) TableName() string {
	return "product_categories"
}

// Product 产品实体
type Product struct {
	ID                string     `json:"id" gorm:"primaryKey;size:32"`
	Code              string     `json:"code" gorm:"size:64;not null;uniqueIndex"`
	Name              string     `json:"name" gorm:"size:128;not null"`
	CategoryID        string     `json:"category_id" gorm:"size:32;not null"`
	Status            string     `json:"status" gorm:"size:16;not null;default:draft"`
	Description       string     `json:"description" gorm:"type:text"`
	Specs             JSONB      `json:"specs" gorm:"type:jsonb"`
	ThumbnailURL      string     `json:"thumbnail_url" gorm:"size:512"`
	CurrentBOMVersion string     `json:"current_bom_version" gorm:"size:16"`
	CreatedBy         string     `json:"created_by" gorm:"size:32;not null"`
	UpdatedBy         string     `json:"updated_by" gorm:"size:32"`
	ReleasedAt        *time.Time `json:"released_at"`
	DiscontinuedAt    *time.Time `json:"discontinued_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at" gorm:"index"`

	// 关联
	Category   *ProductCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Creator    *User            `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	BOMHeaders []BOMHeader      `json:"bom_headers,omitempty" gorm:"foreignKey:ProductID"`
	Projects   []Project        `json:"projects,omitempty" gorm:"foreignKey:ProductID"`
}

func (Product) TableName() string {
	return "products"
}

// ProductStatus 产品状态
const (
	ProductStatusDraft        = "draft"
	ProductStatusDeveloping   = "developing"
	ProductStatusActive       = "active"
	ProductStatusDiscontinued = "discontinued"
)
