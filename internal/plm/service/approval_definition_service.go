package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalDefinitionService 审批定义服务
type ApprovalDefinitionService struct {
	db           *gorm.DB
	feishuClient *feishu.FeishuClient
	approvalSvc  *ApprovalService
}

// NewApprovalDefinitionService 创建审批定义服务
func NewApprovalDefinitionService(db *gorm.DB, fc *feishu.FeishuClient, approvalSvc *ApprovalService) *ApprovalDefinitionService {
	return &ApprovalDefinitionService{db: db, feishuClient: fc, approvalSvc: approvalSvc}
}

// CreateDefinitionReq 创建审批定义请求
type CreateDefinitionReq struct {
	Code        string          `json:"code" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	GroupName   string          `json:"group_name"`
	FormSchema  json.RawMessage `json:"form_schema"`
	FlowSchema  json.RawMessage `json:"flow_schema"`
	Visibility  string          `json:"visibility"`
	AdminUserID string          `json:"admin_user_id"`
	SortOrder   int             `json:"sort_order"`
}

// UpdateDefinitionReq 更新审批定义请求
type UpdateDefinitionReq struct {
	Name        *string          `json:"name"`
	Description *string          `json:"description"`
	Icon        *string          `json:"icon"`
	GroupName   *string          `json:"group_name"`
	FormSchema  json.RawMessage  `json:"form_schema"`
	FlowSchema  json.RawMessage  `json:"flow_schema"`
	Visibility  *string          `json:"visibility"`
	AdminUserID *string          `json:"admin_user_id"`
	SortOrder   *int             `json:"sort_order"`
}

// DefinitionGroup 定义分组
type DefinitionGroup struct {
	GroupName   string                      `json:"group_name"`
	Definitions []entity.ApprovalDefinition `json:"definitions"`
}

// CreateInstanceReq 从定义发起审批请求
type CreateInstanceReq struct {
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	ProjectID         string                 `json:"project_id"`
	TaskID            string                 `json:"task_id"`
	FormData          map[string]interface{} `json:"form_data"`
	SelectedApprovers map[string][]string    `json:"selected_approvers"` // node_index(string) -> user_ids (for self_select nodes)
}

// Create 创建审批定义
func (s *ApprovalDefinitionService) Create(ctx context.Context, req CreateDefinitionReq, userID string) (*entity.ApprovalDefinition, error) {
	now := time.Now()

	icon := req.Icon
	if icon == "" {
		icon = "approval"
	}
	groupName := req.GroupName
	if groupName == "" {
		groupName = "其他"
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = "全员"
	}
	formSchema := req.FormSchema
	if formSchema == nil {
		formSchema = json.RawMessage(`[]`)
	}
	flowSchema := req.FlowSchema
	if flowSchema == nil {
		flowSchema = json.RawMessage(`{"nodes":[]}`)
	}

	def := &entity.ApprovalDefinition{
		ID:          uuid.New().String(),
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Icon:        icon,
		GroupName:   groupName,
		FormSchema:  formSchema,
		FlowSchema:  flowSchema,
		Visibility:  visibility,
		Status:      entity.ApprovalDefStatusDraft,
		AdminUserID: req.AdminUserID,
		SortOrder:   req.SortOrder,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.db.WithContext(ctx).Create(def).Error; err != nil {
		return nil, fmt.Errorf("创建审批定义失败: %w", err)
	}

	return def, nil
}

// Update 更新审批定义
func (s *ApprovalDefinitionService) Update(ctx context.Context, id string, req UpdateDefinitionReq) (*entity.ApprovalDefinition, error) {
	var def entity.ApprovalDefinition
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&def).Error; err != nil {
		return nil, fmt.Errorf("审批定义不存在: %w", err)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.GroupName != nil {
		updates["group_name"] = *req.GroupName
	}
	if req.FormSchema != nil {
		updates["form_schema"] = req.FormSchema
	}
	if req.FlowSchema != nil {
		updates["flow_schema"] = req.FlowSchema
	}
	if req.Visibility != nil {
		updates["visibility"] = *req.Visibility
	}
	if req.AdminUserID != nil {
		updates["admin_user_id"] = *req.AdminUserID
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if err := s.db.WithContext(ctx).Model(&def).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新审批定义失败: %w", err)
	}

	// Reload
	s.db.WithContext(ctx).Where("id = ?", id).First(&def)
	return &def, nil
}

// Get 获取审批定义
func (s *ApprovalDefinitionService) Get(ctx context.Context, id string) (*entity.ApprovalDefinition, error) {
	var def entity.ApprovalDefinition
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&def).Error; err != nil {
		return nil, fmt.Errorf("审批定义不存在: %w", err)
	}
	return &def, nil
}

// List 获取审批定义列表（按分组）
func (s *ApprovalDefinitionService) List(ctx context.Context) ([]DefinitionGroup, error) {
	var defs []entity.ApprovalDefinition
	if err := s.db.WithContext(ctx).Order("sort_order ASC, created_at ASC").Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("获取审批定义列表失败: %w", err)
	}

	// Group by group_name
	groupMap := make(map[string][]entity.ApprovalDefinition)
	groupOrder := []string{}
	for _, d := range defs {
		if _, exists := groupMap[d.GroupName]; !exists {
			groupOrder = append(groupOrder, d.GroupName)
		}
		groupMap[d.GroupName] = append(groupMap[d.GroupName], d)
	}

	var result []DefinitionGroup
	for _, gn := range groupOrder {
		result = append(result, DefinitionGroup{
			GroupName:   gn,
			Definitions: groupMap[gn],
		})
	}

	return result, nil
}

// Delete 删除审批定义
func (s *ApprovalDefinitionService) Delete(ctx context.Context, id string) error {
	var def entity.ApprovalDefinition
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&def).Error; err != nil {
		return fmt.Errorf("审批定义不存在: %w", err)
	}
	if def.Status == entity.ApprovalDefStatusPublished {
		return fmt.Errorf("已发布的审批定义不能删除，请先取消发布")
	}
	if err := s.db.WithContext(ctx).Delete(&def).Error; err != nil {
		return fmt.Errorf("删除审批定义失败: %w", err)
	}
	return nil
}

// Publish 发布审批定义
func (s *ApprovalDefinitionService) Publish(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&entity.ApprovalDefinition{}).
		Where("id = ? AND status = ?", id, entity.ApprovalDefStatusDraft).
		Update("status", entity.ApprovalDefStatusPublished)
	if result.Error != nil {
		return fmt.Errorf("发布失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("审批定义不存在或已发布")
	}
	return nil
}

// Unpublish 取消发布审批定义
func (s *ApprovalDefinitionService) Unpublish(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&entity.ApprovalDefinition{}).
		Where("id = ? AND status = ?", id, entity.ApprovalDefStatusPublished).
		Update("status", entity.ApprovalDefStatusDraft)
	if result.Error != nil {
		return fmt.Errorf("取消发布失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("审批定义不存在或未发布")
	}
	return nil
}

// ListGroups 获取审批分组列表
func (s *ApprovalDefinitionService) ListGroups(ctx context.Context) ([]entity.ApprovalGroup, error) {
	var groups []entity.ApprovalGroup
	if err := s.db.WithContext(ctx).Order("sort_order ASC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("获取分组列表失败: %w", err)
	}
	return groups, nil
}

// CreateGroup 创建审批分组
func (s *ApprovalDefinitionService) CreateGroup(ctx context.Context, name string) (*entity.ApprovalGroup, error) {
	group := &entity.ApprovalGroup{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(group).Error; err != nil {
		return nil, fmt.Errorf("创建分组失败: %w", err)
	}
	return group, nil
}

// DeleteGroup 删除审批分组
func (s *ApprovalDefinitionService) DeleteGroup(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.ApprovalGroup{}).Error; err != nil {
		return fmt.Errorf("删除分组失败: %w", err)
	}
	return nil
}

// CreateInstance 从审批定义发起审批实例
func (s *ApprovalDefinitionService) CreateInstance(ctx context.Context, definitionID string, req CreateInstanceReq, submitterID string) (*entity.ApprovalRequest, error) {
	// 1. 获取审批定义
	var def entity.ApprovalDefinition
	if err := s.db.WithContext(ctx).Where("id = ?", definitionID).First(&def).Error; err != nil {
		return nil, fmt.Errorf("审批定义不存在: %w", err)
	}
	if def.Status != entity.ApprovalDefStatusPublished {
		return nil, fmt.Errorf("审批定义未发布，无法发起")
	}

	// 2. 解析 flow_schema
	var flowSchema entity.FlowSchema
	if err := json.Unmarshal(def.FlowSchema, &flowSchema); err != nil {
		return nil, fmt.Errorf("解析流程定义失败: %w", err)
	}

	// 3. 找到第一个 approve 节点
	firstApproveIndex := -1
	for i, node := range flowSchema.Nodes {
		if node.Type == "approve" {
			firstApproveIndex = i
			break
		}
	}
	if firstApproveIndex == -1 {
		return nil, fmt.Errorf("流程定义中没有审批节点")
	}

	// 4. 确定第一个审批节点的审批人
	firstNode := flowSchema.Nodes[firstApproveIndex]
	approverIDs, err := s.resolveApprovers(firstNode, submitterID, req.SelectedApprovers, firstApproveIndex)
	if err != nil {
		return nil, fmt.Errorf("确定审批人失败: %w", err)
	}
	if len(approverIDs) == 0 {
		return nil, fmt.Errorf("第一个审批节点没有审批人")
	}

	// 5. 构建 form_data
	formDataJSON, _ := json.Marshal(req.FormData)
	var formData entity.JSONB
	json.Unmarshal(formDataJSON, &formData)

	title := req.Title
	if title == "" {
		title = def.Name
	}

	now := time.Now()
	approval := &entity.ApprovalRequest{
		ID:           uuid.New().String(),
		ProjectID:    req.ProjectID,
		TaskID:       req.TaskID,
		Title:        title,
		Description:  req.Description,
		Type:         "definition",
		Status:       entity.PLMApprovalStatusPending,
		FormData:     formData,
		RequestedBy:  submitterID,
		DefinitionID: definitionID,
		Code:         def.Code,
		CurrentNode:  firstApproveIndex,
		FlowSnapshot: def.FlowSchema,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 6. 创建审批人记录
	var reviewers []entity.ApprovalReviewer
	for i, uid := range approverIDs {
		reviewers = append(reviewers, entity.ApprovalReviewer{
			ID:         uuid.New().String(),
			ApprovalID: approval.ID,
			UserID:     uid,
			Status:     entity.PLMApprovalStatusPending,
			Sequence:   i,
			NodeIndex:  firstApproveIndex,
			NodeName:   firstNode.Name,
			ReviewType: "approve",
		})
	}
	approval.Reviewers = reviewers

	// 7. 保存到数据库
	if err := s.db.WithContext(ctx).Create(approval).Error; err != nil {
		return nil, fmt.Errorf("创建审批实例失败: %w", err)
	}

	// 8. 发通知给第一批审批人
	if s.feishuClient != nil {
		go func() {
			bgCtx := context.Background()
			for _, reviewer := range reviewers {
				s.notifyReviewer(bgCtx, approval, reviewer.UserID)
			}
		}()
	}

	// 9. 加载关联
	s.db.WithContext(ctx).
		Preload("Reviewers").
		Preload("Reviewers.User").
		Preload("Requester").
		First(approval, "id = ?", approval.ID)

	return approval, nil
}

// resolveApprovers 根据节点配置确定审批人
func (s *ApprovalDefinitionService) resolveApprovers(node entity.FlowNode, submitterID string, selectedApprovers map[string][]string, nodeIndex int) ([]string, error) {
	switch node.Config.ApproverType {
	case "designated":
		return node.Config.ApproverIDs, nil
	case "self_select":
		key := fmt.Sprintf("%d", nodeIndex)
		if approvers, ok := selectedApprovers[key]; ok && len(approvers) > 0 {
			return approvers, nil
		}
		return nil, fmt.Errorf("自选审批人节点[%s]未指定审批人", node.Name)
	case "submitter":
		return []string{submitterID}, nil
	default:
		// For supervisor, dept_leader, role — fallback to designated if approver_ids provided
		if len(node.Config.ApproverIDs) > 0 {
			return node.Config.ApproverIDs, nil
		}
		return nil, fmt.Errorf("审批人类型[%s]暂不支持自动解析，请指定审批人", node.Config.ApproverType)
	}
}

// notifyReviewer 通知审批人
func (s *ApprovalDefinitionService) notifyReviewer(ctx context.Context, approval *entity.ApprovalRequest, reviewerUserID string) {
	var user entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", reviewerUserID).First(&user).Error; err != nil {
		log.Printf("[ApprovalDefNotify] 查找审批人失败 (user_id=%s): %v", reviewerUserID, err)
		return
	}
	if user.FeishuOpenID == "" {
		log.Printf("[ApprovalDefNotify] 审批人[%s]没有飞书open_id，跳过通知", user.Name)
		return
	}

	requesterName := "未知"
	var requester entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", approval.RequestedBy).First(&requester).Error; err == nil {
		requesterName = requester.Name
	}

	card := NewApprovalRequestCard(approval.Title, requesterName, approval.Description)
	if err := s.feishuClient.SendUserCard(ctx, user.FeishuOpenID, card); err != nil {
		log.Printf("[ApprovalDefNotify] 发送通知给[%s]失败: %v", user.Name, err)
	} else {
		log.Printf("[ApprovalDefNotify] 已通知审批人[%s]", user.Name)
	}
}
