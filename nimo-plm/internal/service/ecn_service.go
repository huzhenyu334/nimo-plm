package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/google/uuid"
)

// ECNService ECN服务
type ECNService struct {
	ecnRepo     *repository.ECNRepository
	productRepo *repository.ProductRepository
	feishuSvc   *FeishuIntegrationService
}

// NewECNService 创建ECN服务
func NewECNService(ecnRepo *repository.ECNRepository, productRepo *repository.ProductRepository, feishuSvc *FeishuIntegrationService) *ECNService {
	return &ECNService{
		ecnRepo:     ecnRepo,
		productRepo: productRepo,
		feishuSvc:   feishuSvc,
	}
}

// CreateECNRequest 创建ECN请求
type CreateECNRequest struct {
	Title          string                 `json:"title" binding:"required"`
	ProductID      string                 `json:"product_id" binding:"required"`
	ChangeType     string                 `json:"change_type" binding:"required"`
	Urgency        string                 `json:"urgency"`
	Reason         string                 `json:"reason" binding:"required"`
	Description    string                 `json:"description"`
	ImpactAnalysis string                 `json:"impact_analysis"`
	AffectedItems  []AffectedItemInput    `json:"affected_items"`
	ApproverIDs    []string               `json:"approver_ids"`
}

// AffectedItemInput 受影响项目输入
type AffectedItemInput struct {
	ItemType          string                 `json:"item_type" binding:"required"`
	ItemID            string                 `json:"item_id" binding:"required"`
	BeforeValue       map[string]interface{} `json:"before_value"`
	AfterValue        map[string]interface{} `json:"after_value"`
	ChangeDescription string                 `json:"change_description"`
}

// UpdateECNRequest 更新ECN请求
type UpdateECNRequest struct {
	Title          string `json:"title"`
	ChangeType     string `json:"change_type"`
	Urgency        string `json:"urgency"`
	Reason         string `json:"reason"`
	Description    string `json:"description"`
	ImpactAnalysis string `json:"impact_analysis"`
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Decision string `json:"decision" binding:"required"` // approve/reject
	Comment  string `json:"comment"`
}

// ECNListResult ECN列表结果
type ECNListResult struct {
	Items      []entity.ECN `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}

// List 获取ECN列表
func (s *ECNService) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) (*ECNListResult, error) {
	ecns, total, err := s.ecnRepo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list ECNs: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &ECNListResult{
		Items:      ecns,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Get 获取ECN详情
func (s *ECNService) Get(ctx context.Context, id string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}
	return ecn, nil
}

// Create 创建ECN
func (s *ECNService) Create(ctx context.Context, userID string, req *CreateECNRequest) (*entity.ECN, error) {
	// 验证产品存在
	_, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// 生成ECN编码
	code, err := s.ecnRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	urgency := req.Urgency
	if urgency == "" {
		urgency = entity.ECNUrgencyMedium
	}

	now := time.Now()
	ecn := &entity.ECN{
		ID:             uuid.New().String()[:32],
		Code:           code,
		Title:          req.Title,
		ProductID:      req.ProductID,
		ChangeType:     req.ChangeType,
		Urgency:        urgency,
		Status:         entity.ECNStatusDraft,
		Reason:         req.Reason,
		Description:    req.Description,
		ImpactAnalysis: req.ImpactAnalysis,
		RequestedBy:    userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.ecnRepo.Create(ctx, ecn); err != nil {
		return nil, fmt.Errorf("create ECN: %w", err)
	}

	// 添加受影响项目
	for _, item := range req.AffectedItems {
		affectedItem := &entity.ECNAffectedItem{
			ID:                uuid.New().String()[:32],
			ECNID:             ecn.ID,
			ItemType:          item.ItemType,
			ItemID:            item.ItemID,
			BeforeValue:       entity.JSONB(item.BeforeValue),
			AfterValue:        entity.JSONB(item.AfterValue),
			ChangeDescription: item.ChangeDescription,
			CreatedAt:         now,
		}
		if err := s.ecnRepo.AddAffectedItem(ctx, affectedItem); err != nil {
			return nil, fmt.Errorf("add affected item: %w", err)
		}
	}

	// 添加审批人
	for i, approverID := range req.ApproverIDs {
		approval := &entity.ECNApproval{
			ID:         uuid.New().String()[:32],
			ECNID:      ecn.ID,
			ApproverID: approverID,
			Sequence:   i + 1,
			Status:     "pending",
			CreatedAt:  now,
		}
		if err := s.ecnRepo.AddApproval(ctx, approval); err != nil {
			return nil, fmt.Errorf("add approval: %w", err)
		}
	}

	return ecn, nil
}

// Update 更新ECN
func (s *ECNService) Update(ctx context.Context, id string, req *UpdateECNRequest) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	// 只有草稿状态可以编辑
	if ecn.Status != entity.ECNStatusDraft {
		return nil, fmt.Errorf("ECN can only be updated in draft status")
	}

	if req.Title != "" {
		ecn.Title = req.Title
	}
	if req.ChangeType != "" {
		ecn.ChangeType = req.ChangeType
	}
	if req.Urgency != "" {
		ecn.Urgency = req.Urgency
	}
	if req.Reason != "" {
		ecn.Reason = req.Reason
	}
	if req.Description != "" {
		ecn.Description = req.Description
	}
	if req.ImpactAnalysis != "" {
		ecn.ImpactAnalysis = req.ImpactAnalysis
	}

	ecn.UpdatedAt = time.Now()

	if err := s.ecnRepo.Update(ctx, ecn); err != nil {
		return nil, fmt.Errorf("update ECN: %w", err)
	}

	return ecn, nil
}

// Submit 提交审批
func (s *ECNService) Submit(ctx context.Context, id string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft {
		return nil, fmt.Errorf("ECN can only be submitted from draft status")
	}

	// 检查是否有审批人
	approvals, err := s.ecnRepo.ListApprovals(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	if len(approvals) == 0 {
		return nil, fmt.Errorf("no approvers assigned")
	}

	if err := s.ecnRepo.SubmitForApproval(ctx, id); err != nil {
		return nil, fmt.Errorf("submit for approval: %w", err)
	}

	// 如果配置了飞书审批，创建审批实例
	if s.feishuSvc != nil {
		// TODO: 发起飞书审批
	}

	return s.ecnRepo.FindByID(ctx, id)
}

// Approve 审批通过
func (s *ECNService) Approve(ctx context.Context, id string, userID string, comment string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusPending {
		return nil, fmt.Errorf("ECN is not pending approval")
	}

	if err := s.ecnRepo.Approve(ctx, id, userID, comment); err != nil {
		return nil, fmt.Errorf("approve ECN: %w", err)
	}

	return s.ecnRepo.FindByID(ctx, id)
}

// Reject 审批拒绝
func (s *ECNService) Reject(ctx context.Context, id string, userID string, reason string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusPending {
		return nil, fmt.Errorf("ECN is not pending approval")
	}

	if err := s.ecnRepo.Reject(ctx, id, userID, reason); err != nil {
		return nil, fmt.Errorf("reject ECN: %w", err)
	}

	return s.ecnRepo.FindByID(ctx, id)
}

// Implement 实施ECN
func (s *ECNService) Implement(ctx context.Context, id string, userID string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusApproved {
		return nil, fmt.Errorf("ECN must be approved before implementation")
	}

	if err := s.ecnRepo.Implement(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("implement ECN: %w", err)
	}

	return s.ecnRepo.FindByID(ctx, id)
}

// AddAffectedItem 添加受影响项目
func (s *ECNService) AddAffectedItem(ctx context.Context, ecnID string, input *AffectedItemInput) (*entity.ECNAffectedItem, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft {
		return nil, fmt.Errorf("cannot add affected items to non-draft ECN")
	}

	item := &entity.ECNAffectedItem{
		ID:                uuid.New().String()[:32],
		ECNID:             ecnID,
		ItemType:          input.ItemType,
		ItemID:            input.ItemID,
		BeforeValue:       entity.JSONB(input.BeforeValue),
		AfterValue:        entity.JSONB(input.AfterValue),
		ChangeDescription: input.ChangeDescription,
		CreatedAt:         time.Now(),
	}

	if err := s.ecnRepo.AddAffectedItem(ctx, item); err != nil {
		return nil, fmt.Errorf("add affected item: %w", err)
	}

	return item, nil
}

// RemoveAffectedItem 移除受影响项目
func (s *ECNService) RemoveAffectedItem(ctx context.Context, ecnID, itemID string) error {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft {
		return fmt.Errorf("cannot remove affected items from non-draft ECN")
	}

	return s.ecnRepo.RemoveAffectedItem(ctx, itemID)
}

// ListAffectedItems 获取受影响项目列表
func (s *ECNService) ListAffectedItems(ctx context.Context, ecnID string) ([]entity.ECNAffectedItem, error) {
	return s.ecnRepo.ListAffectedItems(ctx, ecnID)
}

// ListApprovals 获取审批记录
func (s *ECNService) ListApprovals(ctx context.Context, ecnID string) ([]entity.ECNApproval, error) {
	return s.ecnRepo.ListApprovals(ctx, ecnID)
}

// AddApprover 添加审批人
func (s *ECNService) AddApprover(ctx context.Context, ecnID string, approverID string) (*entity.ECNApproval, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft {
		return nil, fmt.Errorf("cannot add approvers to non-draft ECN")
	}

	// 获取当前最大序号
	approvals, err := s.ecnRepo.ListApprovals(ctx, ecnID)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}

	maxSeq := 0
	for _, a := range approvals {
		if a.Sequence > maxSeq {
			maxSeq = a.Sequence
		}
	}

	approval := &entity.ECNApproval{
		ID:         uuid.New().String()[:32],
		ECNID:      ecnID,
		ApproverID: approverID,
		Sequence:   maxSeq + 1,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := s.ecnRepo.AddApproval(ctx, approval); err != nil {
		return nil, fmt.Errorf("add approval: %w", err)
	}

	return approval, nil
}

// ListByProduct 获取产品的ECN列表
func (s *ECNService) ListByProduct(ctx context.Context, productID string) ([]entity.ECN, error) {
	return s.ecnRepo.ListByProduct(ctx, productID)
}
