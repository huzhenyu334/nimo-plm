package service

import (
	"context"
	"errors"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
)

// SettlementService 结算服务
type SettlementService struct {
	repo *repository.SettlementRepository
}

func NewSettlementService(repo *repository.SettlementRepository) *SettlementService {
	return &SettlementService{repo: repo}
}

// CreateSettlementRequest 创建对账单请求
type CreateSettlementRequest struct {
	SupplierID  string  `json:"supplier_id" binding:"required"`
	PeriodStart *string `json:"period_start"`
	PeriodEnd   *string `json:"period_end"`
	Deduction   *float64 `json:"deduction"`
	Currency    string  `json:"currency"`
	Notes       string  `json:"notes"`
}

// UpdateSettlementRequest 更新对账单请求
type UpdateSettlementRequest struct {
	InvoiceNo     *string  `json:"invoice_no"`
	InvoiceAmount *float64 `json:"invoice_amount"`
	InvoiceURL    *string  `json:"invoice_url"`
	Deduction     *float64 `json:"deduction"`
	Notes         *string  `json:"notes"`
}

// GenerateSettlementRequest 自动生成对账单请求
type GenerateSettlementRequest struct {
	SupplierID  string `json:"supplier_id" binding:"required"`
	PeriodStart string `json:"period_start" binding:"required"`
	PeriodEnd   string `json:"period_end" binding:"required"`
}

// CreateDisputeRequest 创建差异记录请求
type CreateDisputeRequest struct {
	DisputeType string   `json:"dispute_type" binding:"required"`
	Description string   `json:"description"`
	AmountDiff  *float64 `json:"amount_diff"`
}

// UpdateDisputeRequest 更新差异记录请求
type UpdateDisputeRequest struct {
	Status     *string `json:"status"`
	Resolution *string `json:"resolution"`
}

// List 获取对账单列表
func (s *SettlementService) List(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Settlement, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize, filters)
}

// Get 获取对账单详情
func (s *SettlementService) Get(ctx context.Context, id string) (*entity.Settlement, error) {
	return s.repo.FindByID(ctx, id)
}

// Create 创建对账单
func (s *SettlementService) Create(ctx context.Context, userID string, req *CreateSettlementRequest) (*entity.Settlement, error) {
	code, err := s.repo.GenerateCode(ctx)
	if err != nil {
		return nil, err
	}

	settlement := &entity.Settlement{
		ID:             uuid.New().String()[:32],
		SettlementCode: code,
		SupplierID:     req.SupplierID,
		Status:         "draft",
		Currency:       "CNY",
		CreatedBy:      userID,
		Notes:          req.Notes,
	}

	if req.Currency != "" {
		settlement.Currency = req.Currency
	}
	if req.Deduction != nil {
		settlement.Deduction = req.Deduction
	}

	if req.PeriodStart != nil {
		t, err := time.Parse("2006-01-02", *req.PeriodStart)
		if err == nil {
			settlement.PeriodStart = &t
		}
	}
	if req.PeriodEnd != nil {
		t, err := time.Parse("2006-01-02", *req.PeriodEnd)
		if err == nil {
			settlement.PeriodEnd = &t
		}
	}

	if err := s.repo.Create(ctx, settlement); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, settlement.ID)
}

// Generate 自动生成对账单（汇总已收货PO）
func (s *SettlementService) Generate(ctx context.Context, userID string, req *GenerateSettlementRequest) (*entity.Settlement, error) {
	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		return nil, errors.New("无效的开始日期")
	}
	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		return nil, errors.New("无效的结束日期")
	}

	pos, err := s.repo.FindReceivedPOs(ctx, req.SupplierID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	var poAmount float64
	for _, po := range pos {
		if po.TotalAmount != nil {
			poAmount += *po.TotalAmount
		}
	}

	code, err := s.repo.GenerateCode(ctx)
	if err != nil {
		return nil, err
	}

	finalAmount := poAmount
	settlement := &entity.Settlement{
		ID:             uuid.New().String()[:32],
		SettlementCode: code,
		SupplierID:     req.SupplierID,
		PeriodStart:    &periodStart,
		PeriodEnd:      &periodEnd,
		Status:         "draft",
		POAmount:       &poAmount,
		ReceivedAmount: &poAmount,
		FinalAmount:    &finalAmount,
		Currency:       "CNY",
		CreatedBy:      userID,
	}

	if err := s.repo.Create(ctx, settlement); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, settlement.ID)
}

// Update 更新对账单
func (s *SettlementService) Update(ctx context.Context, id string, req *UpdateSettlementRequest) (*entity.Settlement, error) {
	settlement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.InvoiceNo != nil {
		settlement.InvoiceNo = *req.InvoiceNo
	}
	if req.InvoiceAmount != nil {
		settlement.InvoiceAmount = req.InvoiceAmount
	}
	if req.InvoiceURL != nil {
		settlement.InvoiceURL = *req.InvoiceURL
	}
	if req.Deduction != nil {
		settlement.Deduction = req.Deduction
		// 重新计算最终金额
		if settlement.POAmount != nil {
			final := *settlement.POAmount - *req.Deduction
			settlement.FinalAmount = &final
		}
	}
	if req.Notes != nil {
		settlement.Notes = *req.Notes
	}

	if err := s.repo.Update(ctx, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}

// Delete 删除对账单
func (s *SettlementService) Delete(ctx context.Context, id string) error {
	settlement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if settlement.Status != "draft" {
		return errors.New("只能删除草稿状态的对账单")
	}
	return s.repo.Delete(ctx, id)
}

// ConfirmByBuyer 采购方确认
func (s *SettlementService) ConfirmByBuyer(ctx context.Context, id string) (*entity.Settlement, error) {
	settlement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if settlement.Status != "draft" {
		return nil, errors.New("当前状态不允许确认")
	}

	settlement.ConfirmedByBuyer = true
	if settlement.ConfirmedBySupplier {
		now := time.Now()
		settlement.Status = "confirmed"
		settlement.ConfirmedAt = &now
	}

	if err := s.repo.Update(ctx, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}

// ConfirmBySupplier 供应商确认
func (s *SettlementService) ConfirmBySupplier(ctx context.Context, id string) (*entity.Settlement, error) {
	settlement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if settlement.Status != "draft" {
		return nil, errors.New("当前状态不允许确认")
	}

	settlement.ConfirmedBySupplier = true
	if settlement.ConfirmedByBuyer {
		now := time.Now()
		settlement.Status = "confirmed"
		settlement.ConfirmedAt = &now
	}

	if err := s.repo.Update(ctx, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}

// Transition status methods for invoiced/paid
func (s *SettlementService) UpdateStatus(ctx context.Context, id string, newStatus string) (*entity.Settlement, error) {
	settlement, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	validTransitions := map[string][]string{
		"draft":     {"confirmed"},
		"confirmed": {"invoiced"},
		"invoiced":  {"paid"},
	}

	allowed := false
	for _, target := range validTransitions[settlement.Status] {
		if target == newStatus {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("状态转换不允许")
	}

	settlement.Status = newStatus
	if err := s.repo.Update(ctx, settlement); err != nil {
		return nil, err
	}
	return settlement, nil
}

// AddDispute 添加差异记录
func (s *SettlementService) AddDispute(ctx context.Context, settlementID string, req *CreateDisputeRequest) (*entity.SettlementDispute, error) {
	// 验证对账单存在
	_, err := s.repo.FindByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}

	dispute := &entity.SettlementDispute{
		ID:           uuid.New().String()[:32],
		SettlementID: settlementID,
		DisputeType:  req.DisputeType,
		Description:  req.Description,
		AmountDiff:   req.AmountDiff,
		Status:       "open",
	}

	if err := s.repo.CreateDispute(ctx, dispute); err != nil {
		return nil, err
	}
	return dispute, nil
}

// UpdateDispute 更新差异记录
func (s *SettlementService) UpdateDispute(ctx context.Context, disputeID string, req *UpdateDisputeRequest) (*entity.SettlementDispute, error) {
	dispute, err := s.repo.FindDisputeByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}

	if req.Status != nil {
		dispute.Status = *req.Status
	}
	if req.Resolution != nil {
		dispute.Resolution = *req.Resolution
	}

	if err := s.repo.UpdateDispute(ctx, dispute); err != nil {
		return nil, err
	}
	return dispute, nil
}
