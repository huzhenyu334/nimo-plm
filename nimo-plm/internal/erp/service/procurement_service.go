package service

import (
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/google/uuid"
)

type ProcurementService struct {
	purchaseRepo  *repository.PurchaseRepository
	supplierRepo  *repository.SupplierRepository
	inventoryRepo *repository.InventoryRepository
}

func NewProcurementService(pr *repository.PurchaseRepository, sr *repository.SupplierRepository, ir *repository.InventoryRepository) *ProcurementService {
	return &ProcurementService{purchaseRepo: pr, supplierRepo: sr, inventoryRepo: ir}
}

// --- Purchase Requisition ---

type CreatePRRequest struct {
	MaterialID   string  `json:"material_id" binding:"required"`
	MaterialCode string  `json:"material_code"`
	MaterialName string  `json:"material_name"`
	Quantity     float64 `json:"quantity" binding:"required,gt=0"`
	Unit         string  `json:"unit"`
	RequiredDate string  `json:"required_date"` // YYYY-MM-DD
	Notes        string  `json:"notes"`
}

func (s *ProcurementService) CreatePR(req CreatePRRequest, userID string) (*entity.PurchaseRequisition, error) {
	code := fmt.Sprintf("PR-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
	unit := req.Unit
	if unit == "" {
		unit = "pcs"
	}

	pr := &entity.PurchaseRequisition{
		ID:           uuid.New().String(),
		PRCode:       code,
		MaterialID:   req.MaterialID,
		MaterialCode: req.MaterialCode,
		MaterialName: req.MaterialName,
		Quantity:     req.Quantity,
		Unit:         unit,
		Status:       entity.PRStatusDraft,
		Source:       "MANUAL",
		Notes:        req.Notes,
		CreatedBy:    userID,
	}

	if req.RequiredDate != "" {
		t, err := time.Parse("2006-01-02", req.RequiredDate)
		if err == nil {
			pr.RequiredDate = &t
		}
	}

	if err := s.purchaseRepo.CreatePR(pr); err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}
	return pr, nil
}

func (s *ProcurementService) ListPRs(status string, page, size int) ([]entity.PurchaseRequisition, int64, error) {
	return s.purchaseRepo.ListPRs(status, page, size)
}

func (s *ProcurementService) ApprovePR(id, userID string) error {
	pr, err := s.purchaseRepo.GetPRByID(id)
	if err != nil {
		return fmt.Errorf("PR not found: %w", err)
	}
	if pr.Status != entity.PRStatusDraft && pr.Status != entity.PRStatusPending {
		return fmt.Errorf("PR状态不允许审批: %s", pr.Status)
	}
	now := time.Now()
	pr.Status = entity.PRStatusApproved
	pr.ApprovedBy = userID
	pr.ApprovedAt = &now
	return s.purchaseRepo.UpdatePR(pr)
}

// --- Purchase Order ---

type CreatePORequest struct {
	SupplierID   string          `json:"supplier_id" binding:"required"`
	ExpectedDate string          `json:"expected_date"`
	Currency     string          `json:"currency"`
	Notes        string          `json:"notes"`
	Items        []CreatePOItem  `json:"items" binding:"required,min=1"`
}

type CreatePOItem struct {
	MaterialID   string  `json:"material_id" binding:"required"`
	MaterialCode string  `json:"material_code"`
	MaterialName string  `json:"material_name"`
	Quantity     float64 `json:"quantity" binding:"required,gt=0"`
	Unit         string  `json:"unit"`
	UnitPrice    float64 `json:"unit_price" binding:"required,gt=0"`
	PRID         *string `json:"pr_id"` // 关联的PR
}

func (s *ProcurementService) CreatePO(req CreatePORequest, userID string) (*entity.PurchaseOrder, error) {
	// 验证供应商存在
	if _, err := s.supplierRepo.GetByID(req.SupplierID); err != nil {
		return nil, fmt.Errorf("供应商不存在: %w", err)
	}

	code := fmt.Sprintf("PO-%s%04d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)
	now := time.Now()
	currency := req.Currency
	if currency == "" {
		currency = "CNY"
	}

	po := &entity.PurchaseOrder{
		ID:         uuid.New().String(),
		POCode:     code,
		SupplierID: req.SupplierID,
		Status:     entity.POStatusDraft,
		Currency:   currency,
		OrderDate:  &now,
		Notes:      req.Notes,
		CreatedBy:  userID,
	}

	if req.ExpectedDate != "" {
		t, err := time.Parse("2006-01-02", req.ExpectedDate)
		if err == nil {
			po.ExpectedDate = &t
		}
	}

	var totalAmount float64
	var items []entity.POItem
	for _, item := range req.Items {
		amount := item.Quantity * item.UnitPrice
		totalAmount += amount
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		items = append(items, entity.POItem{
			ID:           uuid.New().String(),
			POID:         po.ID,
			MaterialID:   item.MaterialID,
			MaterialCode: item.MaterialCode,
			MaterialName: item.MaterialName,
			Quantity:     item.Quantity,
			Unit:         unit,
			UnitPrice:    item.UnitPrice,
			Amount:       amount,
			Status:       entity.POItemStatusOpen,
			PRID:         item.PRID,
		})
	}
	po.TotalAmount = totalAmount
	po.Items = items

	if err := s.purchaseRepo.CreatePO(po); err != nil {
		return nil, fmt.Errorf("failed to create PO: %w", err)
	}
	return po, nil
}

func (s *ProcurementService) GetPOByID(id string) (*entity.PurchaseOrder, error) {
	return s.purchaseRepo.GetPOByID(id)
}

func (s *ProcurementService) ListPOs(params repository.POListParams) ([]entity.PurchaseOrder, int64, error) {
	return s.purchaseRepo.ListPOs(params)
}

func (s *ProcurementService) SubmitPO(id string) error {
	po, err := s.purchaseRepo.GetPOByID(id)
	if err != nil {
		return fmt.Errorf("PO not found: %w", err)
	}
	if po.Status != entity.POStatusDraft {
		return fmt.Errorf("只有草稿状态的PO可以提交")
	}
	po.Status = entity.POStatusPending
	return s.purchaseRepo.UpdatePO(po)
}

func (s *ProcurementService) ApprovePO(id, userID string) error {
	po, err := s.purchaseRepo.GetPOByID(id)
	if err != nil {
		return fmt.Errorf("PO not found: %w", err)
	}
	if po.Status != entity.POStatusPending {
		return fmt.Errorf("只有待审批状态的PO可以审批")
	}
	now := time.Now()
	po.Status = entity.POStatusApproved
	po.ApprovedBy = userID
	po.ApprovedAt = &now
	return s.purchaseRepo.UpdatePO(po)
}

func (s *ProcurementService) RejectPO(id string) error {
	po, err := s.purchaseRepo.GetPOByID(id)
	if err != nil {
		return fmt.Errorf("PO not found: %w", err)
	}
	if po.Status != entity.POStatusPending {
		return fmt.Errorf("只有待审批状态的PO可以驳回")
	}
	po.Status = entity.POStatusDraft
	return s.purchaseRepo.UpdatePO(po)
}

func (s *ProcurementService) SendPO(id string) error {
	po, err := s.purchaseRepo.GetPOByID(id)
	if err != nil {
		return fmt.Errorf("PO not found: %w", err)
	}
	if po.Status != entity.POStatusApproved {
		return fmt.Errorf("只有已审批的PO可以发送")
	}
	po.Status = entity.POStatusSent
	return s.purchaseRepo.UpdatePO(po)
}

type ReceiveItemRequest struct {
	ItemID      string  `json:"item_id" binding:"required"`
	ReceivedQty float64 `json:"received_qty" binding:"required,gt=0"`
	WarehouseID string  `json:"warehouse_id" binding:"required"`
	BatchNo     string  `json:"batch_no"`
}

func (s *ProcurementService) ReceivePO(id string, items []ReceiveItemRequest, userID string) error {
	po, err := s.purchaseRepo.GetPOByID(id)
	if err != nil {
		return fmt.Errorf("PO not found: %w", err)
	}
	if po.Status != entity.POStatusSent && po.Status != entity.POStatusPartial {
		return fmt.Errorf("PO状态不允许收货: %s", po.Status)
	}

	allReceived := true
	for _, receiveItem := range items {
		for i := range po.Items {
			if po.Items[i].ID == receiveItem.ItemID {
				po.Items[i].ReceivedQty += receiveItem.ReceivedQty
				if po.Items[i].ReceivedQty >= po.Items[i].Quantity {
					po.Items[i].Status = entity.POItemStatusReceived
				} else {
					po.Items[i].Status = entity.POItemStatusPartial
					allReceived = false
				}
				if err := s.purchaseRepo.UpdatePOItem(&po.Items[i]); err != nil {
					return err
				}

				// 创建库存入库
				batchNo := receiveItem.BatchNo
				if batchNo == "" {
					batchNo = fmt.Sprintf("%s%03d", time.Now().Format("20060102"), time.Now().UnixNano()%1000)
				}
				inv := &entity.Inventory{
					ID:            uuid.New().String(),
					MaterialID:    po.Items[i].MaterialID,
					MaterialCode:  po.Items[i].MaterialCode,
					MaterialName:  po.Items[i].MaterialName,
					WarehouseID:   receiveItem.WarehouseID,
					BatchNo:       batchNo,
					InventoryType: entity.InventoryTypeRaw,
					Quantity:      receiveItem.ReceivedQty,
					AvailableQty:  receiveItem.ReceivedQty,
					UnitCost:      po.Items[i].UnitPrice,
					Unit:          po.Items[i].Unit,
				}
				now := time.Now()
				inv.LastMovedAt = &now

				// 先尝试查询已有库存记录
				existingInv, findErr := s.inventoryRepo.GetByMaterialAndWarehouse(po.Items[i].MaterialID, receiveItem.WarehouseID)
				if findErr == nil && existingInv != nil {
					existingInv.Quantity += receiveItem.ReceivedQty
					existingInv.AvailableQty += receiveItem.ReceivedQty
					existingInv.LastMovedAt = &now
					s.inventoryRepo.Update(existingInv)
				} else {
					s.inventoryRepo.UpsertInventory(inv)
				}

				// 创建库存交易记录
				tx := &entity.InventoryTransaction{
					ID:              uuid.New().String(),
					MaterialID:      po.Items[i].MaterialID,
					MaterialCode:    po.Items[i].MaterialCode,
					MaterialName:    po.Items[i].MaterialName,
					WarehouseID:     receiveItem.WarehouseID,
					TransactionType: entity.TxTypePurchaseIn,
					Quantity:        receiveItem.ReceivedQty,
					BatchNo:         batchNo,
					UnitCost:        po.Items[i].UnitPrice,
					ReferenceType:   "PO",
					ReferenceID:     po.ID,
					ReferenceCode:   po.POCode,
					CreatedBy:       userID,
				}
				s.inventoryRepo.CreateTransaction(tx)

				break
			}
		}
	}

	now := time.Now()
	if allReceived {
		// 检查所有行是否都已收货
		allDone := true
		for _, item := range po.Items {
			if item.Status != entity.POItemStatusReceived {
				allDone = false
				break
			}
		}
		if allDone {
			po.Status = entity.POStatusReceived
			po.ReceivedDate = &now
		} else {
			po.Status = entity.POStatusPartial
		}
	} else {
		po.Status = entity.POStatusPartial
	}

	return s.purchaseRepo.UpdatePO(po)
}
