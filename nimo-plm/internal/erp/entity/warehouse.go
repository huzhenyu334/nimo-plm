package entity

import (
	"time"
)

// WarehouseStatus 仓库状态
const (
	WarehouseStatusActive   = "ACTIVE"
	WarehouseStatusInactive = "INACTIVE"
)

// Warehouse 仓库
type Warehouse struct {
	ID        string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Code      string     `json:"code" gorm:"size:50;not null;uniqueIndex"`
	Name      string     `json:"name" gorm:"size:100;not null"`
	Address   string     `json:"address" gorm:"size:500"`
	Manager   string     `json:"manager" gorm:"size:64"` // 负责人飞书ID
	Status    string     `json:"status" gorm:"size:20;not null;default:ACTIVE"`
	Notes     string     `json:"notes" gorm:"type:text"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	Zones []WarehouseZone `json:"zones,omitempty" gorm:"foreignKey:WarehouseID"`
}

func (Warehouse) TableName() string {
	return "erp_warehouses"
}

// WarehouseZone 库区
type WarehouseZone struct {
	ID          string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	WarehouseID string     `json:"warehouse_id" gorm:"type:uuid;not null;index"`
	Code        string     `json:"code" gorm:"size:50;not null"`
	Name        string     `json:"name" gorm:"size:100;not null"`
	ZoneType    string     `json:"zone_type" gorm:"size:20"` // RAW, WIP, FG, SPARE
	Status      string     `json:"status" gorm:"size:20;not null;default:ACTIVE"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"index"`

	Warehouse *Warehouse         `json:"warehouse,omitempty" gorm:"foreignKey:WarehouseID"`
	Locations []WarehouseLocation `json:"locations,omitempty" gorm:"foreignKey:ZoneID"`
}

func (WarehouseZone) TableName() string {
	return "erp_warehouse_zones"
}

// WarehouseLocation 库位
type WarehouseLocation struct {
	ID        string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ZoneID    string     `json:"zone_id" gorm:"type:uuid;not null;index"`
	Code      string     `json:"code" gorm:"size:50;not null"`
	Name      string     `json:"name" gorm:"size:100"`
	Capacity  float64    `json:"capacity" gorm:"type:decimal(12,4);default:0"`
	Status    string     `json:"status" gorm:"size:20;not null;default:ACTIVE"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	Zone *WarehouseZone `json:"zone,omitempty" gorm:"foreignKey:ZoneID"`
}

func (WarehouseLocation) TableName() string {
	return "erp_warehouse_locations"
}
