package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/google/uuid"
)

type InventoryService struct {
	repo *repository.InventoryRepository
}

func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) List(params repository.InventoryListParams) ([]entity.Inventory, int64, error) {
	return s.repo.List(params)
}

func (s *InventoryService) GetByMaterial(materialID string) ([]entity.Inventory, error) {
	return s.repo.GetByMaterial(materialID)
}

func (s *InventoryService) GetAlerts() ([]entity.Inventory, error) {
	return s.repo.GetAlerts()
}

func (s *InventoryService) ListTransactions(materialID string, page, size int) ([]entity.InventoryTransaction, int64, error) {
	return s.repo.ListTransactions(materialID, page, size)
}

type InboundRequest struct {
	MaterialID    string  `json:"material_id" binding:"required"`
	MaterialCode  string  `json:"material_code"`
	MaterialName  string  `json:"material_name"`
	WarehouseID   string  `json:"warehouse_id" binding:"required"`
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	BatchNo       string  `json:"batch_no"`
	UnitCost      float64 `json:"unit_cost"`
	Unit          string  `json:"unit"`
	InventoryType string  `json:"inventory_type"`
	ReferenceType string  `json:"reference_type" binding:"required"` // PO, WO, RETURN
	ReferenceID   string  `json:"reference_id" binding:"required"`
	ReferenceCode string  `json:"reference_code"`
	Notes         string  `json:"notes"`
}

func (s *InventoryService) Inbound(req InboundRequest, userID string) error {
	now := time.Now()
	batchNo := req.BatchNo
	if batchNo == "" {
		batchNo = fmt.Sprintf("%s%03d", now.Format("20060102"), now.UnixNano()%1000)
	}
	unit := req.Unit
	if unit == "" {
		unit = "pcs"
	}
	invType := req.InventoryType
	if invType == "" {
		invType = entity.InventoryTypeRaw
	}

	// 更新库存
	existingInv, err := s.repo.GetByMaterialAndWarehouse(req.MaterialID, req.WarehouseID)
	if err == nil && existingInv != nil {
		existingInv.Quantity += req.Quantity
		existingInv.AvailableQty += req.Quantity
		existingInv.LastMovedAt = &now
		if err := s.repo.Update(existingInv); err != nil {
			return fmt.Errorf("更新库存失败: %w", err)
		}
	} else {
		inv := &entity.Inventory{
			ID:            uuid.New().String(),
			MaterialID:    req.MaterialID,
			MaterialCode:  req.MaterialCode,
			MaterialName:  req.MaterialName,
			WarehouseID:   req.WarehouseID,
			BatchNo:       batchNo,
			InventoryType: invType,
			Quantity:      req.Quantity,
			AvailableQty:  req.Quantity,
			UnitCost:      req.UnitCost,
			Unit:          unit,
			LastMovedAt:   &now,
		}
		if err := s.repo.UpsertInventory(inv); err != nil {
			return fmt.Errorf("创建库存失败: %w", err)
		}
	}

	// 交易记录
	txType := entity.TxTypePurchaseIn
	switch req.ReferenceType {
	case "WO":
		txType = entity.TxTypeProductionIn
	case "RETURN":
		txType = entity.TxTypeReturnIn
	}

	tx := &entity.InventoryTransaction{
		ID:              uuid.New().String(),
		MaterialID:      req.MaterialID,
		MaterialCode:    req.MaterialCode,
		MaterialName:    req.MaterialName,
		WarehouseID:     req.WarehouseID,
		TransactionType: txType,
		Quantity:        req.Quantity,
		BatchNo:         batchNo,
		UnitCost:        req.UnitCost,
		ReferenceType:   req.ReferenceType,
		ReferenceID:     req.ReferenceID,
		ReferenceCode:   req.ReferenceCode,
		Notes:           req.Notes,
		CreatedBy:       userID,
	}
	return s.repo.CreateTransaction(tx)
}

type OutboundRequest struct {
	MaterialID    string  `json:"material_id" binding:"required"`
	MaterialCode  string  `json:"material_code"`
	MaterialName  string  `json:"material_name"`
	WarehouseID   string  `json:"warehouse_id" binding:"required"`
	Quantity      float64 `json:"quantity" binding:"required,gt=0"`
	ReferenceType string  `json:"reference_type" binding:"required"` // WO, SO, SCRAP
	ReferenceID   string  `json:"reference_id" binding:"required"`
	ReferenceCode string  `json:"reference_code"`
	Notes         string  `json:"notes"`
}

func (s *InventoryService) Outbound(req OutboundRequest, userID string) error {
	now := time.Now()

	// 查找库存
	inv, err := s.repo.GetByMaterialAndWarehouse(req.MaterialID, req.WarehouseID)
	if err != nil {
		return fmt.Errorf("库存记录不存在: %w", err)
	}
	if inv.AvailableQty < req.Quantity {
		return fmt.Errorf("可用库存不足: 需要%.4f, 可用%.4f", req.Quantity, inv.AvailableQty)
	}

	inv.Quantity -= req.Quantity
	inv.AvailableQty -= req.Quantity
	inv.LastMovedAt = &now
	if err := s.repo.Update(inv); err != nil {
		return fmt.Errorf("更新库存失败: %w", err)
	}

	txType := entity.TxTypeSalesOut
	switch req.ReferenceType {
	case "WO":
		txType = entity.TxTypeProductionOut
	case "SCRAP":
		txType = entity.TxTypeScrapOut
	}

	tx := &entity.InventoryTransaction{
		ID:              uuid.New().String(),
		MaterialID:      req.MaterialID,
		MaterialCode:    req.MaterialCode,
		MaterialName:    req.MaterialName,
		WarehouseID:     req.WarehouseID,
		TransactionType: txType,
		Quantity:        -req.Quantity, // 负数表示出库
		ReferenceType:   req.ReferenceType,
		ReferenceID:     req.ReferenceID,
		ReferenceCode:   req.ReferenceCode,
		Notes:           req.Notes,
		CreatedBy:       userID,
	}
	return s.repo.CreateTransaction(tx)
}

type AdjustRequest struct {
	MaterialID    string  `json:"material_id" binding:"required"`
	WarehouseID   string  `json:"warehouse_id" binding:"required"`
	AdjustQty     float64 `json:"adjust_qty" binding:"required"` // 正数增加，负数减少
	Reason        string  `json:"reason" binding:"required"`
}

func (s *InventoryService) Adjust(req AdjustRequest, userID string) error {
	now := time.Now()

	inv, err := s.repo.GetByMaterialAndWarehouse(req.MaterialID, req.WarehouseID)
	if err != nil {
		return fmt.Errorf("库存记录不存在: %w", err)
	}

	inv.Quantity += req.AdjustQty
	inv.AvailableQty += req.AdjustQty
	inv.LastMovedAt = &now
	if inv.AvailableQty < 0 {
		return fmt.Errorf("调整后可用库存不能为负数")
	}
	if err := s.repo.Update(inv); err != nil {
		return fmt.Errorf("更新库存失败: %w", err)
	}

	tx := &entity.InventoryTransaction{
		ID:              uuid.New().String(),
		MaterialID:      req.MaterialID,
		MaterialCode:    inv.MaterialCode,
		MaterialName:    inv.MaterialName,
		WarehouseID:     req.WarehouseID,
		TransactionType: entity.TxTypeAdjust,
		Quantity:        req.AdjustQty,
		ReferenceType:   "ADJUST",
		ReferenceID:     uuid.New().String(),
		Notes:           req.Reason,
		CreatedBy:       userID,
	}
	return s.repo.CreateTransaction(tx)
}
