package repository

import (
	"context"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActivityLogRepository 操作日志仓库
type ActivityLogRepository struct {
	db *gorm.DB
}

func NewActivityLogRepository(db *gorm.DB) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

// Create 创建操作日志
func (r *ActivityLogRepository) Create(ctx context.Context, log *entity.ActivityLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()[:32]
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByEntity 查询某实体的操作日志
func (r *ActivityLogRepository) FindByEntity(ctx context.Context, entityType, entityID string, page, pageSize int) ([]entity.ActivityLog, int64, error) {
	var items []entity.ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.ActivityLog{}).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// LogActivity 便捷记录操作日志
func (r *ActivityLogRepository) LogActivity(ctx context.Context, entityType, entityID, entityCode, action, fromStatus, toStatus, content, operatorID, operatorName string) {
	log := &entity.ActivityLog{
		ID:           uuid.New().String()[:32],
		EntityType:   entityType,
		EntityID:     entityID,
		EntityCode:   entityCode,
		Action:       action,
		FromStatus:   fromStatus,
		ToStatus:     toStatus,
		Content:      content,
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	// 异步写日志，忽略错误
	r.db.WithContext(ctx).Create(log)
}
