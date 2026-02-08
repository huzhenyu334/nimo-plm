package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/sse"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApprovalService 审批服务
type ApprovalService struct {
	db           *gorm.DB
	feishuClient *feishu.FeishuClient
}

// NewApprovalService 创建审批服务
func NewApprovalService(db *gorm.DB, fc *feishu.FeishuClient) *ApprovalService {
	return &ApprovalService{db: db, feishuClient: fc}
}

// CreateApprovalReq 创建审批请求参数
type CreateApprovalReq struct {
	ProjectID   string   `json:"project_id" binding:"required"`
	TaskID      string   `json:"task_id" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	ReviewerIDs []string `json:"reviewer_ids" binding:"required"`
	FormData    entity.JSONB `json:"form_data"`
}

// CreateApproval 创建审批请求
func (s *ApprovalService) CreateApproval(ctx context.Context, req CreateApprovalReq, requestedBy string) (*entity.ApprovalRequest, error) {
	if len(req.ReviewerIDs) == 0 {
		return nil, fmt.Errorf("至少需要一个审批人")
	}

	approvalType := req.Type
	if approvalType == "" {
		approvalType = "task_review"
	}

	approval := &entity.ApprovalRequest{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		TaskID:      req.TaskID,
		Title:       req.Title,
		Description: req.Description,
		Type:        approvalType,
		Status:      entity.PLMApprovalStatusPending,
		FormData:    req.FormData,
		RequestedBy: requestedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 创建审批人
	var reviewers []entity.ApprovalReviewer
	for i, uid := range req.ReviewerIDs {
		reviewers = append(reviewers, entity.ApprovalReviewer{
			ID:         uuid.New().String(),
			ApprovalID: approval.ID,
			UserID:     uid,
			Status:     entity.PLMApprovalStatusPending,
			Sequence:   i,
		})
	}
	approval.Reviewers = reviewers

	// 事务：创建审批 + 审批人
	if err := s.db.WithContext(ctx).Create(approval).Error; err != nil {
		return nil, fmt.Errorf("创建审批请求失败: %w", err)
	}

	// 更新关联任务状态为 reviewing
	s.db.WithContext(ctx).Model(&entity.Task{}).
		Where("id = ?", req.TaskID).
		Update("status", entity.TaskStatusReviewing)

	// 异步发飞书通知给审批人
	if s.feishuClient != nil {
		go func() {
			bgCtx := context.Background()
			for _, reviewer := range reviewers {
				s.notifyReviewer(bgCtx, approval, reviewer.UserID)
			}
		}()
	}

	// 加载关联信息
	s.loadApprovalRelations(ctx, approval)

	return approval, nil
}

// Approve 审批通过（支持多级审批）
func (s *ApprovalService) Approve(ctx context.Context, approvalID, reviewerUserID, comment string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查找审批人记录
		var reviewer entity.ApprovalReviewer
		if err := tx.Where("approval_id = ? AND user_id = ? AND status = ?", approvalID, reviewerUserID, entity.PLMApprovalStatusPending).First(&reviewer).Error; err != nil {
			return fmt.Errorf("未找到待审批记录: %w", err)
		}

		// 更新审批人状态
		now := time.Now()
		reviewer.Status = entity.PLMApprovalStatusApproved
		reviewer.Comment = comment
		reviewer.DecidedAt = &now
		if err := tx.Save(&reviewer).Error; err != nil {
			return fmt.Errorf("更新审批人状态失败: %w", err)
		}

		// 获取审批请求
		var approval entity.ApprovalRequest
		if err := tx.Where("id = ?", approvalID).First(&approval).Error; err != nil {
			return fmt.Errorf("审批请求不存在: %w", err)
		}

		// 检查当前节点的所有审批人是否都已通过
		currentNode := approval.CurrentNode
		var pendingCount int64
		tx.Model(&entity.ApprovalReviewer{}).
			Where("approval_id = ? AND node_index = ? AND status = ?", approvalID, currentNode, entity.PLMApprovalStatusPending).
			Count(&pendingCount)

		if pendingCount > 0 {
			// 当前节点还有人未审批
			return nil
		}

		// 当前节点所有人都已通过，检查是否有下一个审批节点
		if approval.FlowSnapshot != nil && len(approval.FlowSnapshot) > 0 {
			var flowSchema entity.FlowSchema
			if err := json.Unmarshal(approval.FlowSnapshot, &flowSchema); err == nil {
				// 找下一个 approve 节点
				nextApproveIndex := -1
				for i := currentNode + 1; i < len(flowSchema.Nodes); i++ {
					if flowSchema.Nodes[i].Type == "approve" {
						nextApproveIndex = i
						break
					}
				}

				if nextApproveIndex != -1 {
					// 存在下一个审批节点，激活它
					nextNode := flowSchema.Nodes[nextApproveIndex]
					approverIDs := nextNode.Config.ApproverIDs
					if len(approverIDs) == 0 && nextNode.Config.ApproverType == "submitter" {
						approverIDs = []string{approval.RequestedBy}
					}

					if len(approverIDs) > 0 {
						// 创建下一节点的审批人记录
						for i, uid := range approverIDs {
							nextReviewer := entity.ApprovalReviewer{
								ID:         uuid.New().String(),
								ApprovalID: approvalID,
								UserID:     uid,
								Status:     entity.PLMApprovalStatusPending,
								Sequence:   i,
								NodeIndex:  nextApproveIndex,
								NodeName:   nextNode.Name,
								ReviewType: "approve",
							}
							if err := tx.Create(&nextReviewer).Error; err != nil {
								return fmt.Errorf("创建下一节点审批人失败: %w", err)
							}

							// 异步通知
							if s.feishuClient != nil {
								go s.notifyReviewer(context.Background(), &approval, uid)
							}
						}

						// 更新 current_node
						if err := tx.Model(&entity.ApprovalRequest{}).
							Where("id = ?", approvalID).
							Updates(map[string]interface{}{
								"current_node": nextApproveIndex,
								"updated_at":   now,
							}).Error; err != nil {
							return fmt.Errorf("更新当前节点失败: %w", err)
						}

						return nil // 还没结束，等下一个节点
					}
				}
			}
		}

		// 没有下一个审批节点，或者是旧的审批（无 flow_snapshot），整体通过
		if err := tx.Model(&entity.ApprovalRequest{}).
			Where("id = ?", approvalID).
			Updates(map[string]interface{}{
				"status":     entity.PLMApprovalStatusApproved,
				"result":     entity.PLMApprovalStatusApproved,
				"updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("更新审批状态失败: %w", err)
		}

		// 更新关联任务状态: reviewing → confirmed（审批通过自动确认）
		if approval.TaskID != "" {
			completedAt := time.Now()
			result := tx.Model(&entity.Task{}).
				Where("id = ? AND status = ?", approval.TaskID, entity.TaskStatusReviewing).
				Updates(map[string]interface{}{
					"status":     entity.TaskStatusConfirmed,
					"actual_end": completedAt,
					"progress":   100,
					"updated_at": completedAt,
				})
			// 审批通过且任务变为completed时，自动启动依赖此任务的后续任务
			if result.RowsAffected > 0 {
				var task entity.Task
				if err := tx.Where("id = ?", approval.TaskID).First(&task).Error; err == nil {
					s.autoStartDependentTasks(ctx, tx, task.ProjectID, task.ID)
				}
			}
		}

		// 发通知给发起人
		if s.feishuClient != nil {
			go s.notifyRequester(context.Background(), &approval, "approved", comment)
		}

		// SSE: 通知前端审批通过
		go sse.PublishTaskUpdate(approval.ProjectID, approval.TaskID, "approval_approved")

		return nil
	})
}

// Reject 审批驳回
func (s *ApprovalService) Reject(ctx context.Context, approvalID, reviewerUserID, comment string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查找审批人记录
		var reviewer entity.ApprovalReviewer
		if err := tx.Where("approval_id = ? AND user_id = ?", approvalID, reviewerUserID).First(&reviewer).Error; err != nil {
			return fmt.Errorf("未找到审批人记录: %w", err)
		}
		if reviewer.Status != entity.PLMApprovalStatusPending {
			return fmt.Errorf("该审批人已处理（当前状态: %s）", reviewer.Status)
		}

		// 更新审批人状态
		now := time.Now()
		reviewer.Status = entity.PLMApprovalStatusRejected
		reviewer.Comment = comment
		reviewer.DecidedAt = &now
		if err := tx.Save(&reviewer).Error; err != nil {
			return fmt.Errorf("更新审批人状态失败: %w", err)
		}

		// 只要有一人驳回 → 整体驳回
		if err := tx.Model(&entity.ApprovalRequest{}).
			Where("id = ?", approvalID).
			Updates(map[string]interface{}{
				"status":         entity.PLMApprovalStatusRejected,
				"result":         entity.PLMApprovalStatusRejected,
				"result_comment": comment,
				"updated_at":     now,
			}).Error; err != nil {
			return fmt.Errorf("更新审批状态失败: %w", err)
		}

		// 获取审批请求
		var approval entity.ApprovalRequest
		if err := tx.Where("id = ?", approvalID).First(&approval).Error; err == nil {
			// 更新关联任务状态: reviewing → in_progress（打回修改）
			tx.Model(&entity.Task{}).
				Where("id = ? AND status = ?", approval.TaskID, entity.TaskStatusReviewing).
				Updates(map[string]interface{}{
					"status":     entity.TaskStatusInProgress,
					"updated_at": now,
				})

			// 发通知给发起人
			if s.feishuClient != nil {
				go s.notifyRequester(context.Background(), &approval, "rejected", comment)
			}

			// SSE: 通知前端审批驳回
			go sse.PublishTaskUpdate(approval.ProjectID, approval.TaskID, "approval_rejected")
		}

		return nil
	})
}

// ListMyPending 获取我的待审批列表
func (s *ApprovalService) ListMyPending(ctx context.Context, userID string) ([]entity.ApprovalRequest, error) {
	var approvals []entity.ApprovalRequest

	// 先找到用户作为审批人且状态为 pending 的审批IDs
	var approvalIDs []string
	if err := s.db.WithContext(ctx).Model(&entity.ApprovalReviewer{}).
		Select("approval_id").
		Where("user_id = ? AND status = ?", userID, entity.PLMApprovalStatusPending).
		Find(&approvalIDs).Error; err != nil {
		return nil, err
	}

	if len(approvalIDs) == 0 {
		return []entity.ApprovalRequest{}, nil
	}

	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", approvalIDs, entity.PLMApprovalStatusPending).
		Preload("Reviewers").
		Preload("Reviewers.User").
		Preload("Requester").
		Preload("Task").
		Preload("Project").
		Order("created_at DESC").
		Find(&approvals).Error; err != nil {
		return nil, err
	}

	return approvals, nil
}

// ListApprovals 获取审批列表（可筛选）
func (s *ApprovalService) ListApprovals(ctx context.Context, status, userID string, myPending bool) ([]entity.ApprovalRequest, error) {
	if myPending && userID != "" {
		return s.ListMyPending(ctx, userID)
	}

	var approvals []entity.ApprovalRequest
	query := s.db.WithContext(ctx)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.
		Preload("Reviewers").
		Preload("Reviewers.User").
		Preload("Requester").
		Preload("Task").
		Preload("Project").
		Order("created_at DESC").
		Find(&approvals).Error; err != nil {
		return nil, err
	}

	return approvals, nil
}

// GetApproval 获取审批详情
func (s *ApprovalService) GetApproval(ctx context.Context, approvalID string) (*entity.ApprovalRequest, error) {
	var approval entity.ApprovalRequest
	if err := s.db.WithContext(ctx).
		Where("id = ?", approvalID).
		Preload("Reviewers").
		Preload("Reviewers.User").
		Preload("Requester").
		Preload("Task").
		Preload("Project").
		First(&approval).Error; err != nil {
		return nil, fmt.Errorf("审批请求不存在: %w", err)
	}
	return &approval, nil
}

// loadApprovalRelations 加载审批关联数据
func (s *ApprovalService) loadApprovalRelations(ctx context.Context, approval *entity.ApprovalRequest) {
	s.db.WithContext(ctx).
		Preload("Reviewers").
		Preload("Reviewers.User").
		Preload("Requester").
		Preload("Task").
		Preload("Project").
		First(approval, "id = ?", approval.ID)
}

// notifyReviewer 通知审批人（飞书卡片消息）
func (s *ApprovalService) notifyReviewer(ctx context.Context, approval *entity.ApprovalRequest, reviewerUserID string) {
	// 查找审批人的飞书 open_id
	var user entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", reviewerUserID).First(&user).Error; err != nil {
		log.Printf("[ApprovalNotify] 查找审批人失败 (user_id=%s): %v", reviewerUserID, err)
		return
	}
	if user.FeishuOpenID == "" {
		log.Printf("[ApprovalNotify] 审批人[%s]没有飞书 open_id，跳过通知", user.Name)
		return
	}

	// 查找发起人名字
	requesterName := "未知"
	var requester entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", approval.RequestedBy).First(&requester).Error; err == nil {
		requesterName = requester.Name
	}

	card := NewApprovalRequestCard(approval.Title, requesterName, approval.Description)
	if err := s.feishuClient.SendUserCard(ctx, user.FeishuOpenID, card); err != nil {
		log.Printf("[ApprovalNotify] 发送审批通知给[%s]失败: %v", user.Name, err)
	} else {
		log.Printf("[ApprovalNotify] 已通知审批人[%s]", user.Name)
	}
}

// notifyRequester 通知发起人审批结果
func (s *ApprovalService) notifyRequester(ctx context.Context, approval *entity.ApprovalRequest, result, comment string) {
	var requester entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", approval.RequestedBy).First(&requester).Error; err != nil {
		log.Printf("[ApprovalNotify] 查找发起人失败: %v", err)
		return
	}
	if requester.FeishuOpenID == "" {
		log.Printf("[ApprovalNotify] 发起人[%s]没有飞书 open_id，跳过通知", requester.Name)
		return
	}

	resultText := "通过"
	if result == "rejected" {
		resultText = "驳回"
	}

	card := feishu.NewReviewResultCard(approval.Title, resultText, comment)
	if err := s.feishuClient.SendUserCard(ctx, requester.FeishuOpenID, card); err != nil {
		log.Printf("[ApprovalNotify] 发送审批结果通知给[%s]失败: %v", requester.Name, err)
	} else {
		log.Printf("[ApprovalNotify] 已通知发起人[%s]审批结果: %s", requester.Name, resultText)
	}
}

// NewApprovalRequestCard 创建审批请求通知卡片
func NewApprovalRequestCard(title, requesterName, description string) feishu.InteractiveCard {
	elements := []feishu.CardElement{
		{
			Tag: "div",
			Fields: []feishu.CardField{
				{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**审批标题**\n%s", title)}},
				{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**发起人**\n%s", requesterName)}},
			},
		},
	}

	if description != "" {
		elements = append(elements,
			feishu.CardElement{
				Tag:  "div",
				Text: &feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**审批说明**\n%s", description)},
			},
		)
	}

	elements = append(elements,
		feishu.CardElement{Tag: "hr"},
		feishu.CardElement{
			Tag: "note",
			Elements: []feishu.CardElement{
				{Tag: "plain_text", Content: "请登录 PLM 系统处理此审批请求"},
			},
		},
	)

	return feishu.InteractiveCard{
		Config: &feishu.CardConfig{WideScreenMode: true},
		Header: &feishu.CardHeader{
			Title:    feishu.CardText{Tag: "plain_text", Content: "📋 新审批请求"},
			Template: "orange",
		},
		Elements: elements,
	}
}

// autoStartDependentTasks 审批通过后自动启动依赖任务
func (s *ApprovalService) autoStartDependentTasks(ctx context.Context, db *gorm.DB, projectID, completedTaskID string) {
	var deps []entity.TaskDependency
	if err := db.Where("depends_on_task_id = ?", completedTaskID).Find(&deps).Error; err != nil {
		log.Printf("[ApprovalService] 查找依赖任务失败: %v", err)
		return
	}

	for _, dep := range deps {
		var task entity.Task
		if err := db.Where("id = ?", dep.TaskID).First(&task).Error; err != nil {
			continue
		}
		if task.Status != entity.TaskStatusPending {
			continue
		}

		// 检查该任务的所有前置依赖是否都已完成
		allCompleted := true
		var allDeps []entity.TaskDependency
		if err := db.Where("task_id = ?", task.ID).Find(&allDeps).Error; err != nil {
			continue
		}
		for _, d := range allDeps {
			var depTask entity.Task
			if err := db.Where("id = ?", d.DependsOnID).First(&depTask).Error; err != nil {
				allCompleted = false
				break
			}
			if depTask.Status != entity.TaskStatusCompleted && depTask.Status != entity.TaskStatusConfirmed {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			now := time.Now()
			task.Status = entity.TaskStatusInProgress
			task.ActualStart = &now
			if err := db.Save(&task).Error; err != nil {
				log.Printf("[ApprovalService] 自动启动任务失败 (task=%s): %v", task.ID, err)
				continue
			}
			log.Printf("[ApprovalService] 审批通过后自动启动任务 task=%s (依赖任务 %s 完成)", task.ID, completedTaskID)
		}
	}
}
