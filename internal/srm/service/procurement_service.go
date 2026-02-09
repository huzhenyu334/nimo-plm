package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BOMProvider PLM BOM数据接口（避免直接依赖PLM包）
type BOMProvider interface {
	GetBOMWithItems(ctx context.Context, bomID string) (projectID string, phase string, items []BOMItemInfo, err error)
}

// BOMItemInfo BOM行项信息
type BOMItemInfo struct {
	MaterialID    string
	MaterialCode  string
	MaterialName  string
	Specification string
	Category      string
	Quantity      float64
	Unit          string
}

// ProcurementService 采购服务
type ProcurementService struct {
	prRepo *repository.PRRepository
	poRepo *repository.PORepository
	db     *gorm.DB
}

func NewProcurementService(prRepo *repository.PRRepository, poRepo *repository.PORepository, db *gorm.DB) *ProcurementService {
	return &ProcurementService{
		prRepo: prRepo,
		poRepo: poRepo,
		db:     db,
	}
}

// === 采购需求(PR) ===

// ListPRs 获取PR列表
func (s *ProcurementService) ListPRs(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseRequest, int64, error) {
	return s.prRepo.FindAll(ctx, page, pageSize, filters)
}

// GetPR 获取PR详情
func (s *ProcurementService) GetPR(ctx context.Context, id string) (*entity.PurchaseRequest, error) {
	return s.prRepo.FindByID(ctx, id)
}

// CreatePRRequest 创建PR请求
type CreatePRRequest struct {
	Title        string         `json:"title" binding:"required"`
	Type         string         `json:"type" binding:"required"`
	Priority     string         `json:"priority"`
	ProjectID    *string        `json:"project_id"`
	Phase        string         `json:"phase"`
	RequiredDate *time.Time     `json:"required_date"`
	Notes        string         `json:"notes"`
	Items        []CreatePRItem `json:"items"`
}

type CreatePRItem struct {
	MaterialID    *string  `json:"material_id"`
	MaterialCode  string   `json:"material_code"`
	MaterialName  string   `json:"material_name" binding:"required"`
	Specification string   `json:"specification"`
	Category      string   `json:"category"`
	Quantity      float64  `json:"quantity" binding:"required"`
	Unit          string   `json:"unit"`
	ExpectedDate  *time.Time `json:"expected_date"`
	Notes         string   `json:"notes"`
}

// CreatePR 创建采购需求
func (s *ProcurementService) CreatePR(ctx context.Context, userID string, req *CreatePRRequest) (*entity.PurchaseRequest, error) {
	code, err := s.prRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PR编码失败: %w", err)
	}

	pr := &entity.PurchaseRequest{
		ID:           uuid.New().String()[:32],
		PRCode:       code,
		Title:        req.Title,
		Type:         req.Type,
		Priority:     req.Priority,
		Status:       entity.PRStatusDraft,
		ProjectID:    req.ProjectID,
		Phase:        req.Phase,
		RequiredDate: req.RequiredDate,
		RequestedBy:  userID,
		Notes:        req.Notes,
	}

	if pr.Priority == "" {
		pr.Priority = "normal"
	}

	// 创建行项
	for i, item := range req.Items {
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		pr.Items = append(pr.Items, entity.PRItem{
			ID:            uuid.New().String()[:32],
			PRID:          pr.ID,
			MaterialID:    item.MaterialID,
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Category:      item.Category,
			Quantity:      item.Quantity,
			Unit:          unit,
			Status:        entity.PRItemStatusPending,
			ExpectedDate:  item.ExpectedDate,
			Notes:         item.Notes,
			SortOrder:     i + 1,
		})
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// UpdatePRRequest 更新PR请求
type UpdatePRRequest struct {
	Title        *string    `json:"title"`
	Type         *string    `json:"type"`
	Priority     *string    `json:"priority"`
	Phase        *string    `json:"phase"`
	RequiredDate *time.Time `json:"required_date"`
	Notes        *string    `json:"notes"`
}

// UpdatePR 更新采购需求
func (s *ProcurementService) UpdatePR(ctx context.Context, id string, req *UpdatePRRequest) (*entity.PurchaseRequest, error) {
	pr, err := s.prRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		pr.Title = *req.Title
	}
	if req.Type != nil {
		pr.Type = *req.Type
	}
	if req.Priority != nil {
		pr.Priority = *req.Priority
	}
	if req.Phase != nil {
		pr.Phase = *req.Phase
	}
	if req.RequiredDate != nil {
		pr.RequiredDate = req.RequiredDate
	}
	if req.Notes != nil {
		pr.Notes = *req.Notes
	}

	if err := s.prRepo.Update(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// ApprovePR 审批PR
func (s *ProcurementService) ApprovePR(ctx context.Context, id, userID string) (*entity.PurchaseRequest, error) {
	pr, err := s.prRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	pr.Status = entity.PRStatusApproved
	pr.ApprovedBy = &userID
	pr.ApprovedAt = &now

	if err := s.prRepo.Update(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// CreatePRFromBOM 从BOM创建采购需求
func (s *ProcurementService) CreatePRFromBOM(ctx context.Context, projectID, bomID, userID string, bomItems []BOMItemInfo, phase string) (*entity.PurchaseRequest, error) {
	// 防重复：检查是否已有该BOM的PR
	existing, err := s.prRepo.FindByBOMID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	code, err := s.prRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PR编码失败: %w", err)
	}

	pr := &entity.PurchaseRequest{
		ID:          uuid.New().String()[:32],
		PRCode:      code,
		Title:       fmt.Sprintf("BOM自动生成采购需求 - %s", phase),
		Type:        entity.PRTypeSample,
		Priority:    "normal",
		Status:      entity.PRStatusDraft,
		ProjectID:   &projectID,
		BOMID:       &bomID,
		Phase:       phase,
		RequestedBy: userID,
	}

	for i, item := range bomItems {
		pr.Items = append(pr.Items, entity.PRItem{
			ID:            uuid.New().String()[:32],
			PRID:          pr.ID,
			MaterialID:    strPtr(item.MaterialID),
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Category:      item.Category,
			Quantity:      item.Quantity,
			Unit:          item.Unit,
			Status:        entity.PRItemStatusPending,
			SortOrder:     i + 1,
		})
	}

	if err := s.prRepo.Create(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// === 采购订单(PO) ===

// ListPOs 获取PO列表
func (s *ProcurementService) ListPOs(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.PurchaseOrder, int64, error) {
	return s.poRepo.FindAll(ctx, page, pageSize, filters)
}

// GetPO 获取PO详情
func (s *ProcurementService) GetPO(ctx context.Context, id string) (*entity.PurchaseOrder, error) {
	return s.poRepo.FindByID(ctx, id)
}

// CreatePORequest 创建PO请求
type CreatePORequest struct {
	SupplierID      string         `json:"supplier_id" binding:"required"`
	PRID            *string        `json:"pr_id"`
	Type            string         `json:"type" binding:"required"`
	ExpectedDate    *time.Time     `json:"expected_date"`
	ShippingAddress string         `json:"shipping_address"`
	PaymentTerms    string         `json:"payment_terms"`
	Notes           string         `json:"notes"`
	Items           []CreatePOItem `json:"items"`
}

type CreatePOItem struct {
	PRItemID      *string  `json:"pr_item_id"`
	MaterialID    *string  `json:"material_id"`
	MaterialCode  string   `json:"material_code"`
	MaterialName  string   `json:"material_name" binding:"required"`
	Specification string   `json:"specification"`
	Quantity      float64  `json:"quantity" binding:"required"`
	Unit          string   `json:"unit"`
	UnitPrice     *float64 `json:"unit_price"`
	Notes         string   `json:"notes"`
}

// CreatePO 创建采购订单
func (s *ProcurementService) CreatePO(ctx context.Context, userID string, req *CreatePORequest) (*entity.PurchaseOrder, error) {
	code, err := s.poRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成PO编码失败: %w", err)
	}

	po := &entity.PurchaseOrder{
		ID:              uuid.New().String()[:32],
		POCode:          code,
		SupplierID:      req.SupplierID,
		PRID:            req.PRID,
		Type:            req.Type,
		Status:          entity.POStatusDraft,
		Currency:        "CNY",
		ExpectedDate:    req.ExpectedDate,
		ShippingAddress: req.ShippingAddress,
		PaymentTerms:    req.PaymentTerms,
		CreatedBy:       userID,
		Notes:           req.Notes,
	}

	var totalAmount float64
	for i, item := range req.Items {
		unit := item.Unit
		if unit == "" {
			unit = "pcs"
		}
		var itemTotal *float64
		if item.UnitPrice != nil {
			t := *item.UnitPrice * item.Quantity
			itemTotal = &t
			totalAmount += t
		}
		po.Items = append(po.Items, entity.POItem{
			ID:            uuid.New().String()[:32],
			POID:          po.ID,
			PRItemID:      item.PRItemID,
			MaterialID:    item.MaterialID,
			MaterialCode:  item.MaterialCode,
			MaterialName:  item.MaterialName,
			Specification: item.Specification,
			Quantity:      item.Quantity,
			Unit:          unit,
			UnitPrice:     item.UnitPrice,
			TotalAmount:   itemTotal,
			Status:        entity.POItemStatusPending,
			SortOrder:     i + 1,
			Notes:         item.Notes,
		})
	}

	if totalAmount > 0 {
		po.TotalAmount = &totalAmount
	}

	if err := s.poRepo.Create(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// UpdatePORequest 更新PO请求
type UpdatePORequest struct {
	ExpectedDate    *time.Time `json:"expected_date"`
	ShippingAddress *string    `json:"shipping_address"`
	PaymentTerms    *string    `json:"payment_terms"`
	Notes           *string    `json:"notes"`
}

// UpdatePO 更新采购订单
func (s *ProcurementService) UpdatePO(ctx context.Context, id string, req *UpdatePORequest) (*entity.PurchaseOrder, error) {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.ExpectedDate != nil {
		po.ExpectedDate = req.ExpectedDate
	}
	if req.ShippingAddress != nil {
		po.ShippingAddress = *req.ShippingAddress
	}
	if req.PaymentTerms != nil {
		po.PaymentTerms = *req.PaymentTerms
	}
	if req.Notes != nil {
		po.Notes = *req.Notes
	}

	if err := s.poRepo.Update(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// ApprovePO 审批PO
func (s *ProcurementService) ApprovePO(ctx context.Context, id, userID string) (*entity.PurchaseOrder, error) {
	po, err := s.poRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	po.Status = entity.POStatusApproved
	po.ApprovedBy = &userID
	po.ApprovedAt = &now

	if err := s.poRepo.Update(ctx, po); err != nil {
		return nil, err
	}
	return po, nil
}

// ReceiveItemRequest 收货请求
type ReceiveItemRequest struct {
	ReceivedQty float64 `json:"received_qty" binding:"required"`
}

// ReceiveItem 收货
func (s *ProcurementService) ReceiveItem(ctx context.Context, poID, itemID string) error {
	// 验证PO存在
	_, err := s.poRepo.FindByID(ctx, poID)
	if err != nil {
		return err
	}
	return nil // 实际收货在handler层调用repo
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
