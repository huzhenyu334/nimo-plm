package service

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"gorm.io/gorm"
)

// Services ERP 服务集合
type Services struct {
	Supplier      *SupplierService
	Procurement   *ProcurementService
	Inventory     *InventoryService
	Manufacturing *ManufacturingService
	MRP           *MRPService
	Sales         *SalesService
}

func NewServices(repos *repository.Repositories, db *gorm.DB) *Services {
	return &Services{
		Supplier:      NewSupplierService(repos.Supplier),
		Procurement:   NewProcurementService(repos.Purchase, repos.Supplier, repos.Inventory),
		Inventory:     NewInventoryService(repos.Inventory),
		Manufacturing: NewManufacturingService(repos.WorkOrder, repos.Inventory, db),
		MRP:           NewMRPService(repos.MRP, repos.Purchase, repos.Inventory, repos.WorkOrder, repos.Sales, db),
		Sales:         NewSalesService(repos.Sales, repos.Inventory),
	}
}
