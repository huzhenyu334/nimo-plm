package service

import (
	"context"
	"fmt"
	"log"
	"time"

	plmentity "github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SRMProjectService 采购项目服务
type SRMProjectService struct {
	projectRepo      *repository.ProjectRepository
	prRepo           *repository.PRRepository
	activityLogRepo  *repository.ActivityLogRepository
	delayRequestRepo *repository.DelayRequestRepository
	db               *gorm.DB
}

func NewSRMProjectService(
	projectRepo *repository.ProjectRepository,
	prRepo *repository.PRRepository,
	activityLogRepo *repository.ActivityLogRepository,
	delayRequestRepo *repository.DelayRequestRepository,
	db *gorm.DB,
) *SRMProjectService {
	return &SRMProjectService{
		projectRepo:      projectRepo,
		prRepo:           prRepo,
		activityLogRepo:  activityLogRepo,
		delayRequestRepo: delayRequestRepo,
		db:               db,
	}
}

// === 采购项目 CRUD ===

// ListProjects 获取采购项目列表
func (s *SRMProjectService) ListProjects(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.SRMProject, int64, error) {
	return s.projectRepo.FindAll(ctx, page, pageSize, filters)
}

// GetProject 获取采购项目详情
func (s *SRMProjectService) GetProject(ctx context.Context, id string) (*entity.SRMProject, error) {
	return s.projectRepo.FindByID(ctx, id)
}

// CreateProjectRequest 创建采购项目请求
type CreateProjectRequest struct {
	Name         string     `json:"name" binding:"required"`
	Type         string     `json:"type" binding:"required"` // sample/production
	Phase        string     `json:"phase"`
	PLMProjectID *string    `json:"plm_project_id"`
	PLMBOMID     *string    `json:"plm_bom_id"`
	StartDate    *time.Time `json:"start_date"`
	TargetDate   *time.Time `json:"target_date"`
}

// CreateProject 创建采购项目
func (s *SRMProjectService) CreateProject(ctx context.Context, userID string, req *CreateProjectRequest) (*entity.SRMProject, error) {
	code, err := s.projectRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成编码失败: %w", err)
	}

	now := time.Now()
	project := &entity.SRMProject{
		ID:           uuid.New().String()[:32],
		Code:         code,
		Name:         req.Name,
		Type:         req.Type,
		Phase:        req.Phase,
		Status:       entity.SRMProjectStatusActive,
		PLMProjectID: req.PLMProjectID,
		PLMBOMID:     req.PLMBOMID,
		StartDate:    req.StartDate,
		TargetDate:   req.TargetDate,
		CreatedBy:    userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if project.StartDate == nil {
		project.StartDate = &now
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}

	// 记录操作日志
	s.activityLogRepo.LogActivity(ctx, "project", project.ID, project.Code,
		"create", "", entity.SRMProjectStatusActive,
		fmt.Sprintf("创建采购项目: %s", project.Name), userID, "")

	return project, nil
}

// UpdateProjectRequest 更新采购项目请求
type UpdateProjectRequest struct {
	Name       *string    `json:"name"`
	Phase      *string    `json:"phase"`
	Status     *string    `json:"status"`
	TargetDate *time.Time `json:"target_date"`
}

// UpdateProject 更新采购项目
func (s *SRMProjectService) UpdateProject(ctx context.Context, id string, req *UpdateProjectRequest) (*entity.SRMProject, error) {
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldStatus := project.Status

	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Phase != nil {
		project.Phase = *req.Phase
	}
	if req.Status != nil {
		project.Status = *req.Status
		if *req.Status == entity.SRMProjectStatusCompleted {
			now := time.Now()
			project.ActualDate = &now
		}
	}
	if req.TargetDate != nil {
		project.TargetDate = req.TargetDate
	}
	project.UpdatedAt = time.Now()

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}

	if req.Status != nil && *req.Status != oldStatus {
		s.activityLogRepo.LogActivity(ctx, "project", project.ID, project.Code,
			"status_change", oldStatus, *req.Status,
			fmt.Sprintf("采购项目状态变更: %s → %s", oldStatus, *req.Status), "", "")
	}

	return project, nil
}

// === 从BOM创建采购项目 ===

// CreateFromBOM 从PLM BOM审批创建采购项目+PR
func (s *SRMProjectService) CreateFromBOM(ctx context.Context, plmProjectID, bomID, userID string) (*entity.SRMProject, error) {
	// 防重复
	existing, err := s.projectRepo.FindByPLMBOMID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		log.Printf("[SRM] BOM %s 已有采购项目 %s，跳过", bomID, existing.Code)
		return existing, nil
	}

	// 读取PLM项目和BOM信息
	var plmProject plmentity.Project
	if err := s.db.WithContext(ctx).Where("id = ?", plmProjectID).First(&plmProject).Error; err != nil {
		return nil, fmt.Errorf("读取PLM项目失败: %w", err)
	}

	var bom plmentity.ProjectBOM
	if err := s.db.WithContext(ctx).Where("id = ?", bomID).First(&bom).Error; err != nil {
		return nil, fmt.Errorf("读取BOM失败: %w", err)
	}

	var bomItems []plmentity.ProjectBOMItem
	if err := s.db.WithContext(ctx).Where("bom_id = ?", bomID).Order("item_number").Find(&bomItems).Error; err != nil {
		return nil, fmt.Errorf("读取BOM行项失败: %w", err)
	}

	if len(bomItems) == 0 {
		log.Printf("[SRM] BOM %s 无行项，跳过", bomID)
		return nil, nil
	}

	// 创建采购项目
	code, err := s.projectRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成编码失败: %w", err)
	}

	now := time.Now()
	project := &entity.SRMProject{
		ID:           uuid.New().String()[:32],
		Code:         code,
		Name:         fmt.Sprintf("%s - %s 打样采购", plmProject.Name, bom.Name),
		Type:         entity.SRMProjectTypeSample,
		Phase:        bom.BOMType,
		Status:       entity.SRMProjectStatusActive,
		PLMProjectID: &plmProjectID,
		PLMBOMID:     &bomID,
		TotalItems:   len(bomItems),
		StartDate:    &now,
		CreatedBy:    userID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("创建采购项目失败: %w", err)
	}

	// 记录操作日志
	s.activityLogRepo.LogActivity(ctx, "project", project.ID, project.Code,
		"create", "", entity.SRMProjectStatusActive,
		fmt.Sprintf("从PLM BOM自动创建采购项目，共%d个零件", len(bomItems)), userID, "")

	log.Printf("[SRM] 创建采购项目: %s (PLM项目=%s, BOM=%s, %d项)", project.Code, plmProject.Name, bom.Name, len(bomItems))

	return project, nil
}

// === 进度管理 ===

// UpdateProgress 重新计算并更新采购项目进度
func (s *SRMProjectService) UpdateProgress(ctx context.Context, srmProjectID string) error {
	project, err := s.projectRepo.FindByID(ctx, srmProjectID)
	if err != nil {
		return err
	}

	// 查询关联PR的所有行项状态统计
	type StatusCount struct {
		Status string
		Count  int
	}
	var counts []StatusCount
	err = s.db.WithContext(ctx).
		Raw(`SELECT i.status, COUNT(*) as count
			FROM srm_pr_items i
			JOIN srm_purchase_requests pr ON i.pr_id = pr.id
			WHERE pr.srm_project_id = ?
			GROUP BY i.status`, srmProjectID).
		Scan(&counts).Error
	if err != nil {
		return fmt.Errorf("统计进度失败: %w", err)
	}

	var total, sourcing, ordered, received, passed, failed int
	for _, c := range counts {
		total += c.Count
		switch c.Status {
		case "sourcing":
			sourcing += c.Count
		case "ordered":
			ordered += c.Count
		case "received":
			received += c.Count
		case "completed":
			passed += c.Count
		case "inspected":
			// 需要检查inspection_result
			passed += c.Count
		}
	}

	project.TotalItems = total
	project.SourcingCount = sourcing
	project.OrderedCount = ordered
	project.ReceivedCount = received
	project.PassedCount = passed
	project.FailedCount = failed
	project.UpdatedAt = time.Now()

	// 如果全部通过，自动完成项目
	if total > 0 && passed == total {
		project.Status = entity.SRMProjectStatusCompleted
		now := time.Now()
		project.ActualDate = &now
	}

	return s.projectRepo.Update(ctx, project)
}

// GetProjectProgress 获取采购项目进度（供PLM查询）
func (s *SRMProjectService) GetProjectProgress(ctx context.Context, srmProjectID string) (*entity.SRMProject, error) {
	// 先更新进度
	if err := s.UpdateProgress(ctx, srmProjectID); err != nil {
		log.Printf("[SRM] 更新进度失败: %v", err)
	}
	return s.projectRepo.FindByID(ctx, srmProjectID)
}

// === 延期审批 ===

// CreateDelayRequest 创建延期申请
type CreateDelayRequestReq struct {
	SRMProjectID  string `json:"srm_project_id" binding:"required"`
	PRItemID      string `json:"pr_item_id"`
	MaterialName  string `json:"material_name"`
	OriginalDays  int    `json:"original_days"`
	RequestedDays int    `json:"requested_days" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	ReasonType    string `json:"reason_type"`
}

func (s *SRMProjectService) CreateDelayRequest(ctx context.Context, userID string, req *CreateDelayRequestReq) (*entity.DelayRequest, error) {
	code, err := s.delayRequestRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("生成延期编码失败: %w", err)
	}

	now := time.Now()
	dr := &entity.DelayRequest{
		ID:            uuid.New().String()[:32],
		Code:          code,
		SRMProjectID:  req.SRMProjectID,
		PRItemID:      req.PRItemID,
		MaterialName:  req.MaterialName,
		OriginalDays:  req.OriginalDays,
		RequestedDays: req.RequestedDays,
		Reason:        req.Reason,
		ReasonType:    req.ReasonType,
		Status:        entity.DelayRequestStatusPending,
		RequestedBy:   userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.delayRequestRepo.Create(ctx, dr); err != nil {
		return nil, err
	}

	// 记录操作日志
	s.activityLogRepo.LogActivity(ctx, "pr_item", req.PRItemID, "",
		"delay_request", "", "",
		fmt.Sprintf("申请延期: %s交期从%d天延至%d天，原因: %s", req.MaterialName, req.OriginalDays, req.RequestedDays, req.Reason),
		userID, "")

	return dr, nil
}

// ListDelayRequests 获取延期申请列表
func (s *SRMProjectService) ListDelayRequests(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.DelayRequest, int64, error) {
	return s.delayRequestRepo.FindAll(ctx, page, pageSize, filters)
}

// GetDelayRequest 获取延期申请详情
func (s *SRMProjectService) GetDelayRequest(ctx context.Context, id string) (*entity.DelayRequest, error) {
	return s.delayRequestRepo.FindByID(ctx, id)
}

// ApproveDelayRequest 审批通过延期申请
func (s *SRMProjectService) ApproveDelayRequest(ctx context.Context, id, userID string) (*entity.DelayRequest, error) {
	dr, err := s.delayRequestRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr.Status != entity.DelayRequestStatusPending {
		return nil, fmt.Errorf("延期申请状态不正确: %s", dr.Status)
	}

	now := time.Now()
	dr.Status = entity.DelayRequestStatusApproved
	dr.ApprovedBy = &userID
	dr.ApprovedAt = &now
	dr.UpdatedAt = now

	if err := s.delayRequestRepo.Update(ctx, dr); err != nil {
		return nil, err
	}

	// 记录操作日志
	s.activityLogRepo.LogActivity(ctx, "pr_item", dr.PRItemID, "",
		"delay_approved", "pending", "approved",
		fmt.Sprintf("延期审批通过: %s延至%d天", dr.MaterialName, dr.RequestedDays),
		userID, "")

	return dr, nil
}

// RejectDelayRequest 驳回延期申请
func (s *SRMProjectService) RejectDelayRequest(ctx context.Context, id, userID string) (*entity.DelayRequest, error) {
	dr, err := s.delayRequestRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dr.Status != entity.DelayRequestStatusPending {
		return nil, fmt.Errorf("延期申请状态不正确: %s", dr.Status)
	}

	now := time.Now()
	dr.Status = entity.DelayRequestStatusRejected
	dr.ApprovedBy = &userID
	dr.ApprovedAt = &now
	dr.UpdatedAt = now

	if err := s.delayRequestRepo.Update(ctx, dr); err != nil {
		return nil, err
	}

	return dr, nil
}

// === 操作日志查询 ===

// ListActivityLogs 查询操作日志
func (s *SRMProjectService) ListActivityLogs(ctx context.Context, entityType, entityID string, page, pageSize int) ([]entity.ActivityLog, int64, error) {
	return s.activityLogRepo.FindByEntity(ctx, entityType, entityID, page, pageSize)
}

// GetActivityLogRepo 获取日志仓库（供其他服务写日志用）
func (s *SRMProjectService) GetActivityLogRepo() *repository.ActivityLogRepository {
	return s.activityLogRepo
}
