package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"gorm.io/gorm"
)

// ECNRepository ECN仓储
type ECNRepository struct {
	db *gorm.DB
}

// NewECNRepository 创建ECN仓储
func NewECNRepository(db *gorm.DB) *ECNRepository {
	return &ECNRepository{db: db}
}

// FindByID 根据ID查找ECN
func (r *ECNRepository) FindByID(ctx context.Context, id string) (*entity.ECN, error) {
	var ecn entity.ECN
	err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Requester").
		Preload("Approver").
		Preload("Implementer").
		Preload("AffectedItems").
		Preload("Approvals").
		Preload("Approvals.Approver").
		Preload("Tasks").
		Preload("Tasks.Assignee").
		Where("id = ?", id).
		First(&ecn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ecn, nil
}

// FindByCode 根据编码查找ECN
func (r *ECNRepository) FindByCode(ctx context.Context, code string) (*entity.ECN, error) {
	var ecn entity.ECN
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&ecn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ecn, nil
}

// Create 创建ECN
func (r *ECNRepository) Create(ctx context.Context, ecn *entity.ECN) error {
	return r.db.WithContext(ctx).Create(ecn).Error
}

// Update 更新ECN
func (r *ECNRepository) Update(ctx context.Context, ecn *entity.ECN) error {
	return r.db.WithContext(ctx).Save(ecn).Error
}

// UpdateStatus 更新ECN状态
func (r *ECNRepository) UpdateStatus(ctx context.Context, id string, status string, updatedBy string) error {
	return r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// List 获取ECN列表
func (r *ECNRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.ECN, int64, error) {
	var ecns []entity.ECN
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.ECN{})

	// 应用过滤条件
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("title ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if productID, ok := filters["product_id"].(string); ok && productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if changeType, ok := filters["change_type"].(string); ok && changeType != "" {
		query = query.Where("change_type = ?", changeType)
	}
	if requestedBy, ok := filters["requested_by"].(string); ok && requestedBy != "" {
		query = query.Where("requested_by = ?", requestedBy)
	}
	if urgency, ok := filters["urgency"].(string); ok && urgency != "" {
		query = query.Where("urgency = ?", urgency)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Product").
		Preload("Requester").
		Preload("AffectedItems").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&ecns).Error
	if err != nil {
		return nil, 0, err
	}

	return ecns, total, nil
}

// ListByProduct 获取产品的ECN列表
func (r *ECNRepository) ListByProduct(ctx context.Context, productID string) ([]entity.ECN, error) {
	var ecns []entity.ECN
	err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Preload("Requester").
		Order("created_at DESC").
		Find(&ecns).Error
	if err != nil {
		return nil, err
	}
	return ecns, nil
}

// GenerateCode 生成ECN编码
func (r *ECNRepository) GenerateCode(ctx context.Context) (string, error) {
	var seq int
	err := r.db.WithContext(ctx).Raw("SELECT nextval('ecn_code_seq')").Scan(&seq).Error
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("ECN-%d-%04d", year, seq), nil
}

// ============================================================
// ECN统计
// ============================================================

// ECNStats ECN统计数据
type ECNStats struct {
	PendingApproval int64 `json:"pending_approval"`
	Executing       int64 `json:"executing"`
	MonthCreated    int64 `json:"month_created"`
	MonthClosed     int64 `json:"month_closed"`
}

// GetStats 获取ECN统计数据
func (r *ECNRepository) GetStats(ctx context.Context, userID string) (*ECNStats, error) {
	stats := &ECNStats{}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 待我审批数
	r.db.WithContext(ctx).
		Model(&entity.ECNApproval{}).
		Joins("JOIN ecns ON ecns.id = ecn_approvals.ecn_id").
		Where("ecn_approvals.approver_id = ? AND ecn_approvals.status = ? AND ecns.status = ?",
			userID, "pending", entity.ECNStatusPending).
		Count(&stats.PendingApproval)

	// 执行中数
	r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("status IN ?", []string{entity.ECNStatusExecuting, entity.ECNStatusApproved}).
		Count(&stats.Executing)

	// 本月新建数
	r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("created_at >= ?", monthStart).
		Count(&stats.MonthCreated)

	// 本月关闭数
	r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("status = ? AND updated_at >= ?", entity.ECNStatusClosed, monthStart).
		Count(&stats.MonthClosed)

	return stats, nil
}

// ListMyPending 获取待我审批的ECN
func (r *ECNRepository) ListMyPending(ctx context.Context, userID string) ([]entity.ECN, error) {
	var ecnIDs []string
	err := r.db.WithContext(ctx).
		Model(&entity.ECNApproval{}).
		Select("ecn_id").
		Joins("JOIN ecns ON ecns.id = ecn_approvals.ecn_id").
		Where("ecn_approvals.approver_id = ? AND ecn_approvals.status = ? AND ecns.status = ?",
			userID, "pending", entity.ECNStatusPending).
		Find(&ecnIDs).Error
	if err != nil {
		return nil, err
	}
	if len(ecnIDs) == 0 {
		return []entity.ECN{}, nil
	}

	var ecns []entity.ECN
	err = r.db.WithContext(ctx).
		Preload("Product").
		Preload("Requester").
		Preload("AffectedItems").
		Where("id IN ?", ecnIDs).
		Order("created_at DESC").
		Find(&ecns).Error
	return ecns, err
}

// ============================================================
// ECN受影响项目相关操作
// ============================================================

// AddAffectedItem 添加受影响项目
func (r *ECNRepository) AddAffectedItem(ctx context.Context, item *entity.ECNAffectedItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// UpdateAffectedItem 更新受影响项目
func (r *ECNRepository) UpdateAffectedItem(ctx context.Context, item *entity.ECNAffectedItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// RemoveAffectedItem 移除受影响项目
func (r *ECNRepository) RemoveAffectedItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.ECNAffectedItem{}, "id = ?", id).Error
}

// ListAffectedItems 获取ECN受影响项目
func (r *ECNRepository) ListAffectedItems(ctx context.Context, ecnID string) ([]entity.ECNAffectedItem, error) {
	var items []entity.ECNAffectedItem
	err := r.db.WithContext(ctx).
		Where("ecn_id = ?", ecnID).
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// FindAffectedItemByID 根据ID查找受影响项
func (r *ECNRepository) FindAffectedItemByID(ctx context.Context, id string) (*entity.ECNAffectedItem, error) {
	var item entity.ECNAffectedItem
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ============================================================
// ECN审批相关操作
// ============================================================

// AddApproval 添加审批人
func (r *ECNRepository) AddApproval(ctx context.Context, approval *entity.ECNApproval) error {
	return r.db.WithContext(ctx).Create(approval).Error
}

// UpdateApproval 更新审批
func (r *ECNRepository) UpdateApproval(ctx context.Context, approval *entity.ECNApproval) error {
	return r.db.WithContext(ctx).Save(approval).Error
}

// FindApprovalByID 根据ID查找审批记录
func (r *ECNRepository) FindApprovalByID(ctx context.Context, id string) (*entity.ECNApproval, error) {
	var approval entity.ECNApproval
	err := r.db.WithContext(ctx).
		Preload("Approver").
		Where("id = ?", id).
		First(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &approval, nil
}

// ListApprovals 获取ECN审批列表
func (r *ECNRepository) ListApprovals(ctx context.Context, ecnID string) ([]entity.ECNApproval, error) {
	var approvals []entity.ECNApproval
	err := r.db.WithContext(ctx).
		Where("ecn_id = ?", ecnID).
		Preload("Approver").
		Order("sequence ASC").
		Find(&approvals).Error
	if err != nil {
		return nil, err
	}
	return approvals, nil
}

// GetPendingApproval 获取当前待审批记录
func (r *ECNRepository) GetPendingApproval(ctx context.Context, ecnID string) (*entity.ECNApproval, error) {
	var approval entity.ECNApproval
	err := r.db.WithContext(ctx).
		Where("ecn_id = ? AND status = ?", ecnID, "pending").
		Order("sequence ASC").
		First(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没有待审批记录不是错误
		}
		return nil, err
	}
	return &approval, nil
}

// CheckAllApproved 检查是否所有审批都已通过
func (r *ECNRepository) CheckAllApproved(ctx context.Context, ecnID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.ECNApproval{}).
		Where("ecn_id = ? AND status != ?", ecnID, "approved").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// SubmitForApproval 提交审批
func (r *ECNRepository) SubmitForApproval(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 更新ECN状态
		if err := tx.Model(&entity.ECN{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":       entity.ECNStatusPending,
				"requested_at": now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

// Approve 审批通过
func (r *ECNRepository) Approve(ctx context.Context, ecnID string, approverID string, comment string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 更新审批记录
		if err := tx.Model(&entity.ECNApproval{}).
			Where("ecn_id = ? AND approver_id = ? AND status = ?", ecnID, approverID, "pending").
			Updates(map[string]interface{}{
				"status":     "approved",
				"decision":   "approve",
				"comment":    comment,
				"decided_at": now,
			}).Error; err != nil {
			return err
		}

		// 检查是否所有审批都已通过
		var pendingCount int64
		if err := tx.Model(&entity.ECNApproval{}).
			Where("ecn_id = ? AND status = ?", ecnID, "pending").
			Count(&pendingCount).Error; err != nil {
			return err
		}

		// 如果所有审批都通过，更新ECN状态为执行中
		if pendingCount == 0 {
			if err := tx.Model(&entity.ECN{}).
				Where("id = ?", ecnID).
				Updates(map[string]interface{}{
					"status":      entity.ECNStatusExecuting,
					"approved_by": approverID,
					"approved_at": now,
					"updated_at":  now,
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Reject 审批拒绝
func (r *ECNRepository) Reject(ctx context.Context, ecnID string, approverID string, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 更新审批记录
		if err := tx.Model(&entity.ECNApproval{}).
			Where("ecn_id = ? AND approver_id = ? AND status = ?", ecnID, approverID, "pending").
			Updates(map[string]interface{}{
				"status":     "rejected",
				"decision":   "reject",
				"comment":    reason,
				"decided_at": now,
			}).Error; err != nil {
			return err
		}

		// 更新ECN状态为拒绝
		if err := tx.Model(&entity.ECN{}).
			Where("id = ?", ecnID).
			Updates(map[string]interface{}{
				"status":           entity.ECNStatusRejected,
				"rejection_reason": reason,
				"updated_at":       now,
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// Implement 实施ECN（启动执行）
func (r *ECNRepository) Implement(ctx context.Context, ecnID string, implementedBy string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("id = ? AND status IN ?", ecnID, []string{entity.ECNStatusApproved, entity.ECNStatusExecuting}).
		Updates(map[string]interface{}{
			"status":         entity.ECNStatusExecuting,
			"implemented_by": implementedBy,
			"implemented_at": now,
			"updated_at":     now,
		}).Error
}

// ============================================================
// ECN执行任务相关操作
// ============================================================

// CreateTask 创建执行任务
func (r *ECNRepository) CreateTask(ctx context.Context, task *entity.ECNTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// UpdateTask 更新执行任务
func (r *ECNRepository) UpdateTask(ctx context.Context, task *entity.ECNTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// FindTaskByID 根据ID查找执行任务
func (r *ECNRepository) FindTaskByID(ctx context.Context, id string) (*entity.ECNTask, error) {
	var task entity.ECNTask
	err := r.db.WithContext(ctx).
		Preload("Assignee").
		Where("id = ?", id).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

// ListTasks 获取ECN执行任务列表
func (r *ECNRepository) ListTasks(ctx context.Context, ecnID string) ([]entity.ECNTask, error) {
	var tasks []entity.ECNTask
	err := r.db.WithContext(ctx).
		Where("ecn_id = ?", ecnID).
		Preload("Assignee").
		Order("sort_order ASC, created_at ASC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTaskCompletion 获取ECN任务完成率
func (r *ECNRepository) GetTaskCompletion(ctx context.Context, ecnID string) (int, error) {
	var total, completed int64
	r.db.WithContext(ctx).
		Model(&entity.ECNTask{}).
		Where("ecn_id = ? AND status != ?", ecnID, entity.ECNTaskStatusSkipped).
		Count(&total)
	r.db.WithContext(ctx).
		Model(&entity.ECNTask{}).
		Where("ecn_id = ? AND status = ?", ecnID, entity.ECNTaskStatusCompleted).
		Count(&completed)

	if total == 0 {
		return 0, nil
	}
	return int(completed * 100 / total), nil
}

// ============================================================
// ECN历史记录相关操作
// ============================================================

// AddHistory 添加操作历史
func (r *ECNRepository) AddHistory(ctx context.Context, history *entity.ECNHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// ListHistory 获取ECN操作历史
func (r *ECNRepository) ListHistory(ctx context.Context, ecnID string) ([]entity.ECNHistory, error) {
	var histories []entity.ECNHistory
	err := r.db.WithContext(ctx).
		Where("ecn_id = ?", ecnID).
		Preload("User").
		Order("created_at DESC").
		Find(&histories).Error
	if err != nil {
		return nil, err
	}
	return histories, nil
}

// ============================================================
// ECN关闭
// ============================================================

// CloseECN 关闭ECN
func (r *ECNRepository) CloseECN(ctx context.Context, ecnID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("id = ?", ecnID).
		Updates(map[string]interface{}{
			"status":          entity.ECNStatusClosed,
			"completion_rate": 100,
			"updated_at":      now,
		}).Error
}

// UpdateCompletionRate 更新ECN完成率
func (r *ECNRepository) UpdateCompletionRate(ctx context.Context, ecnID string, rate int) error {
	return r.db.WithContext(ctx).
		Model(&entity.ECN{}).
		Where("id = ?", ecnID).
		Updates(map[string]interface{}{
			"completion_rate": rate,
			"updated_at":     time.Now(),
		}).Error
}
