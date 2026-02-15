package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/bitfantasy/nimo/internal/shared/engine"
	"github.com/bitfantasy/nimo/internal/shared/feishu"
	srmsvc "github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleAssignment 角色指派信息
type RoleAssignment struct {
	RoleCode     string `json:"role_code"`
	UserID       string `json:"user_id"`
	FeishuUserID string `json:"feishu_user_id"`
}

// WorkflowService 工作流服务 —— 连接状态机引擎和飞书集成
type WorkflowService struct {
	db                  *gorm.DB
	engine              *engine.Engine
	feishuClient        *feishu.FeishuClient
	projectRepo         *repository.ProjectRepository
	taskRepo            *repository.TaskRepository
	routingService      *RoutingService
	taskFormRepo        *repository.TaskFormRepository
	srmProcurementSvc   *srmsvc.ProcurementService
	bomRepo             *repository.ProjectBOMRepository
}

// NewWorkflowService 创建工作流服务
func NewWorkflowService(db *gorm.DB, eng *engine.Engine, fc *feishu.FeishuClient, projectRepo *repository.ProjectRepository, taskRepo *repository.TaskRepository) *WorkflowService {
	return &WorkflowService{
		db:           db,
		engine:       eng,
		feishuClient: fc,
		projectRepo:  projectRepo,
		taskRepo:     taskRepo,
	}
}

// SetRoutingService 注入路由服务
func (s *WorkflowService) SetRoutingService(rs *RoutingService) {
	s.routingService = rs
}

// SetTaskFormRepo 注入任务表单仓库
func (s *WorkflowService) SetTaskFormRepo(repo *repository.TaskFormRepository) {
	s.taskFormRepo = repo
}

// SetSRMProcurementService 注入SRM采购服务
func (s *WorkflowService) SetSRMProcurementService(svc *srmsvc.ProcurementService) {
	s.srmProcurementSvc = svc
}

// SetBOMRepo 注入BOM仓库
func (s *WorkflowService) SetBOMRepo(repo *repository.ProjectBOMRepository) {
	s.bomRepo = repo
}

// AssignTask 指派任务
// 把任务状态从 unassigned → pending，记录操作日志，可选创建飞书任务
func (s *WorkflowService) AssignTask(ctx context.Context, projectID, taskID, assigneeID, feishuUserID, operatorID string) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("查找任务失败: %w", err)
	}
	if task.ProjectID != projectID {
		return fmt.Errorf("任务不属于该项目")
	}

	// 允许从 unassigned 或 pending 指派/重新指派
	if task.Status != entity.TaskStatusUnassigned && task.Status != entity.TaskStatusPending {
		return fmt.Errorf("任务当前状态[%s]不允许指派，需要处于 unassigned 或 pending 状态", task.Status)
	}

	fromStatus := task.Status

	// 更新任务
	task.AssigneeID = &assigneeID
	task.Status = entity.TaskStatusPending
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务失败: %w", err)
	}

	// 记录操作日志
	s.logAction(ctx, projectID, taskID, entity.TaskActionAssign, fromStatus, entity.TaskStatusPending, operatorID, map[string]interface{}{
		"assignee_id":   assigneeID,
		"feishu_user_id": feishuUserID,
	}, "")

	// 异步创建飞书任务（不阻断主流程）
	if s.feishuClient != nil && task.AutoCreateFeishuTask && feishuUserID != "" {
		go func() {
			bgCtx := context.Background()
			taskGUID, err := s.feishuClient.CreateTask(bgCtx, feishu.CreateTaskReq{
				Summary:     task.Title,
				Description: task.Description,
				Members: []feishu.TaskMember{
					{ID: feishuUserID, Role: "assignee"},
				},
			})
			if err != nil {
				log.Printf("[WorkflowService] 飞书任务创建失败 (task=%s): %v", taskID, err)
				return
			}
			// 保存飞书任务ID到task
			s.db.WithContext(bgCtx).Model(&entity.Task{}).Where("id = ?", taskID).Update("feishu_task_id", taskGUID)
			log.Printf("[WorkflowService] 飞书任务创建成功 task=%s feishu_guid=%s", taskID, taskGUID)
		}()
	}

	// 异步发飞书卡片通知给被指派人
	if s.feishuClient != nil {
		go s.notifyTaskAssigned(context.Background(), task, assigneeID, projectID)
	}

	return nil
}

// StartTask 开始任务
// 检查前置依赖是否完成，状态 pending → in_progress
func (s *WorkflowService) StartTask(ctx context.Context, projectID, taskID, operatorID string) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("查找任务失败: %w", err)
	}
	if task.ProjectID != projectID {
		return fmt.Errorf("任务不属于该项目")
	}
	if task.Status != entity.TaskStatusPending {
		return fmt.Errorf("任务当前状态[%s]不允许启动，需要处于 pending 状态", task.Status)
	}

	// 检查前置依赖是否全部完成
	if err := s.checkDependenciesCompleted(ctx, taskID); err != nil {
		return err
	}

	// 更新状态
	now := time.Now()
	task.Status = entity.TaskStatusInProgress
	task.ActualStart = &now
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务失败: %w", err)
	}

	// 记录操作日志
	s.logAction(ctx, projectID, taskID, entity.TaskActionStart, entity.TaskStatusPending, entity.TaskStatusInProgress, operatorID, nil, "")

	// Hook: 检测 procurement_control 字段，自动创建采购需求
	go s.handleProcurementControl(context.Background(), task, operatorID)

	return nil
}

// CompleteTask 完成任务
// 如果 requires_approval → reviewing，否则 → completed
func (s *WorkflowService) CompleteTask(ctx context.Context, projectID, taskID, operatorID string) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("查找任务失败: %w", err)
	}
	if task.ProjectID != projectID {
		return fmt.Errorf("任务不属于该项目")
	}
	if task.Status != entity.TaskStatusInProgress {
		return fmt.Errorf("任务当前状态[%s]不允许完成，需要处于 in_progress 状态", task.Status)
	}

	if task.RequiresApproval {
		// 智能路由：判断走 agent 自动审批还是人工审批
		if s.routingService != nil {
			routeCtx := map[string]interface{}{
				"project_id": projectID,
				"task_id":    taskID,
				"task_code":  task.Code,
				"task_type":  task.TaskType,
			}
			decision, err := s.routingService.EvaluateRoute(ctx, "plm_task", "task_complete", routeCtx)
			if err != nil {
				log.Printf("[WorkflowService] 路由评估失败，走默认人工审批: %v", err)
			} else if decision.Channel == entity.RoutingChannelAgent {
				// Agent 自动审批 → 直接 completed
				now := time.Now()
				task.Status = entity.TaskStatusCompleted
				task.CompletedAt = &now
				task.Progress = 100
				if err := s.taskRepo.Update(ctx, task); err != nil {
					return fmt.Errorf("更新任务失败: %w", err)
				}
				s.logAction(ctx, projectID, taskID, entity.TaskActionApprove, entity.TaskStatusInProgress, entity.TaskStatusCompleted, "agent", map[string]interface{}{
					"routing_rule_id":   decision.RuleID,
					"routing_rule_name": decision.RuleName,
					"routing_reason":    decision.Reason,
				}, "智能路由: Agent自动审批通过")

				s.checkAndStartDependentTasks(ctx, projectID, taskID)

				if s.feishuClient != nil && task.FeishuTaskID != "" {
					go func() {
						if err := s.feishuClient.CompleteTask(context.Background(), task.FeishuTaskID); err != nil {
							log.Printf("[WorkflowService] 飞书任务完成失败 (feishu_task=%s): %v", task.FeishuTaskID, err)
						}
					}()
				}
				return nil
			}
		}

		// 需要审批 → reviewing（人工审批流程）
		task.Status = entity.TaskStatusReviewing
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("更新任务失败: %w", err)
		}
		s.logAction(ctx, projectID, taskID, entity.TaskActionSubmitReview, entity.TaskStatusInProgress, entity.TaskStatusReviewing, operatorID, nil, "")
	} else {
		// 不需审批 → completed
		now := time.Now()
		task.Status = entity.TaskStatusCompleted
		task.CompletedAt = &now
		task.Progress = 100
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("更新任务失败: %w", err)
		}
		s.logAction(ctx, projectID, taskID, entity.TaskActionComplete, entity.TaskStatusInProgress, entity.TaskStatusCompleted, operatorID, nil, "")

		// 检查并启动依赖任务
		s.checkAndStartDependentTasks(ctx, projectID, taskID)

		// 异步完成飞书任务
		if s.feishuClient != nil && task.FeishuTaskID != "" {
			go func() {
				if err := s.feishuClient.CompleteTask(context.Background(), task.FeishuTaskID); err != nil {
					log.Printf("[WorkflowService] 飞书任务完成失败 (feishu_task=%s): %v", task.FeishuTaskID, err)
				}
			}()
		}
	}

	return nil
}

// SubmitReview 提交评审结果
func (s *WorkflowService) SubmitReview(ctx context.Context, projectID, taskID, outcomeCode, comment, operatorID string) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("查找任务失败: %w", err)
	}
	if task.ProjectID != projectID {
		return fmt.Errorf("任务不属于该项目")
	}
	if task.Status != entity.TaskStatusReviewing {
		return fmt.Errorf("任务当前状态[%s]不允许评审，需要处于 reviewing 状态", task.Status)
	}

	// 查找评审结果配置
	var outcome entity.TemplateTaskOutcome
	outcomeFound := false

	// 获取项目模板ID
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err == nil && project.TemplateID != nil {
		if err := s.db.WithContext(ctx).
			Where("template_id = ? AND task_code = ? AND outcome_code = ?", *project.TemplateID, task.Code, outcomeCode).
			First(&outcome).Error; err == nil {
			outcomeFound = true
		}
	}

	// 根据结果类型处理
	if outcomeFound && outcome.OutcomeType == "fail_rollback" {
		// 评审不通过，需要回退
		task.Status = entity.TaskStatusRejected
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("更新任务失败: %w", err)
		}
		s.logAction(ctx, projectID, taskID, entity.TaskActionReject, entity.TaskStatusReviewing, entity.TaskStatusRejected, operatorID, map[string]interface{}{
			"outcome_code": outcomeCode,
		}, comment)

		// 执行回退
		if outcome.RollbackToTaskCode != "" {
			if err := s.RollbackTask(ctx, projectID, taskID, outcome.RollbackToTaskCode, outcome.RollbackCascade, operatorID); err != nil {
				log.Printf("[WorkflowService] 回退失败 (task=%s rollback_to=%s): %v", taskID, outcome.RollbackToTaskCode, err)
			}
		}
	} else if outcomeCode == "reject" || outcomeCode == "rejected" {
		// 简单驳回逻辑（无模板配置时）
		task.Status = entity.TaskStatusInProgress
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("更新任务失败: %w", err)
		}
		s.logAction(ctx, projectID, taskID, entity.TaskActionReject, entity.TaskStatusReviewing, entity.TaskStatusInProgress, operatorID, map[string]interface{}{
			"outcome_code": outcomeCode,
		}, comment)
	} else {
		// 审批通过 → completed
		now := time.Now()
		task.Status = entity.TaskStatusCompleted
		task.CompletedAt = &now
		task.Progress = 100
		if err := s.taskRepo.Update(ctx, task); err != nil {
			return fmt.Errorf("更新任务失败: %w", err)
		}
		s.logAction(ctx, projectID, taskID, entity.TaskActionApprove, entity.TaskStatusReviewing, entity.TaskStatusCompleted, operatorID, map[string]interface{}{
			"outcome_code": outcomeCode,
		}, comment)

		// 检查并启动依赖任务
		s.checkAndStartDependentTasks(ctx, projectID, taskID)

		// 异步完成飞书任务
		if s.feishuClient != nil && task.FeishuTaskID != "" {
			go func() {
				if err := s.feishuClient.CompleteTask(context.Background(), task.FeishuTaskID); err != nil {
					log.Printf("[WorkflowService] 飞书任务完成失败 (feishu_task=%s): %v", task.FeishuTaskID, err)
				}
			}()
		}
	}

	return nil
}

// RollbackTask 回退任务
func (s *WorkflowService) RollbackTask(ctx context.Context, projectID, taskID, rollbackToTaskCode string, cascade bool, operatorID string) error {
	// 查找目标任务
	var targetTask entity.Task
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND code = ?", projectID, rollbackToTaskCode).
		First(&targetTask).Error; err != nil {
		return fmt.Errorf("查找回退目标任务失败 (code=%s): %w", rollbackToTaskCode, err)
	}

	// 重置目标任务为 in_progress
	fromStatus := targetTask.Status
	targetTask.Status = entity.TaskStatusInProgress
	targetTask.CompletedAt = nil
	targetTask.Progress = 0
	if err := s.db.WithContext(ctx).Save(&targetTask).Error; err != nil {
		return fmt.Errorf("重置目标任务失败: %w", err)
	}
	s.logAction(ctx, projectID, targetTask.ID, entity.TaskActionRollback, fromStatus, entity.TaskStatusInProgress, operatorID, map[string]interface{}{
		"triggered_by_task": taskID,
		"cascade":           cascade,
	}, "")

	if cascade {
		// 获取目标任务所在阶段的后续任务
		var subsequentTasks []entity.Task
		if err := s.db.WithContext(ctx).
			Where("project_id = ? AND phase_id = ? AND sequence > ? AND id != ?",
				projectID, targetTask.PhaseID, targetTask.Sequence, targetTask.ID).
			Find(&subsequentTasks).Error; err != nil {
			log.Printf("[WorkflowService] 查找后续任务失败: %v", err)
			return nil
		}

		for _, t := range subsequentTasks {
			if t.Status == entity.TaskStatusCompleted || t.Status == entity.TaskStatusInProgress || t.Status == entity.TaskStatusReviewing {
				oldStatus := t.Status
				t.Status = entity.TaskStatusPending
				t.CompletedAt = nil
				t.Progress = 0
				if err := s.db.WithContext(ctx).Save(&t).Error; err != nil {
					log.Printf("[WorkflowService] 重置后续任务失败 (task=%s): %v", t.ID, err)
					continue
				}
				s.logAction(ctx, projectID, t.ID, entity.TaskActionRollback, oldStatus, entity.TaskStatusPending, operatorID, map[string]interface{}{
					"triggered_by_task": taskID,
					"cascade":           true,
				}, "级联回退")
			}
		}
	}

	return nil
}

// AssignPhaseRoles 指派阶段角色
func (s *WorkflowService) AssignPhaseRoles(ctx context.Context, projectID, phase string, assignments []RoleAssignment, operatorID string) error {
	for _, a := range assignments {
		// 保存角色指派（upsert）
		assignment := entity.ProjectRoleAssignment{
			ID:           uuid.New().String(),
			ProjectID:    projectID,
			Phase:        phase,
			RoleCode:     a.RoleCode,
			UserID:       a.UserID,
			FeishuUserID: a.FeishuUserID,
			AssignedBy:   operatorID,
			AssignedAt:   time.Now(),
		}

		result := s.db.WithContext(ctx).
			Where("project_id = ? AND phase = ? AND role_code = ?", projectID, phase, a.RoleCode).
			Assign(map[string]interface{}{
				"user_id":        a.UserID,
				"feishu_user_id": a.FeishuUserID,
				"assigned_by":    operatorID,
				"assigned_at":    time.Now(),
			}).
			FirstOrCreate(&assignment)
		if result.Error != nil {
			return fmt.Errorf("保存角色指派失败 (role=%s): %w", a.RoleCode, result.Error)
		}

		// 查找该阶段中默认角色匹配的未指派任务
		var unassignedTasks []entity.Task
		s.db.WithContext(ctx).
			Joins("JOIN project_phases ON tasks.phase_id = project_phases.id").
			Where("tasks.project_id = ? AND project_phases.phase = ? AND (tasks.assignee_id IS NULL OR tasks.assignee_id = '') AND tasks.status = ?",
				projectID, phase, entity.TaskStatusUnassigned).
			Find(&unassignedTasks)

		// 还需要检查模板任务的 default_assignee_role
		for _, task := range unassignedTasks {
			// 查模板任务的默认角色
			var templateTask entity.TemplateTask
			if err := s.db.WithContext(ctx).
				Where("task_code = ? AND default_assignee_role = ?", task.Code, a.RoleCode).
				First(&templateTask).Error; err != nil {
				continue // 不匹配则跳过
			}

			// 指派任务
			if err := s.AssignTask(ctx, projectID, task.ID, a.UserID, a.FeishuUserID, operatorID); err != nil {
				log.Printf("[WorkflowService] 自动指派任务失败 (task=%s role=%s): %v", task.ID, a.RoleCode, err)
			}
		}
	}

	return nil
}

// GetTaskHistory 获取任务操作历史
func (s *WorkflowService) GetTaskHistory(ctx context.Context, projectID, taskID string) ([]entity.TaskActionLog, error) {
	var logs []entity.TaskActionLog
	err := s.db.WithContext(ctx).
		Where("project_id = ? AND task_id = ?", projectID, taskID).
		Order("created_at DESC").
		Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("查询操作历史失败: %w", err)
	}
	return logs, nil
}

// checkDependenciesCompleted 检查任务的所有前置依赖是否已完成
func (s *WorkflowService) checkDependenciesCompleted(ctx context.Context, taskID string) error {
	var deps []entity.TaskDependency
	if err := s.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&deps).Error; err != nil {
		return fmt.Errorf("查询任务依赖失败: %w", err)
	}

	for _, dep := range deps {
		var depTask entity.Task
		if err := s.db.WithContext(ctx).Where("id = ?", dep.DependsOnID).First(&depTask).Error; err != nil {
			return fmt.Errorf("查找依赖任务失败 (id=%s): %w", dep.DependsOnID, err)
		}
		if depTask.Status != entity.TaskStatusCompleted {
			return fmt.Errorf("前置任务[%s]尚未完成（当前状态: %s），无法启动", depTask.Title, depTask.Status)
		}
	}

	return nil
}

// checkAndStartDependentTasks 检查并启动依赖当前任务的后续任务
func (s *WorkflowService) checkAndStartDependentTasks(ctx context.Context, projectID, completedTaskID string) {
	// 查找依赖于已完成任务的任务
	var deps []entity.TaskDependency
	if err := s.db.WithContext(ctx).Where("depends_on_task_id = ?", completedTaskID).Find(&deps).Error; err != nil {
		log.Printf("[WorkflowService] 查找依赖任务失败: %v", err)
		return
	}

	for _, dep := range deps {
		// 获取依赖任务
		var task entity.Task
		if err := s.db.WithContext(ctx).Where("id = ?", dep.TaskID).First(&task).Error; err != nil {
			log.Printf("[WorkflowService] 查找任务失败 (id=%s): %v", dep.TaskID, err)
			continue
		}

		// 只处理 pending 状态的任务
		if task.Status != entity.TaskStatusPending {
			continue
		}

		// 检查该任务的所有依赖是否都已完成
		allCompleted := true
		var allDeps []entity.TaskDependency
		if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).Find(&allDeps).Error; err != nil {
			log.Printf("[WorkflowService] 查找任务所有依赖失败: %v", err)
			continue
		}

		for _, d := range allDeps {
			var depTask entity.Task
			if err := s.db.WithContext(ctx).Where("id = ?", d.DependsOnID).First(&depTask).Error; err != nil {
				allCompleted = false
				break
			}
			if depTask.Status != entity.TaskStatusCompleted {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			// 自动启动任务
			now := time.Now()
			task.Status = entity.TaskStatusInProgress
			task.ActualStart = &now
			if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
				log.Printf("[WorkflowService] 自动启动任务失败 (task=%s): %v", task.ID, err)
				continue
			}
			s.logAction(ctx, projectID, task.ID, entity.TaskActionStart, entity.TaskStatusPending, entity.TaskStatusInProgress, "system", map[string]interface{}{
				"auto_started":       true,
				"completed_dep_task": completedTaskID,
			}, "依赖任务完成，自动启动")
			log.Printf("[WorkflowService] 自动启动任务 task=%s (依赖任务 %s 完成)", task.ID, completedTaskID)

			// Hook: 自动启动时也触发采购控件
			go s.handleProcurementControl(context.Background(), &task, "system")
		}
	}
}

// =============================================================================
// 采购控件处理
// =============================================================================

// TriggerProcurementControl 公开方法：按需触发采购控件数据拉取
func (s *WorkflowService) TriggerProcurementControl(ctx context.Context, taskID string) {
	if s.taskFormRepo == nil || s.srmProcurementSvc == nil {
		return
	}
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil || task == nil {
		return
	}
	s.handleProcurementControl(ctx, task, "system")
}

// handleProcurementControl 处理采购控件：从源任务提取物料清单，创建SRM采购需求
func (s *WorkflowService) handleProcurementControl(ctx context.Context, task *entity.Task, operatorID string) {
	if s.taskFormRepo == nil || s.srmProcurementSvc == nil {
		return
	}

	// 1. 获取当前任务的表单定义
	formDef, err := s.taskFormRepo.FindByTaskID(ctx, task.ID)
	if err != nil || formDef == nil {
		return
	}

	// 2. 解析字段定义，找到 procurement_control 类型的字段
	type formFieldDef struct {
		Key               string   `json:"key"`
		Type              string   `json:"type"`
		SourceTaskCode    string   `json:"source_task_code"`
		SourceFieldKeys   []string `json:"source_field_keys"`
		BOMCategories     []string `json:"bom_categories"`
		BOMSubCategories  []string `json:"bom_sub_categories"`
	}
	var fields []formFieldDef
	if err := json.Unmarshal(formDef.Fields, &fields); err != nil {
		log.Printf("[WorkflowService] 解析表单字段失败: %v", err)
		return
	}

	for _, field := range fields {
		if field.Type != "procurement_control" || field.SourceTaskCode == "" || len(field.SourceFieldKeys) == 0 {
			continue
		}

		// 3. 通过 source_task_code 找到源任务
		var sourceTask entity.Task
		if err := s.db.WithContext(ctx).
			Where("project_id = ? AND code = ?", task.ProjectID, field.SourceTaskCode).
			First(&sourceTask).Error; err != nil {
			log.Printf("[WorkflowService] 找不到源任务 (project=%s code=%s): %v", task.ProjectID, field.SourceTaskCode, err)
			s.saveProcurementResult(ctx, task, formDef, field.Key, nil, fmt.Sprintf("找不到源任务: %s", field.SourceTaskCode))
			continue
		}

		// 4. 获取源任务的表单提交
		submission, err := s.taskFormRepo.FindLatestSubmission(ctx, sourceTask.ID)
		if err != nil || submission == nil {
			log.Printf("[WorkflowService] 源任务无提交数据 (task=%s): %v", sourceTask.ID, err)
			s.saveProcurementResult(ctx, task, formDef, field.Key, nil, "源任务无提交数据")
			continue
		}

		// 5. 获取源任务的表单定义（用于读取字段label）
		sourceFormDef, _ := s.taskFormRepo.FindByTaskID(ctx, sourceTask.ID)
		var sourceFields []formFieldDef
		if sourceFormDef != nil {
			json.Unmarshal(sourceFormDef.Fields, &sourceFields)
		}
		sourceFieldLabelMap := map[string]string{}
		sourceFieldTypeMap := map[string]string{}
		for _, sf := range sourceFields {
			sourceFieldLabelMap[sf.Key] = sf.Key // default to key
			sourceFieldTypeMap[sf.Key] = sf.Type
		}
		// also try to get label from full field definitions
		type fullFieldDef struct {
			Key   string `json:"key"`
			Label string `json:"label"`
			Type  string `json:"type"`
		}
		var fullFields []fullFieldDef
		if sourceFormDef != nil {
			json.Unmarshal(sourceFormDef.Fields, &fullFields)
		}
		for _, ff := range fullFields {
			if ff.Label != "" {
				sourceFieldLabelMap[ff.Key] = ff.Label
			}
			if ff.Type != "" {
				sourceFieldTypeMap[ff.Key] = ff.Type
			}
		}

		// 6. 从 submission.data 按 source_field_keys 提取物料列表
		submData := map[string]interface{}(submission.Data)
		var prItems []srmsvc.CreatePRItem
		var sources []map[string]interface{}

		for _, fk := range field.SourceFieldKeys {
			raw, ok := submData[fk]
			if !ok {
				continue
			}
			rawMap, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			itemsRaw, ok := rawMap["items"]
			if !ok {
				continue
			}
			itemsList, ok := itemsRaw.([]interface{})
			if !ok {
				continue
			}

			fieldType := sourceFieldTypeMap[fk]
			for _, itemRaw := range itemsList {
				item, ok := itemRaw.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := item["name"].(string)
				spec, _ := item["specification"].(string)
				unit, _ := item["unit"].(string)
				qty := 0.0
				if q, ok := item["quantity"].(float64); ok {
					qty = q
				}

				notes := ""
				switch fieldType {
				case "tooling_list":
					notes = "治具"
				case "consumable_list":
					notes = "辅料"
				}

				if name != "" && qty > 0 {
					prItems = append(prItems, srmsvc.CreatePRItem{
						MaterialName:  name,
						Specification: spec,
						Quantity:      qty,
						Unit:          unit,
						Notes:         notes,
					})
				}
			}

			sources = append(sources, map[string]interface{}{
				"task_code":   field.SourceTaskCode,
				"field_key":   fk,
				"field_label": sourceFieldLabelMap[fk],
				"item_count":  len(itemsList),
			})
		}

		// Fallback: if no items from inline form data, try reading from project BOM
		if len(prItems) == 0 && s.bomRepo != nil {
			log.Printf("[WorkflowService] 尝试从项目BOM获取物料 (task=%s project=%s)", task.ID, task.ProjectID)
			// Build category/sub-category filter sets
			catFilter := map[string]bool{}
			for _, c := range field.BOMCategories {
				catFilter[c] = true
			}
			subCatFilter := map[string]bool{}
			for _, sc := range field.BOMSubCategories {
				subCatFilter[sc] = true
			}
			for _, bomType := range []string{"EBOM", "PBOM"} {
				boms, err := s.bomRepo.ListByProject(ctx, task.ProjectID, bomType, "")
				if err != nil {
					continue
				}
				for _, bom := range boms {
					bomDetail, err := s.bomRepo.FindByID(ctx, bom.ID)
					if err != nil {
						continue
					}
					matchCount := 0
					for _, item := range bomDetail.Items {
						// Apply category filters if configured
						if len(catFilter) > 0 && !catFilter[item.Category] {
							continue
						}
						if len(subCatFilter) > 0 && !subCatFilter[item.SubCategory] {
							continue
						}
						if item.Name != "" && item.Quantity > 0 {
							spec := ""
							if item.ExtendedAttrs != nil {
								if s, ok := item.ExtendedAttrs["specification"].(string); ok {
									spec = s
								}
							}
							prItems = append(prItems, srmsvc.CreatePRItem{
								MaterialName:  item.Name,
								Specification: spec,
								Quantity:      item.Quantity,
								Unit:          item.Unit,
								Notes:         fmt.Sprintf("%s/%s", item.Category, item.SubCategory),
							})
							matchCount++
						}
					}
					if matchCount > 0 {
						sources = append(sources, map[string]interface{}{
							"task_code":   field.SourceTaskCode,
							"field_key":   bomType,
							"field_label": fmt.Sprintf("%s (%d项)", bomType, matchCount),
							"item_count":  matchCount,
						})
					}
				}
			}
		}

		if len(prItems) == 0 {
			log.Printf("[WorkflowService] 无可采购物料 (task=%s)", task.ID)
			s.saveProcurementResult(ctx, task, formDef, field.Key, sources, "无可采购物料")
			continue
		}

		// 7. 调用 SRM CreatePR
		projectID := task.ProjectID
		pr, err := s.srmProcurementSvc.CreatePR(ctx, operatorID, &srmsvc.CreatePRRequest{
			Title:     "采购需求 - " + task.Title,
			Type:      "npi_procurement",
			ProjectID: &projectID,
			Items:     prItems,
		})
		if err != nil {
			log.Printf("[WorkflowService] 创建PR失败 (task=%s): %v", task.ID, err)
			s.saveProcurementResult(ctx, task, formDef, field.Key, sources, fmt.Sprintf("创建PR失败: %v", err))
			continue
		}

		// 8. 保存结果到当前任务的 form submission
		resultData := map[string]interface{}{
			"sources":   sources,
			"pr_id":     pr.ID,
			"pr_code":   pr.PRCode,
			"pr_status": "created",
			"error":     nil,
		}
		s.saveProcurementSubmission(ctx, task, formDef, field.Key, resultData)
		log.Printf("[WorkflowService] 采购控件创建PR成功 task=%s pr=%s", task.ID, pr.PRCode)
	}
}

// saveProcurementResult 保存采购控件的错误或空结果
func (s *WorkflowService) saveProcurementResult(ctx context.Context, task *entity.Task, formDef *entity.TaskForm, fieldKey string, sources []map[string]interface{}, errMsg string) {
	resultData := map[string]interface{}{
		"sources":   sources,
		"pr_id":     nil,
		"pr_code":   nil,
		"pr_status": nil,
		"error":     errMsg,
	}
	s.saveProcurementSubmission(ctx, task, formDef, fieldKey, resultData)
}

// saveProcurementSubmission 保存采购控件数据到表单提交
func (s *WorkflowService) saveProcurementSubmission(ctx context.Context, task *entity.Task, formDef *entity.TaskForm, fieldKey string, data map[string]interface{}) {
	// 先查已有提交
	existing, _ := s.taskFormRepo.FindLatestSubmission(ctx, task.ID)
	if existing != nil {
		// 合并到已有提交的data中
		existingData := map[string]interface{}(existing.Data)
		existingData[fieldKey] = data
		existing.Data = entity.JSONB(existingData)
		s.db.WithContext(ctx).Save(existing)
	} else {
		// 创建新提交
		submission := &entity.TaskFormSubmission{
			ID:          uuid.New().String()[:32],
			FormID:      formDef.ID,
			TaskID:      task.ID,
			Data:        entity.JSONB{fieldKey: data},
			SubmittedBy: "system",
			SubmittedAt: time.Now(),
			Version:     0,
		}
		s.taskFormRepo.CreateSubmission(ctx, submission)
	}
}

// =============================================================================
// 飞书通知辅助方法
// =============================================================================

// notifyTaskAssigned 通知任务被指派
func (s *WorkflowService) notifyTaskAssigned(ctx context.Context, task *entity.Task, assigneeID, projectID string) {
	// 查找被指派人
	var assignee entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", assigneeID).First(&assignee).Error; err != nil {
		log.Printf("[WorkflowNotify] 查找被指派人失败 (user_id=%s): %v", assigneeID, err)
		return
	}
	if assignee.FeishuOpenID == "" {
		log.Printf("[WorkflowNotify] 被指派人[%s]没有飞书 open_id，跳过通知", assignee.Name)
		return
	}

	// 查找项目名
	projectName := "未知项目"
	var project entity.Project
	if err := s.db.WithContext(ctx).Where("id = ?", projectID).First(&project).Error; err == nil {
		projectName = project.Name
	}

	dueDate := "无"
	if task.DueDate != nil {
		dueDate = task.DueDate.Format("2006-01-02")
	}

	card := feishu.NewTaskAssignmentCard(task.Title, projectName, assignee.Name, dueDate)
	if err := s.feishuClient.SendUserCard(ctx, assignee.FeishuOpenID, card); err != nil {
		log.Printf("[WorkflowNotify] 发送任务指派通知给[%s]失败: %v", assignee.Name, err)
	} else {
		log.Printf("[WorkflowNotify] 已通知[%s]任务指派: %s", assignee.Name, task.Title)
	}
}

// notifyTaskStatusChange 通知任务状态变更
func (s *WorkflowService) notifyTaskStatusChange(ctx context.Context, task *entity.Task, fromStatus, toStatus, projectID string) {
	if task.AssigneeID == nil || *task.AssigneeID == "" {
		return
	}

	var assignee entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", *task.AssigneeID).First(&assignee).Error; err != nil {
		return
	}
	if assignee.FeishuOpenID == "" {
		return
	}

	projectName := "未知项目"
	var project entity.Project
	if err := s.db.WithContext(ctx).Where("id = ?", projectID).First(&project).Error; err == nil {
		projectName = project.Name
	}

	card := feishu.InteractiveCard{
		Config: &feishu.CardConfig{WideScreenMode: true},
		Header: &feishu.CardHeader{
			Title:    feishu.CardText{Tag: "plain_text", Content: "📝 任务状态变更"},
			Template: "blue",
		},
		Elements: []feishu.CardElement{
			{
				Tag: "div",
				Fields: []feishu.CardField{
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**任务**\n%s", task.Title)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**项目**\n%s", projectName)}},
					{IsShort: true, Text: feishu.CardText{Tag: "lark_md", Content: fmt.Sprintf("**状态变更**\n%s → %s", fromStatus, toStatus)}},
				},
			},
		},
	}

	if err := s.feishuClient.SendUserCard(ctx, assignee.FeishuOpenID, card); err != nil {
		log.Printf("[WorkflowNotify] 发送状态变更通知失败: %v", err)
	}
}

// logAction 记录任务操作日志
func (s *WorkflowService) logAction(ctx context.Context, projectID, taskID, action, fromStatus, toStatus, operatorID string, eventData map[string]interface{}, comment string) {
	actionLog := entity.TaskActionLog{
		ID:           uuid.New().String(),
		ProjectID:    projectID,
		TaskID:       taskID,
		Action:       action,
		FromStatus:   fromStatus,
		ToStatus:     toStatus,
		OperatorID:   operatorID,
		OperatorType: "user",
		Comment:      comment,
	}

	if operatorID == "system" {
		actionLog.OperatorType = "system"
	}

	if eventData != nil {
		actionLog.EventData = entity.JSONB(eventData)
	}

	if err := s.db.WithContext(ctx).Create(&actionLog).Error; err != nil {
		log.Printf("[WorkflowService] 记录操作日志失败: %v", err)
	}
}
