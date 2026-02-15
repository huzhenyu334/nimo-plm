package service

import (
	"context"
	"errors"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
)

// EvaluationService 评估服务
type EvaluationService struct {
	repo         *repository.EvaluationRepository
	supplierRepo *repository.SupplierRepository
}

func NewEvaluationService(repo *repository.EvaluationRepository) *EvaluationService {
	return &EvaluationService{repo: repo}
}

func (s *EvaluationService) SetSupplierRepo(repo *repository.SupplierRepository) {
	s.supplierRepo = repo
}

// CreateEvaluationRequest 创建评估请求
type CreateEvaluationRequest struct {
	SupplierID    string   `json:"supplier_id" binding:"required"`
	Period        string   `json:"period" binding:"required"`
	EvalType      string   `json:"eval_type"`
	QualityScore  *float64 `json:"quality_score"`
	DeliveryScore *float64 `json:"delivery_score"`
	PriceScore    *float64 `json:"price_score"`
	ServiceScore  *float64 `json:"service_score"`
	Remarks       string   `json:"remarks"`
}

// UpdateEvaluationRequest 更新评估请求
type UpdateEvaluationRequest struct {
	QualityScore  *float64 `json:"quality_score"`
	DeliveryScore *float64 `json:"delivery_score"`
	PriceScore    *float64 `json:"price_score"`
	ServiceScore  *float64 `json:"service_score"`
	Remarks       *string  `json:"remarks"`
}

// AutoGenerateRequest 自动生成评估请求
type AutoGenerateRequest struct {
	SupplierID string `json:"supplier_id" binding:"required"`
	Period     string `json:"period" binding:"required"`
	EvalType   string `json:"eval_type"`
}

// List 获取评估列表
func (s *EvaluationService) List(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.SupplierEvaluation, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize, filters)
}

// Get 获取评估详情
func (s *EvaluationService) Get(ctx context.Context, id string) (*entity.SupplierEvaluation, error) {
	return s.repo.FindByID(ctx, id)
}

// Create 创建评估
func (s *EvaluationService) Create(ctx context.Context, userID string, req *CreateEvaluationRequest) (*entity.SupplierEvaluation, error) {
	eval := &entity.SupplierEvaluation{
		ID:             uuid.New().String()[:32],
		SupplierID:     req.SupplierID,
		Period:         req.Period,
		EvalType:       "quarterly",
		QualityScore:   req.QualityScore,
		DeliveryScore:  req.DeliveryScore,
		PriceScore:     req.PriceScore,
		ServiceScore:   req.ServiceScore,
		QualityWeight:  0.30,
		DeliveryWeight: 0.25,
		PriceWeight:    0.25,
		ServiceWeight:  0.20,
		Remarks:        req.Remarks,
		EvaluatorID:    userID,
		Status:         entity.EvalStatusDraft,
	}

	if req.EvalType != "" {
		eval.EvalType = req.EvalType
	}

	// 计算综合评分
	s.calcTotalScore(eval)

	if err := s.repo.Create(ctx, eval); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, eval.ID)
}

// Update 更新评估
func (s *EvaluationService) Update(ctx context.Context, id string, req *UpdateEvaluationRequest) (*entity.SupplierEvaluation, error) {
	eval, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if eval.Status != entity.EvalStatusDraft {
		return nil, errors.New("只能修改草稿状态的评估")
	}

	if req.QualityScore != nil {
		eval.QualityScore = req.QualityScore
	}
	if req.DeliveryScore != nil {
		eval.DeliveryScore = req.DeliveryScore
	}
	if req.PriceScore != nil {
		eval.PriceScore = req.PriceScore
	}
	if req.ServiceScore != nil {
		eval.ServiceScore = req.ServiceScore
	}
	if req.Remarks != nil {
		eval.Remarks = *req.Remarks
	}

	s.calcTotalScore(eval)

	if err := s.repo.Update(ctx, eval); err != nil {
		return nil, err
	}
	return eval, nil
}

// Submit 提交评估
func (s *EvaluationService) Submit(ctx context.Context, id string) (*entity.SupplierEvaluation, error) {
	eval, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if eval.Status != entity.EvalStatusDraft {
		return nil, errors.New("只能提交草稿状态的评估")
	}

	eval.Status = entity.EvalStatusSubmitted
	if err := s.repo.Update(ctx, eval); err != nil {
		return nil, err
	}
	return eval, nil
}

// Approve 审批评估
func (s *EvaluationService) Approve(ctx context.Context, id string) (*entity.SupplierEvaluation, error) {
	eval, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if eval.Status != entity.EvalStatusSubmitted {
		return nil, errors.New("只能审批已提交的评估")
	}

	eval.Status = entity.EvalStatusApproved
	if err := s.repo.Update(ctx, eval); err != nil {
		return nil, err
	}

	// 更新供应商评分
	if s.supplierRepo != nil {
		quality, delivery, price, _, _, avgErr := s.repo.FindSupplierAvgScores(ctx, eval.SupplierID)
		if avgErr == nil {
			_ = s.supplierRepo.UpdateScores(ctx, eval.SupplierID, quality, delivery, price, 0)
		}
	}

	return eval, nil
}

// GetSupplierHistory 获取供应商评估历史
func (s *EvaluationService) GetSupplierHistory(ctx context.Context, supplierID string) ([]entity.SupplierEvaluation, error) {
	return s.repo.FindBySupplier(ctx, supplierID)
}

// AutoGenerate 自动生成评估（基于PO数据）
func (s *EvaluationService) AutoGenerate(ctx context.Context, userID string, req *AutoGenerateRequest) (*entity.SupplierEvaluation, error) {
	evalType := "quarterly"
	if req.EvalType != "" {
		evalType = req.EvalType
	}

	eval := &entity.SupplierEvaluation{
		ID:             uuid.New().String()[:32],
		SupplierID:     req.SupplierID,
		Period:         req.Period,
		EvalType:       evalType,
		QualityWeight:  0.30,
		DeliveryWeight: 0.25,
		PriceWeight:    0.25,
		ServiceWeight:  0.20,
		EvaluatorID:    userID,
		Status:         entity.EvalStatusDraft,
	}

	if err := s.repo.Create(ctx, eval); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, eval.ID)
}

// calcTotalScore 计算综合评分
func (s *EvaluationService) calcTotalScore(eval *entity.SupplierEvaluation) {
	var total float64
	var hasScore bool

	if eval.QualityScore != nil {
		total += *eval.QualityScore * eval.QualityWeight
		hasScore = true
	}
	if eval.DeliveryScore != nil {
		total += *eval.DeliveryScore * eval.DeliveryWeight
		hasScore = true
	}
	if eval.PriceScore != nil {
		total += *eval.PriceScore * eval.PriceWeight
		hasScore = true
	}
	if eval.ServiceScore != nil {
		total += *eval.ServiceScore * eval.ServiceWeight
		hasScore = true
	}

	if hasScore {
		eval.TotalScore = &total
		eval.Grade = entity.CalcGrade(total)
	}
}
