package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
)

// InspectionService 检验服务
type InspectionService struct {
	repo            *repository.InspectionRepository
	prRepo          *repository.PRRepository
	activityLogRepo *repository.ActivityLogRepository
}

func NewInspectionService(repo *repository.InspectionRepository, prRepo *repository.PRRepository) *InspectionService {
	return &InspectionService{
		repo:   repo,
		prRepo: prRepo,
	}
}

// SetActivityLogRepo 注入操作日志仓库
func (s *InspectionService) SetActivityLogRepo(repo *repository.ActivityLogRepository) {
	s.activityLogRepo = repo
}

// ListInspections 获取检验列表
func (s *InspectionService) ListInspections(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.Inspection, int64, error) {
	return s.repo.FindAll(ctx, page, pageSize, filters)
}

// GetInspection 获取检验详情
func (s *InspectionService) GetInspection(ctx context.Context, id string) (*entity.Inspection, error) {
	return s.repo.FindByID(ctx, id)
}

// UpdateInspectionRequest 更新检验请求
type UpdateInspectionRequest struct {
	InspectorID     *string          `json:"inspector_id"`
	SampleQty       *int             `json:"sample_qty"`
	InspectionItems *json.RawMessage `json:"inspection_items"`
	ReportURL       *string          `json:"report_url"`
	Notes           *string          `json:"notes"`
}

// UpdateInspection 更新检验
func (s *InspectionService) UpdateInspection(ctx context.Context, id string, req *UpdateInspectionRequest) (*entity.Inspection, error) {
	inspection, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.InspectorID != nil {
		inspection.InspectorID = req.InspectorID
	}
	if req.SampleQty != nil {
		inspection.SampleQty = req.SampleQty
	}
	if req.InspectionItems != nil {
		inspection.InspectionItems = *req.InspectionItems
	}
	if req.ReportURL != nil {
		inspection.ReportURL = *req.ReportURL
	}
	if req.Notes != nil {
		inspection.Notes = *req.Notes
	}

	// 如果有检验员分配，状态改为进行中
	if inspection.InspectorID != nil && inspection.Status == entity.InspectionStatusPending {
		inspection.Status = entity.InspectionStatusInProgress
	}

	if err := s.repo.Update(ctx, inspection); err != nil {
		return nil, err
	}
	return inspection, nil
}

// CompleteInspectionRequest 完成检验请求
type CompleteInspectionRequest struct {
	Result          string           `json:"result" binding:"required"` // passed/failed/conditional
	InspectionItems *json.RawMessage `json:"inspection_items"`
	Notes           string           `json:"notes"`
}

// CompleteInspection 完成检验
func (s *InspectionService) CompleteInspection(ctx context.Context, id, userID string, req *CompleteInspectionRequest) (*entity.Inspection, error) {
	inspection, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	inspection.Status = entity.InspectionStatusCompleted
	inspection.Result = req.Result
	inspection.InspectorID = &userID
	inspection.InspectedAt = &now
	if req.InspectionItems != nil {
		inspection.InspectionItems = *req.InspectionItems
	}
	if req.Notes != "" {
		inspection.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, inspection); err != nil {
		return nil, err
	}

	// 记录操作日志
	if s.activityLogRepo != nil {
		action := "inspect_pass"
		content := fmt.Sprintf("检验通过: %s", inspection.MaterialName)
		if req.Result == "failed" {
			action = "inspect_fail"
			content = fmt.Sprintf("检验不通过: %s", inspection.MaterialName)
		} else if req.Result == "conditional" {
			action = "inspect_conditional"
			content = fmt.Sprintf("让步接收: %s", inspection.MaterialName)
		}
		s.activityLogRepo.LogActivity(ctx, "inspection", inspection.ID, inspection.InspectionCode,
			action, "in_progress", "completed", content, userID, "")
	}

	return inspection, nil
}

// CreateInspectionFromPOItem 从PO行项创建检验任务
func (s *InspectionService) CreateInspectionFromPOItem(ctx context.Context, poID, poItemID, supplierID, materialID, materialCode, materialName string, quantity float64) (*entity.Inspection, error) {
	code, err := s.repo.GenerateCode(ctx)
	if err != nil {
		return nil, err
	}

	inspection := &entity.Inspection{
		ID:             uuid.New().String()[:32],
		InspectionCode: code,
		POID:           strPtr(poID),
		POItemID:       strPtr(poItemID),
		SupplierID:     strPtr(supplierID),
		MaterialID:     strPtr(materialID),
		MaterialCode:   materialCode,
		MaterialName:   materialName,
		Quantity:       &quantity,
		Status:         entity.InspectionStatusPending,
	}

	if err := s.repo.Create(ctx, inspection); err != nil {
		return nil, err
	}
	return inspection, nil
}
