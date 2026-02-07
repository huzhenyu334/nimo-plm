package repository

import "gorm.io/gorm"

// Repositories ERP 仓库集合
type Repositories struct {
	Supplier  *SupplierRepository
	Purchase  *PurchaseRepository
	Inventory *InventoryRepository
	WorkOrder *WorkOrderRepository
	Sales     *SalesRepository
	MRP       *MRPRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Supplier:  NewSupplierRepository(db),
		Purchase:  NewPurchaseRepository(db),
		Inventory: NewInventoryRepository(db),
		WorkOrder: NewWorkOrderRepository(db),
		Sales:     NewSalesRepository(db),
		MRP:       NewMRPRepository(db),
	}
}
