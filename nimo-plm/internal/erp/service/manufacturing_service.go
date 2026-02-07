package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	plmEntity "github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ManufacturingService struct {
	woRepo        *repository.WorkOrderRepository
	inventoryRepo *repository.InventoryRepository
	db            *gorm.DB // 直接读取PLM数据
}

func NewManufacturingService(woRepo *repository.WorkOrderRepository, invRepo *repository.InventoryRepository, db *gorm.DB) *ManufacturingService {
	return &ManufacturingService{woRepo: woRepo, inventoryRepo: invRepo, db: db}
}

type CreateWorkOrderRequest struct {
	ProductID   string  `json:"product_id" binding:"required"`
	BOMID       string  `json:"bom_id" binding:"required"`
	PlannedQty  float64 `json:"planned_qty" binding:"required,gt=0"`
	Priority    int     `json:"priority"`
	PlannedStart string `json:"planned_start"` // YYYY-MM-DD
	PlannedEnd   string `json:"planned_end"`
	WarehouseID  string `json:"warehouse_id"`
	Notes        string `json:"notes"`
}

func (s *ManufacturingService) Create(req CreateWorkOrderRequest, userID string) (*entity.WorkOrder, error) {
	// 查询PLM产品信息
	var product plmEntity.Product
	if err := s.db.Where("id = ?", req.ProductID).First(&product).Error; err != nil {
		return nil, fmt.Errorf("产品不存在: %w", err)
	}

	// 查询BOM
	var bomHeader plmEntity.BOMHeader
	if err := s.db.Where("id = ?", req.BOMID).First(&bomHeader).Error; err != nil {
		return nil, fmt.Errorf("BOM不存在: %w", err)
	}

	code := fmt.Sprintf("WO-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)

	wo := &entity.WorkOrder{
		ID:          uuid.New().String(),
		WOCode:      code,
		ProductID:   req.ProductID,
		ProductCode: product.Code,
		ProductName: product.Name,
		BOMID:       req.BOMID,
		BOMVersion:  bomHeader.Version,
		PlannedQty:  req.PlannedQty,
		Status:      entity.WOStatusCreated,
		Priority:    req.Priority,
		WarehouseID: req.WarehouseID,
		SourceType:  "MANUAL",
		Notes:       req.Notes,
		CreatedBy:   userID,
	}

	if req.PlannedStart != "" {
		t, err := time.Parse("2006-01-02", req.PlannedStart)
		if err == nil {
			wo.PlannedStart = &t
		}
	}
	if req.PlannedEnd != "" {
		t, err := time.Parse("2006-01-02", req.PlannedEnd)
		if err == nil {
			wo.PlannedEnd = &t
		}
	}

	// 根据BOM生成物料需求
	var bomItems []plmEntity.BOMItem
	if err := s.db.Where("bom_header_id = ?", req.BOMID).Find(&bomItems).Error; err != nil {
		return nil, fmt.Errorf("读取BOM明细失败: %w", err)
	}

	var materials []entity.WorkOrderMaterial
	for _, item := range bomItems {
		var mat plmEntity.Material
		s.db.Where("id = ?", item.MaterialID).First(&mat)

		materials = append(materials, entity.WorkOrderMaterial{
			ID:           uuid.New().String(),
			WorkOrderID:  wo.ID,
			MaterialID:   item.MaterialID,
			MaterialCode: mat.Code,
			MaterialName: mat.Name,
			RequiredQty:  item.Quantity * req.PlannedQty,
			Unit:         item.Unit,
		})
	}
	wo.Materials = materials

	if err := s.woRepo.Create(wo); err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}
	return wo, nil
}

func (s *ManufacturingService) GetByID(id string) (*entity.WorkOrder, error) {
	return s.woRepo.GetByID(id)
}

func (s *ManufacturingService) List(params repository.WOListParams) ([]entity.WorkOrder, int64, error) {
	return s.woRepo.List(params)
}

// Release 下达工单
func (s *ManufacturingService) Release(id string) error {
	wo, err := s.woRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}
	if wo.Status != entity.WOStatusCreated && wo.Status != entity.WOStatusPlanned {
		return fmt.Errorf("工单状态不允许下达: %s", wo.Status)
	}
	wo.Status = entity.WOStatusReleased
	return s.woRepo.Update(wo)
}

// Pick 领料 - 根据BOM计算需求，从库存出库
func (s *ManufacturingService) Pick(id, warehouseID, userID string) error {
	wo, err := s.woRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}
	if wo.Status != entity.WOStatusReleased && wo.Status != entity.WOStatusInProgress {
		return fmt.Errorf("工单状态不允许领料: %s", wo.Status)
	}

	// 遍历物料需求，执行出库
	for i := range wo.Materials {
		mat := &wo.Materials[i]
		needQty := mat.RequiredQty - mat.IssuedQty
		if needQty <= 0 {
			continue
		}

		// 查找库存
		inv, invErr := s.inventoryRepo.GetByMaterialAndWarehouse(mat.MaterialID, warehouseID)
		if invErr != nil {
			return fmt.Errorf("物料 %s 在仓库中无库存", mat.MaterialCode)
		}
		if inv.AvailableQty < needQty {
			return fmt.Errorf("物料 %s 库存不足: 需要%.4f, 可用%.4f", mat.MaterialCode, needQty, inv.AvailableQty)
		}

		// 扣减库存
		now := time.Now()
		inv.Quantity -= needQty
		inv.AvailableQty -= needQty
		inv.LastMovedAt = &now
		if err := s.inventoryRepo.Update(inv); err != nil {
			return fmt.Errorf("扣减库存失败: %w", err)
		}

		// 更新已发料数量
		mat.IssuedQty += needQty
		if err := s.woRepo.UpdateMaterial(mat); err != nil {
			return fmt.Errorf("更新物料发料数量失败: %w", err)
		}

		// 记录库存交易
		tx := &entity.InventoryTransaction{
			ID:              uuid.New().String(),
			MaterialID:      mat.MaterialID,
			MaterialCode:    mat.MaterialCode,
			MaterialName:    mat.MaterialName,
			WarehouseID:     warehouseID,
			TransactionType: entity.TxTypeProductionOut,
			Quantity:        -needQty,
			ReferenceType:   "WO",
			ReferenceID:     wo.ID,
			ReferenceCode:   wo.WOCode,
			CreatedBy:       userID,
		}
		s.inventoryRepo.CreateTransaction(tx)
	}

	if wo.Status == entity.WOStatusReleased {
		wo.Status = entity.WOStatusInProgress
		now := time.Now()
		wo.ActualStart = &now
	}
	return s.woRepo.Update(wo)
}

type ReportRequest struct {
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	ScrapQty float64 `json:"scrap_qty"`
	Notes    string  `json:"notes"`
}

// Report 报工
func (s *ManufacturingService) Report(id string, req ReportRequest, userID string) error {
	wo, err := s.woRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}
	if wo.Status != entity.WOStatusInProgress && wo.Status != entity.WOStatusReleased {
		return fmt.Errorf("工单状态不允许报工: %s", wo.Status)
	}

	now := time.Now()
	report := &entity.WorkOrderReport{
		ID:          uuid.New().String(),
		WorkOrderID: wo.ID,
		Quantity:    req.Quantity,
		ScrapQty:    req.ScrapQty,
		Notes:       req.Notes,
		ReportedBy:  userID,
		ReportedAt:  now,
	}
	if err := s.woRepo.CreateReport(report); err != nil {
		return fmt.Errorf("创建报工记录失败: %w", err)
	}

	wo.CompletedQty += req.Quantity
	wo.ScrapQty += req.ScrapQty
	if wo.Status == entity.WOStatusReleased {
		wo.Status = entity.WOStatusInProgress
		wo.ActualStart = &now
	}
	return s.woRepo.Update(wo)
}

// Complete 完工入库
func (s *ManufacturingService) Complete(id, warehouseID, userID string) error {
	wo, err := s.woRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("工单不存在: %w", err)
	}
	if wo.Status != entity.WOStatusInProgress {
		return fmt.Errorf("工单状态不允许完工: %s", wo.Status)
	}
	if wo.CompletedQty <= 0 {
		return fmt.Errorf("尚未报工，不能完工")
	}

	wID := warehouseID
	if wID == "" {
		wID = wo.WarehouseID
	}

	// 成品入库
	now := time.Now()
	batchNo := fmt.Sprintf("FG-%s%03d", now.Format("20060102"), now.UnixNano()%1000)

	existingInv, findErr := s.inventoryRepo.GetByMaterialAndWarehouse(wo.ProductID, wID)
	if findErr == nil && existingInv != nil {
		existingInv.Quantity += wo.CompletedQty
		existingInv.AvailableQty += wo.CompletedQty
		existingInv.LastMovedAt = &now
		s.inventoryRepo.Update(existingInv)
	} else {
		inv := &entity.Inventory{
			ID:            uuid.New().String(),
			MaterialID:    wo.ProductID,
			MaterialCode:  wo.ProductCode,
			MaterialName:  wo.ProductName,
			WarehouseID:   wID,
			BatchNo:       batchNo,
			InventoryType: entity.InventoryTypeFG,
			Quantity:      wo.CompletedQty,
			AvailableQty:  wo.CompletedQty,
			Unit:          "pcs",
			LastMovedAt:   &now,
		}
		s.inventoryRepo.UpsertInventory(inv)
	}

	// 库存交易记录
	tx := &entity.InventoryTransaction{
		ID:              uuid.New().String(),
		MaterialID:      wo.ProductID,
		MaterialCode:    wo.ProductCode,
		MaterialName:    wo.ProductName,
		WarehouseID:     wID,
		TransactionType: entity.TxTypeProductionIn,
		Quantity:        wo.CompletedQty,
		BatchNo:         batchNo,
		ReferenceType:   "WO",
		ReferenceID:     wo.ID,
		ReferenceCode:   wo.WOCode,
		CreatedBy:       userID,
	}
	s.inventoryRepo.CreateTransaction(tx)

	wo.Status = entity.WOStatusCompleted
	wo.ActualEnd = &now
	return s.woRepo.Update(wo)
}
