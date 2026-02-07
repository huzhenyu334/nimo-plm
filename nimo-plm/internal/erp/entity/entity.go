package entity

import "gorm.io/gorm"

// AutoMigrate 自动迁移所有ERP表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// 基础数据
		&Warehouse{},
		&WarehouseZone{},
		&WarehouseLocation{},
		&Supplier{},
		&Customer{},

		// 采购
		&PurchaseRequisition{},
		&PurchaseOrder{},
		&POItem{},

		// 库存
		&Inventory{},
		&InventoryTransaction{},

		// 生产
		&WorkOrder{},
		&WorkOrderMaterial{},
		&WorkOrderReport{},

		// 销售
		&SalesOrder{},
		&SOItem{},

		// 售后
		&ServiceOrder{},

		// MRP
		&MRPRun{},
		&MRPResult{},

		// 财务
		&FinanceRecord{},
	)
}
