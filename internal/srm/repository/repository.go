package repository

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("record not found")
)

// Repositories SRM仓库集合
type Repositories struct {
	Supplier   *SupplierRepository
	PR         *PRRepository
	PO         *PORepository
	Inspection *InspectionRepository
}

// NewRepositories 创建SRM仓库集合
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Supplier:   NewSupplierRepository(db),
		PR:         NewPRRepository(db),
		PO:         NewPORepository(db),
		Inspection: NewInspectionRepository(db),
	}
}
