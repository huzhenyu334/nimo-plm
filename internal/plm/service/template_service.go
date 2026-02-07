package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateService 模板服务
type TemplateService struct {
	templateRepo *repository.TemplateRepository
	projectRepo  *repository.ProjectRepository
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateRepo *repository.TemplateRepository, projectRepo *repository.ProjectRepository) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
		projectRepo:  projectRepo,
	}
}

// ListTemplates 获取模板列表
func (s *TemplateService) ListTemplates(ctx context.Context, templateType, productType string, activeOnly bool) ([]entity.ProjectTemplate, error) {
	return s.templateRepo.List(ctx, templateType, productType, activeOnly)
}

// GetTemplate 获取模板详情
func (s *TemplateService) GetTemplate(ctx context.Context, id string) (*entity.ProjectTemplate, error) {
	tmpl, err := s.templateRepo.GetWithTasks(ctx, id)
	if err != nil {
		return nil, err
	}

	// 将模板级别的 Dependencies 分配到每个任务的 Dependencies 字段（gorm:"-"）
	depMap := make(map[string][]entity.TemplateTaskDependency)
	for _, d := range tmpl.Dependencies {
		depMap[d.TaskCode] = append(depMap[d.TaskCode], d)
	}
	for i := range tmpl.Tasks {
		tmpl.Tasks[i].Dependencies = depMap[tmpl.Tasks[i].TaskCode]
	}

	return tmpl, nil
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(ctx context.Context, template *entity.ProjectTemplate) error {
	template.ID = uuid.New().String()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	if template.Phases == nil {
		template.Phases = json.RawMessage(`["CONCEPT","EVT","DVT","PVT","MP"]`)
	}
	return s.templateRepo.Create(ctx, template)
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(ctx context.Context, template *entity.ProjectTemplate) error {
	template.UpdatedAt = time.Now()
	return s.templateRepo.Update(ctx, template)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(ctx context.Context, id string) error {
	return s.templateRepo.Delete(ctx, id)
}

// DuplicateTemplate 复制模板
func (s *TemplateService) DuplicateTemplate(ctx context.Context, id string, newCode, newName, createdBy string) (*entity.ProjectTemplate, error) {
	// 获取原模板
	original, err := s.templateRepo.GetWithTasks(ctx, id)
	if err != nil {
		return nil, err
	}

	// 创建新模板
	newTemplate := &entity.ProjectTemplate{
		ID:               uuid.New().String(),
		Code:             newCode,
		Name:             newName,
		Description:      original.Description,
		TemplateType:     "CUSTOM",
		ProductType:      original.ProductType,
		Phases:           original.Phases,
		EstimatedDays:    original.EstimatedDays,
		IsActive:         true,
		ParentTemplateID: &original.ID,
		Version:          "1.0",
		CreatedBy:        createdBy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.templateRepo.Create(ctx, newTemplate); err != nil {
		return nil, err
	}

	// 复制任务
	for _, task := range original.Tasks {
		newTask := &entity.TemplateTask{
			ID:                   uuid.New().String(),
			TemplateID:           newTemplate.ID,
			TaskCode:             task.TaskCode,
			Name:                 task.Name,
			Description:          task.Description,
			Phase:                task.Phase,
			ParentTaskCode:       task.ParentTaskCode,
			TaskType:             task.TaskType,
			DefaultAssigneeRole:  task.DefaultAssigneeRole,
			EstimatedDays:        task.EstimatedDays,
			IsCritical:           task.IsCritical,
			Deliverables:         task.Deliverables,
			Checklist:            task.Checklist,
			RequiresApproval:     task.RequiresApproval,
			ApprovalType:         task.ApprovalType,
			AutoCreateFeishuTask: task.AutoCreateFeishuTask,
			FeishuApprovalCode:   task.FeishuApprovalCode,
			SortOrder:            task.SortOrder,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		if err := s.templateRepo.CreateTask(ctx, newTask); err != nil {
			return nil, err
		}
	}

	// 复制依赖
	for _, dep := range original.Dependencies {
		newDep := &entity.TemplateTaskDependency{
			ID:                uuid.New().String(),
			TemplateID:        newTemplate.ID,
			TaskCode:          dep.TaskCode,
			DependsOnTaskCode: dep.DependsOnTaskCode,
			DependencyType:    dep.DependencyType,
			LagDays:           dep.LagDays,
		}
		if err := s.templateRepo.CreateDependency(ctx, newDep); err != nil {
			return nil, err
		}
	}

	return newTemplate, nil
}

// CreateTaskFromTemplate 模板任务操作
func (s *TemplateService) CreateTemplateTask(ctx context.Context, task *entity.TemplateTask) error {
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	return s.templateRepo.CreateTask(ctx, task)
}

// UpdateTemplateTask 更新模板任务
func (s *TemplateService) UpdateTemplateTask(ctx context.Context, task *entity.TemplateTask) error {
	task.UpdatedAt = time.Now()
	return s.templateRepo.UpdateTask(ctx, task)
}

// DeleteTemplateTask 删除模板任务
func (s *TemplateService) DeleteTemplateTask(ctx context.Context, templateID, taskCode string) error {
	return s.templateRepo.DeleteTask(ctx, templateID, taskCode)
}

// BatchSaveTasks 批量保存任务和依赖（清空旧数据并插入新数据）
func (s *TemplateService) BatchSaveTasks(ctx context.Context, templateID string, tasks []entity.TemplateTask, dependencies []entity.TemplateTaskDependency) error {
	db := s.templateRepo.DB()

	return db.Transaction(func(tx *gorm.DB) error {
		// 用原始SQL硬删除旧任务（避免GORM软删除干扰）
		if err := tx.Exec("DELETE FROM template_tasks WHERE template_id = ?", templateID).Error; err != nil {
			return fmt.Errorf("delete old tasks: %w", err)
		}

		// 删除旧依赖
		if err := tx.Exec("DELETE FROM template_task_dependencies WHERE template_id = ?", templateID).Error; err != nil {
			return fmt.Errorf("delete old dependencies: %w", err)
		}

		// 批量插入新任务
		if len(tasks) > 0 {
			now := time.Now()
			for i := range tasks {
				tasks[i].CreatedAt = now
				tasks[i].UpdatedAt = now
			}
			if err := tx.Create(&tasks).Error; err != nil {
				return fmt.Errorf("create tasks: %w", err)
			}
		}

		// 批量插入新依赖
		if len(dependencies) > 0 {
			if err := tx.Create(&dependencies).Error; err != nil {
				return fmt.Errorf("create dependencies: %w", err)
			}
		}

		return nil
	})
}

// CreateNewVersion 从已发布流程创建新草稿版本
func (s *TemplateService) CreateNewVersion(ctx context.Context, source *entity.ProjectTemplate, newVersion string, createdBy string) (*entity.ProjectTemplate, error) {
	db := s.templateRepo.DB()

	var newTemplate *entity.ProjectTemplate

	err := db.Transaction(func(tx *gorm.DB) error {
		// 创建新模板记录
		newID := uuid.New().String()
		baseCode := source.BaseCode
		if baseCode == "" {
			baseCode = source.Code
		}

		newTemplate = &entity.ProjectTemplate{
			ID:               newID,
			Code:             fmt.Sprintf("%s-v%s", baseCode, newVersion),
			Name:             source.Name,
			Description:      source.Description,
			TemplateType:     source.TemplateType,
			ProductType:      source.ProductType,
			Phases:           source.Phases,
			EstimatedDays:    source.EstimatedDays,
			IsActive:         true,
			ParentTemplateID: &source.ID,
			Version:          newVersion,
			Status:           "draft",
			BaseCode:         baseCode,
			CreatedBy:        createdBy,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := tx.Create(newTemplate).Error; err != nil {
			return fmt.Errorf("create new version: %w", err)
		}

		// 复制所有任务
		if len(source.Tasks) > 0 {
			now := time.Now()
			var newTasks []entity.TemplateTask
			for _, t := range source.Tasks {
				newTask := t
				newTask.ID = uuid.New().String()
				newTask.TemplateID = newID
				newTask.CreatedAt = now
				newTask.UpdatedAt = now
				newTask.Dependencies = nil // gorm:"-" 字段清除，避免干扰
				newTasks = append(newTasks, newTask)
			}
			if err := tx.Create(&newTasks).Error; err != nil {
				return fmt.Errorf("copy tasks: %w", err)
			}
		}

		// 复制依赖关系
		var sourceDeps []entity.TemplateTaskDependency
		if err := tx.Where("template_id = ?", source.ID).Find(&sourceDeps).Error; err != nil {
			return fmt.Errorf("load source deps: %w", err)
		}
		if len(sourceDeps) > 0 {
			var newDeps []entity.TemplateTaskDependency
			for _, d := range sourceDeps {
				newDeps = append(newDeps, entity.TemplateTaskDependency{
					ID:                uuid.New().String(),
					TemplateID:        newID,
					TaskCode:          d.TaskCode,
					DependsOnTaskCode: d.DependsOnTaskCode,
					DependencyType:    d.DependencyType,
					LagDays:           d.LagDays,
				})
			}
			if err := tx.Create(&newDeps).Error; err != nil {
				return fmt.Errorf("copy dependencies: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 重新加载完整数据
	return s.templateRepo.GetWithTasks(ctx, newTemplate.ID)
}

// ListVersions 获取同一流程的所有版本
func (s *TemplateService) ListVersions(ctx context.Context, baseCode string) ([]entity.ProjectTemplate, error) {
	db := s.templateRepo.DB()
	var templates []entity.ProjectTemplate
	err := db.Where("base_code = ? OR code = ?", baseCode, baseCode).
		Order("version DESC").
		Find(&templates).Error
	return templates, err
}

// CreateProjectFromTemplateInput 从模板创建项目的输入
type CreateProjectFromTemplateInput struct {
	TemplateID      string            `json:"template_id"`
	ProjectName     string            `json:"project_name"`
	ProjectCode     string            `json:"project_code"`
	ProductID       string            `json:"product_id"`
	StartDate       time.Time         `json:"start_date"`
	PMID            string            `json:"pm_user_id"`
	SkipWeekends    bool              `json:"skip_weekends"`
	RoleAssignments map[string]string `json:"role_assignments"` // role -> user_id
}

// CreateProjectFromTemplate 从模板创建项目
func (s *TemplateService) CreateProjectFromTemplate(ctx context.Context, input *CreateProjectFromTemplateInput, createdBy string) (*entity.Project, error) {
	// 获取模板
	template, err := s.templateRepo.GetWithTasks(ctx, input.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// 创建项目
	var productID *string
	if input.ProductID != "" {
		productID = &input.ProductID
	}
	
	project := &entity.Project{
		ID:          uuid.New().String()[:32],
		Code:        input.ProjectCode,
		Name:        input.ProjectName,
		ProductID:   productID,
		Phase:       "CONCEPT",
		Status:      "planning",
		StartDate:   &input.StartDate,
		ManagerID:   input.PMID,
		Progress:    0,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	// 创建项目阶段(ProjectPhase)记录
	phaseOrder := []string{"concept", "evt", "dvt", "pvt", "mp"}
	phaseNames := map[string]string{
		"concept": "Concept/立项",
		"evt":     "EVT 工程验证",
		"dvt":     "DVT 设计验证",
		"pvt":     "PVT 产品验证",
		"mp":      "MP 量产",
	}
	phaseIDMap := make(map[string]string) // phase name -> phase_id
	for seq, p := range phaseOrder {
		phaseID := uuid.New().String()[:32]
		phase := &entity.ProjectPhase{
			ID:        phaseID,
			ProjectID: project.ID,
			Phase:     p,
			Name:      phaseNames[p],
			Status:    "pending",
			Sequence:  seq + 1,
		}
		if err := s.projectRepo.DB().Create(phase).Error; err != nil {
			return nil, fmt.Errorf("create phase %s: %w", p, err)
		}
		phaseIDMap[p] = phaseID
	}

	// 构建任务依赖图
	depGraph := buildDependencyGraph(template.Dependencies)
	taskDates := calculateTaskDates(template.Tasks, depGraph, input.StartDate, input.SkipWeekends)

	// 构建模板任务的 task_code -> TemplateTask 映射
	ttMap := make(map[string]entity.TemplateTask)
	for _, tt := range template.Tasks {
		ttMap[tt.TaskCode] = tt
	}

	// 按层级排序创建：先 MILESTONE，再 TASK，最后 SUBTASK
	// 这样 parent_task_id 引用时父任务一定已经创建
	typeOrder := []string{"MILESTONE", "TASK", "SUBTASK"}
	taskMap := make(map[string]string) // task_code -> task_id

	seq := 0
	for _, taskType := range typeOrder {
		for _, tt := range template.Tasks {
			if tt.TaskType != taskType {
				continue
			}

			var assigneeID *string
			if tt.DefaultAssigneeRole != "" {
				if userID, ok := input.RoleAssignments[tt.DefaultAssigneeRole]; ok && userID != "" {
					assigneeID = &userID
				}
			}

			// 设置 parent_task_id（TASK和SUBTASK都可能有parent）
			var parentTaskID *string
			if tt.ParentTaskCode != "" {
				if pid, ok := taskMap[tt.ParentTaskCode]; ok {
					parentTaskID = &pid
				}
			}

			// 设置 phase_id（模板里phase可能是大写如CONCEPT，统一转小写匹配）
			var phaseID *string
			if tt.Phase != "" {
				phaseLower := strings.ToLower(tt.Phase)
				if pid, ok := phaseIDMap[phaseLower]; ok {
					phaseID = &pid
				}
			}

			seq++
			dates := taskDates[tt.TaskCode]
			task := &entity.Task{
				ID:                     uuid.New().String()[:32],
				ProjectID:              project.ID,
				ParentTaskID:           parentTaskID,
				PhaseID:                phaseID,
				Code:                   tt.TaskCode,
				Title:                  tt.Name,
				Description:            tt.Description,
				TaskType:               tt.TaskType,
				Status:                 "pending",
				Priority:               "medium",
				AssigneeID:             assigneeID,
				StartDate:              dates.Start,
				DueDate:                dates.End,
				Progress:               0,
				Sequence:               seq,
				AutoStart:              true,
				RequiresApproval:       tt.RequiresApproval,
				ApprovalType:           tt.ApprovalType,
				AutoCreateFeishuTask:   tt.AutoCreateFeishuTask,
				FeishuApprovalCode:     tt.FeishuApprovalCode,
				CreatedBy:              createdBy,
				CreatedAt:              time.Now(),
				UpdatedAt:              time.Now(),
			}

			if err := s.projectRepo.CreateTask(ctx, task); err != nil {
				return nil, fmt.Errorf("create task %s: %w", tt.TaskCode, err)
			}
			taskMap[tt.TaskCode] = task.ID
		}
	}

	// 创建任务依赖
	for _, dep := range template.Dependencies {
		taskID, ok1 := taskMap[dep.TaskCode]
		depTaskID, ok2 := taskMap[dep.DependsOnTaskCode]
		if !ok1 || !ok2 {
			continue
		}

		taskDep := &entity.TaskDependency{
			ID:             uuid.New().String()[:32],
			TaskID:         taskID,
			DependsOnID:    depTaskID,
			DependencyType: dep.DependencyType,
			LagDays:        dep.LagDays,
		}

		if err := s.projectRepo.CreateTaskDependency(ctx, taskDep); err != nil {
			continue
		}
	}

	return project, nil
}

// TaskDates 任务日期
type TaskDates struct {
	Start *time.Time
	End   *time.Time
}

// buildDependencyGraph 构建依赖图
func buildDependencyGraph(deps []entity.TemplateTaskDependency) map[string][]entity.TemplateTaskDependency {
	graph := make(map[string][]entity.TemplateTaskDependency)
	for _, dep := range deps {
		graph[dep.TaskCode] = append(graph[dep.TaskCode], dep)
	}
	return graph
}

// calculateTaskDates 计算任务日期
func calculateTaskDates(tasks []entity.TemplateTask, depGraph map[string][]entity.TemplateTaskDependency, startDate time.Time, skipWeekends bool) map[string]TaskDates {
	dates := make(map[string]TaskDates)
	taskMap := make(map[string]entity.TemplateTask)
	for _, t := range tasks {
		taskMap[t.TaskCode] = t
	}

	// 递归计算每个任务的开始日期
	var calculateStart func(taskCode string) time.Time
	calculateStart = func(taskCode string) time.Time {
		if d, ok := dates[taskCode]; ok && d.Start != nil {
			return *d.Start
		}

		deps := depGraph[taskCode]
		if len(deps) == 0 {
			// 没有依赖，从项目开始日期算
			return startDate
		}

		// 取所有依赖完成后的最大日期
		maxDate := startDate
		for _, dep := range deps {
			depTask, ok := taskMap[dep.DependsOnTaskCode]
			if !ok {
				continue
			}

			depStart := calculateStart(dep.DependsOnTaskCode)
			depEnd := addWorkDays(depStart, depTask.EstimatedDays, skipWeekends)

			switch dep.DependencyType {
			case "FS": // 完成-开始
				candidateStart := addWorkDays(depEnd, dep.LagDays, skipWeekends)
				if candidateStart.After(maxDate) {
					maxDate = candidateStart
				}
			case "SS": // 开始-开始
				candidateStart := addWorkDays(depStart, dep.LagDays, skipWeekends)
				if candidateStart.After(maxDate) {
					maxDate = candidateStart
				}
			default:
				// 默认 FS
				candidateStart := addWorkDays(depEnd, dep.LagDays, skipWeekends)
				if candidateStart.After(maxDate) {
					maxDate = candidateStart
				}
			}
		}

		return maxDate
	}

	// 计算所有任务日期
	for _, task := range tasks {
		start := calculateStart(task.TaskCode)
		end := addWorkDays(start, task.EstimatedDays, skipWeekends)
		dates[task.TaskCode] = TaskDates{Start: &start, End: &end}
	}

	return dates
}

// addWorkDays 添加工作日
func addWorkDays(start time.Time, days int, skipWeekends bool) time.Time {
	if days <= 0 {
		return start
	}

	result := start
	for i := 0; i < days; i++ {
		result = result.AddDate(0, 0, 1)
		if skipWeekends {
			for result.Weekday() == time.Saturday || result.Weekday() == time.Sunday {
				result = result.AddDate(0, 0, 1)
			}
		}
	}
	return result
}
