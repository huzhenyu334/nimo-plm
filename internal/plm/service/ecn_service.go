package service

import (
	"context"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
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
	TechnicalPlan  string                 `json:"technical_plan"`
	PlannedDate    *time.Time             `json:"planned_date"`
	ApprovalMode   string                 `json:"approval_mode"`
	SOPImpact      map[string]interface{} `json:"sop_impact"`
	AffectedItems  []AffectedItemInput    `json:"affected_items"`
	ApproverIDs    []string               `json:"approver_ids"`
}

// AffectedItemInput 受影响项目输入
type AffectedItemInput struct {
	ItemType          string                 `json:"item_type" binding:"required"`
	ItemID            string                 `json:"item_id" binding:"required"`
	MaterialCode      string                 `json:"material_code"`
	MaterialName      string                 `json:"material_name"`
	AffectedBOMIDs    []string               `json:"affected_bom_ids"`
	BeforeValue       map[string]interface{} `json:"before_value"`
	AfterValue        map[string]interface{} `json:"after_value"`
	ChangeDescription string                 `json:"change_description"`
}

// UpdateECNRequest 更新ECN请求
type UpdateECNRequest struct {
	Title          string                 `json:"title"`
	ChangeType     string                 `json:"change_type"`
	Urgency        string                 `json:"urgency"`
	Reason         string                 `json:"reason"`
	Description    string                 `json:"description"`
	ImpactAnalysis string                 `json:"impact_analysis"`
	TechnicalPlan  string                 `json:"technical_plan"`
	PlannedDate    *time.Time             `json:"planned_date"`
	ApprovalMode   string                 `json:"approval_mode"`
	SOPImpact      map[string]interface{} `json:"sop_impact"`
}

// UpdateAffectedItemRequest 更新受影响项请求
type UpdateAffectedItemRequest struct {
	MaterialCode      string                 `json:"material_code"`
	MaterialName      string                 `json:"material_name"`
	AffectedBOMIDs    []string               `json:"affected_bom_ids"`
	BeforeValue       map[string]interface{} `json:"before_value"`
	AfterValue        map[string]interface{} `json:"after_value"`
	ChangeDescription string                 `json:"change_description"`
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Decision string `json:"decision" binding:"required"` // approve/reject
	Comment  string `json:"comment"`
}

// CreateECNTaskRequest 创建执行任务请求
type CreateECNTaskRequest struct {
	Type        string     `json:"type" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	AssigneeID  string     `json:"assignee_id"`
	DueDate     *time.Time `json:"due_date"`
	SortOrder   int        `json:"sort_order"`
}

// UpdateECNTaskRequest 更新执行任务请求
type UpdateECNTaskRequest struct {
	Status      string     `json:"status"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssigneeID  string     `json:"assignee_id"`
	DueDate     *time.Time `json:"due_date"`
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

// GetStats 获取ECN统计
func (s *ECNService) GetStats(ctx context.Context, userID string) (*repository.ECNStats, error) {
	return s.ecnRepo.GetStats(ctx, userID)
}

// ListMyPending 获取待我审批的ECN
func (s *ECNService) ListMyPending(ctx context.Context, userID string) ([]entity.ECN, error) {
	return s.ecnRepo.ListMyPending(ctx, userID)
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

	approvalMode := req.ApprovalMode
	if approvalMode == "" {
		approvalMode = entity.ECNApprovalModeSerial
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
		TechnicalPlan:  req.TechnicalPlan,
		PlannedDate:    req.PlannedDate,
		ApprovalMode:   approvalMode,
		SOPImpact:      entity.JSONB(req.SOPImpact),
		RequestedBy:    userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.ecnRepo.Create(ctx, ecn); err != nil {
		return nil, fmt.Errorf("create ECN: %w", err)
	}

	// 添加受影响项目
	for _, item := range req.AffectedItems {
		var bomIDs interface{} = item.AffectedBOMIDs
		if bomIDs == nil {
			bomIDs = []string{}
		}
		affectedItem := &entity.ECNAffectedItem{
			ID:                uuid.New().String()[:32],
			ECNID:             ecn.ID,
			ItemType:          item.ItemType,
			ItemID:            item.ItemID,
			MaterialCode:      item.MaterialCode,
			MaterialName:      item.MaterialName,
			AffectedBOMIDs:    entity.JSONB(map[string]interface{}{"ids": item.AffectedBOMIDs}),
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

	// 记录历史
	s.addHistory(ctx, ecn.ID, userID, entity.ECNHistoryCreated, map[string]interface{}{
		"title": ecn.Title,
		"code":  ecn.Code,
	})

	return ecn, nil
}

// Update 更新ECN
func (s *ECNService) Update(ctx context.Context, id string, userID string, req *UpdateECNRequest) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	// 只有草稿和驳回状态可以编辑
	if ecn.Status != entity.ECNStatusDraft && ecn.Status != entity.ECNStatusRejected {
		return nil, fmt.Errorf("ECN can only be updated in draft or rejected status")
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
	if req.TechnicalPlan != "" {
		ecn.TechnicalPlan = req.TechnicalPlan
	}
	if req.PlannedDate != nil {
		ecn.PlannedDate = req.PlannedDate
	}
	if req.ApprovalMode != "" {
		ecn.ApprovalMode = req.ApprovalMode
	}
	if req.SOPImpact != nil {
		ecn.SOPImpact = entity.JSONB(req.SOPImpact)
	}

	// 如果是驳回状态，更新回草稿
	if ecn.Status == entity.ECNStatusRejected {
		ecn.Status = entity.ECNStatusDraft
		ecn.RejectionReason = ""
	}

	ecn.UpdatedAt = time.Now()

	if err := s.ecnRepo.Update(ctx, ecn); err != nil {
		return nil, fmt.Errorf("update ECN: %w", err)
	}

	s.addHistory(ctx, ecn.ID, userID, entity.ECNHistoryUpdated, nil)

	return ecn, nil
}

// Submit 提交审批
func (s *ECNService) Submit(ctx context.Context, id string, userID string) (*entity.ECN, error) {
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

	s.addHistory(ctx, id, userID, entity.ECNHistorySubmitted, nil)

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

	s.addHistory(ctx, id, userID, entity.ECNHistoryApproved, map[string]interface{}{
		"comment": comment,
	})

	// 检查是否全部审批通过（进入执行态），自动生成执行任务
	updatedECN, _ := s.ecnRepo.FindByID(ctx, id)
	if updatedECN != nil && updatedECN.Status == entity.ECNStatusExecuting {
		s.generateDefaultTasks(ctx, updatedECN)
		s.addHistory(ctx, id, userID, entity.ECNHistoryExecuting, nil)
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

	s.addHistory(ctx, id, userID, entity.ECNHistoryRejected, map[string]interface{}{
		"reason": reason,
	})

	return s.ecnRepo.FindByID(ctx, id)
}

// Implement 实施ECN
func (s *ECNService) Implement(ctx context.Context, id string, userID string) (*entity.ECN, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusApproved && ecn.Status != entity.ECNStatusExecuting {
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

	if ecn.Status != entity.ECNStatusDraft && ecn.Status != entity.ECNStatusRejected {
		return nil, fmt.Errorf("cannot add affected items to non-draft ECN")
	}

	item := &entity.ECNAffectedItem{
		ID:                uuid.New().String()[:32],
		ECNID:             ecnID,
		ItemType:          input.ItemType,
		ItemID:            input.ItemID,
		MaterialCode:      input.MaterialCode,
		MaterialName:      input.MaterialName,
		AffectedBOMIDs:    entity.JSONB(map[string]interface{}{"ids": input.AffectedBOMIDs}),
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

// UpdateAffectedItem 更新受影响项目
func (s *ECNService) UpdateAffectedItem(ctx context.Context, ecnID, itemID string, req *UpdateAffectedItemRequest) (*entity.ECNAffectedItem, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft && ecn.Status != entity.ECNStatusRejected {
		return nil, fmt.Errorf("cannot update affected items of non-draft ECN")
	}

	item, err := s.ecnRepo.FindAffectedItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("find affected item: %w", err)
	}

	if req.MaterialCode != "" {
		item.MaterialCode = req.MaterialCode
	}
	if req.MaterialName != "" {
		item.MaterialName = req.MaterialName
	}
	if req.AffectedBOMIDs != nil {
		item.AffectedBOMIDs = entity.JSONB(map[string]interface{}{"ids": req.AffectedBOMIDs})
	}
	if req.BeforeValue != nil {
		item.BeforeValue = entity.JSONB(req.BeforeValue)
	}
	if req.AfterValue != nil {
		item.AfterValue = entity.JSONB(req.AfterValue)
	}
	if req.ChangeDescription != "" {
		item.ChangeDescription = req.ChangeDescription
	}

	if err := s.ecnRepo.UpdateAffectedItem(ctx, item); err != nil {
		return nil, fmt.Errorf("update affected item: %w", err)
	}

	return item, nil
}

// RemoveAffectedItem 移除受影响项目
func (s *ECNService) RemoveAffectedItem(ctx context.Context, ecnID, itemID string) error {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusDraft && ecn.Status != entity.ECNStatusRejected {
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

// ============================================================
// 执行任务管理
// ============================================================

// CreateTask 创建执行任务
func (s *ECNService) CreateTask(ctx context.Context, ecnID, userID string, req *CreateECNTaskRequest) (*entity.ECNTask, error) {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return nil, fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusExecuting && ecn.Status != entity.ECNStatusApproved {
		return nil, fmt.Errorf("can only add tasks to executing ECN")
	}

	now := time.Now()
	task := &entity.ECNTask{
		ID:          uuid.New().String()[:32],
		ECNID:       ecnID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		DueDate:     req.DueDate,
		Status:      entity.ECNTaskStatusPending,
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.ecnRepo.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	return task, nil
}

// UpdateTask 更新执行任务
func (s *ECNService) UpdateTask(ctx context.Context, ecnID, taskID, userID string, req *UpdateECNTaskRequest) (*entity.ECNTask, error) {
	task, err := s.ecnRepo.FindTaskByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	if task.ECNID != ecnID {
		return nil, fmt.Errorf("task does not belong to this ECN")
	}

	if req.Status != "" {
		task.Status = req.Status
		if req.Status == entity.ECNTaskStatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
			task.CompletedBy = &userID
		}
	}
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.AssigneeID != "" {
		task.AssigneeID = req.AssigneeID
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	task.UpdatedAt = time.Now()

	if err := s.ecnRepo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	// 如果标记为完成，记录历史并检查是否所有任务完成
	if req.Status == entity.ECNTaskStatusCompleted {
		s.addHistory(ctx, ecnID, userID, entity.ECNHistoryTaskCompleted, map[string]interface{}{
			"task_title": task.Title,
			"task_type":  task.Type,
		})

		// 更新完成率
		rate, _ := s.ecnRepo.GetTaskCompletion(ctx, ecnID)
		s.ecnRepo.UpdateCompletionRate(ctx, ecnID, rate)

		// 如果所有任务完成，自动关闭ECN
		if rate == 100 {
			s.ecnRepo.CloseECN(ctx, ecnID)
			s.addHistory(ctx, ecnID, userID, entity.ECNHistoryClosed, nil)
		}
	}

	return task, nil
}

// ListTasks 获取执行任务列表
func (s *ECNService) ListTasks(ctx context.Context, ecnID string) ([]entity.ECNTask, error) {
	return s.ecnRepo.ListTasks(ctx, ecnID)
}

// ============================================================
// 历史记录
// ============================================================

// ListHistory 获取操作历史
func (s *ECNService) ListHistory(ctx context.Context, ecnID string) ([]entity.ECNHistory, error) {
	return s.ecnRepo.ListHistory(ctx, ecnID)
}

// addHistory 内部方法：添加操作历史
func (s *ECNService) addHistory(ctx context.Context, ecnID, userID, action string, detail map[string]interface{}) {
	history := &entity.ECNHistory{
		ID:        uuid.New().String()[:32],
		ECNID:     ecnID,
		Action:    action,
		UserID:    userID,
		Detail:    entity.JSONB(detail),
		CreatedAt: time.Now(),
	}
	s.ecnRepo.AddHistory(ctx, history)
}

// generateDefaultTasks 审批通过后自动生成执行任务
func (s *ECNService) generateDefaultTasks(ctx context.Context, ecn *entity.ECN) {
	now := time.Now()
	tasks := []entity.ECNTask{
		{
			ID:        uuid.New().String()[:32],
			ECNID:     ecn.ID,
			Type:      entity.ECNTaskTypeBOMUpdate,
			Title:     "应用BOM变更",
			Status:    entity.ECNTaskStatusPending,
			SortOrder: 1,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String()[:32],
			ECNID:     ecn.ID,
			Type:      entity.ECNTaskTypeDrawingUpdate,
			Title:     "更新相关图纸",
			Status:    entity.ECNTaskStatusPending,
			SortOrder: 2,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        uuid.New().String()[:32],
			ECNID:     ecn.ID,
			Type:      entity.ECNTaskTypeDocUpdate,
			Title:     "更新技术文档",
			Status:    entity.ECNTaskStatusPending,
			SortOrder: 3,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	// 检查SOP影响，如果有则添加SOP更新任务
	if ecn.SOPImpact != nil {
		tasks = append(tasks, entity.ECNTask{
			ID:        uuid.New().String()[:32],
			ECNID:     ecn.ID,
			Type:      entity.ECNTaskTypeSOPUpdate,
			Title:     "更新SOP文档",
			Status:    entity.ECNTaskStatusPending,
			SortOrder: 4,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	for i := range tasks {
		s.ecnRepo.CreateTask(ctx, &tasks[i])
	}
}

// ApplyBOMChanges 一键应用BOM变更（placeholder - 标记任务完成）
func (s *ECNService) ApplyBOMChanges(ctx context.Context, ecnID, userID string) error {
	ecn, err := s.ecnRepo.FindByID(ctx, ecnID)
	if err != nil {
		return fmt.Errorf("find ECN: %w", err)
	}

	if ecn.Status != entity.ECNStatusExecuting {
		return fmt.Errorf("ECN must be in executing status")
	}

	// 记录BOM应用历史
	s.addHistory(ctx, ecnID, userID, entity.ECNHistoryBOMApplied, map[string]interface{}{
		"affected_items_count": len(ecn.AffectedItems),
	})

	// 自动完成BOM更新任务
	tasks, err := s.ecnRepo.ListTasks(ctx, ecnID)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t.Type == entity.ECNTaskTypeBOMUpdate && t.Status != entity.ECNTaskStatusCompleted {
			now := time.Now()
			t.Status = entity.ECNTaskStatusCompleted
			t.CompletedAt = &now
			t.CompletedBy = &userID
			t.UpdatedAt = now
			s.ecnRepo.UpdateTask(ctx, &t)
		}
	}

	// 更新完成率
	rate, _ := s.ecnRepo.GetTaskCompletion(ctx, ecnID)
	s.ecnRepo.UpdateCompletionRate(ctx, ecnID, rate)

	if rate == 100 {
		s.ecnRepo.CloseECN(ctx, ecnID)
		s.addHistory(ctx, ecnID, userID, entity.ECNHistoryClosed, nil)
	}

	return nil
}
