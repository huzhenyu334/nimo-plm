package handler

import "github.com/bitfantasy/nimo-plm/internal/erp/service"

// Handlers ERP HTTP处理器集合
type Handlers struct {
	Supplier      *SupplierHandler
	Procurement   *ProcurementHandler
	Inventory     *InventoryHandler
	Manufacturing *ManufacturingHandler
	MRP           *MRPHandler
	Sales         *SalesHandler
}

func NewHandlers(services *service.Services) *Handlers {
	return &Handlers{
		Supplier:      NewSupplierHandler(services.Supplier),
		Procurement:   NewProcurementHandler(services.Procurement),
		Inventory:     NewInventoryHandler(services.Inventory),
		Manufacturing: NewManufacturingHandler(services.Manufacturing),
		MRP:           NewMRPHandler(services.MRP),
		Sales:         NewSalesHandler(services.Sales),
	}
}
