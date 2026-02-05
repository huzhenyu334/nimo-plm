package entity

import (
	"time"
)

// BOMHeader BOM头表
type BOMHeader struct {
	ID           string     `json:"id" gorm:"primaryKey;size:32"`
	ProductID    string     `json:"product_id" gorm:"size:32;not null"`
	Version      string     `json:"version" gorm:"size:16;not null"`
	Status       string     `json:"status" gorm:"size:16;not null;default:draft"`
	Description  string     `json:"description" gorm:"type:text"`
	TotalItems   int        `json:"total_items" gorm:"not null;default:0"`
	TotalCost    float64    `json:"total_cost" gorm:"type:decimal(15,4)"`
	MaxLevel     int        `json:"max_level" gorm:"not null;default:0"`
	ReleasedBy   string     `json:"released_by" gorm:"size:32"`
	ReleasedAt   *time.Time `json:"released_at"`
	ReleaseNotes string     `json:"release_notes" gorm:"type:text"`
	CreatedBy    string     `json:"created_by" gorm:"size:32;not null"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联
	Product  *Product   `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	Creator  *User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Releaser *User      `json:"releaser,omitempty" gorm:"foreignKey:ReleasedBy"`
	Items    []BOMItem  `json:"items,omitempty" gorm:"foreignKey:BOMHeaderID"`
}

func (BOMHeader) TableName() string {
	return "bom_headers"
}

// BOMItem BOM行项
type BOMItem struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	BOMHeaderID  string    `json:"bom_header_id" gorm:"size:32;not null"`
	ParentItemID string    `json:"parent_item_id" gorm:"size:32"`
	MaterialID   string    `json:"material_id" gorm:"size:32;not null"`
	Level        int       `json:"level" gorm:"not null;default:0"`
	Sequence     int       `json:"sequence" gorm:"not null;default:0"`
	Quantity     float64   `json:"quantity" gorm:"type:decimal(15,4);not null"`
	Unit         string    `json:"unit" gorm:"size:16;not null;default:pcs"`
	Position     string    `json:"position" gorm:"size:32"`
	Reference    string    `json:"reference" gorm:"size:128"`
	Notes        string    `json:"notes" gorm:"type:text"`
	UnitCost     float64   `json:"unit_cost" gorm:"type:decimal(15,4)"`
	ExtendedCost float64   `json:"extended_cost" gorm:"type:decimal(15,4)"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	BOMHeader  *BOMHeader `json:"bom_header,omitempty" gorm:"foreignKey:BOMHeaderID"`
	ParentItem *BOMItem   `json:"parent_item,omitempty" gorm:"foreignKey:ParentItemID"`
	Material   *Material  `json:"material,omitempty" gorm:"foreignKey:MaterialID"`
	Children   []BOMItem  `json:"children,omitempty" gorm:"foreignKey:ParentItemID"`
}

func (BOMItem) TableName() string {
	return "bom_items"
}

// BOMStatus BOM状态
const (
	BOMStatusDraft    = "draft"
	BOMStatusReleased = "released"
	BOMStatusObsolete = "obsolete"
)
