package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"gorm.io/gorm"
)

// DocumentRepository 文档仓储
type DocumentRepository struct {
	db *gorm.DB
}

// NewDocumentRepository 创建文档仓储
func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// FindByID 根据ID查找文档
func (r *DocumentRepository) FindByID(ctx context.Context, id string) (*entity.Document, error) {
	var doc entity.Document
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Uploader").
		Preload("Releaser").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

// FindByCode 根据编码查找文档
func (r *DocumentRepository) FindByCode(ctx context.Context, code string) (*entity.Document, error) {
	var doc entity.Document
	err := r.db.WithContext(ctx).
		Where("code = ? AND deleted_at IS NULL", code).
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

// Create 创建文档
func (r *DocumentRepository) Create(ctx context.Context, doc *entity.Document) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

// Update 更新文档
func (r *DocumentRepository) Update(ctx context.Context, doc *entity.Document) error {
	return r.db.WithContext(ctx).Save(doc).Error
}

// Delete 软删除文档
func (r *DocumentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Document{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// List 获取文档列表
func (r *DocumentRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.Document, int64, error) {
	var docs []entity.Document
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Document{}).Where("deleted_at IS NULL")

	// 应用过滤条件
	if keyword, ok := filters["keyword"].(string); ok && keyword != "" {
		query = query.Where("title ILIKE ? OR code ILIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if categoryID, ok := filters["category_id"].(string); ok && categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if relatedType, ok := filters["related_type"].(string); ok && relatedType != "" {
		query = query.Where("related_type = ?", relatedType)
	}
	if relatedID, ok := filters["related_id"].(string); ok && relatedID != "" {
		query = query.Where("related_id = ?", relatedID)
	}
	if uploadedBy, ok := filters["uploaded_by"].(string); ok && uploadedBy != "" {
		query = query.Where("uploaded_by = ?", uploadedBy)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("Category").
		Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&docs).Error
	if err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

// ListByRelated 获取关联对象的文档列表
func (r *DocumentRepository) ListByRelated(ctx context.Context, relatedType, relatedID string) ([]entity.Document, error) {
	var docs []entity.Document
	err := r.db.WithContext(ctx).
		Where("related_type = ? AND related_id = ? AND deleted_at IS NULL", relatedType, relatedID).
		Preload("Category").
		Preload("Uploader").
		Order("created_at DESC").
		Find(&docs).Error
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// GenerateCode 生成文档编码
func (r *DocumentRepository) GenerateCode(ctx context.Context) (string, error) {
	var seq int
	err := r.db.WithContext(ctx).Raw("SELECT nextval('document_code_seq')").Scan(&seq).Error
	if err != nil {
		return "", err
	}
	year := time.Now().Year()
	return fmt.Sprintf("DOC-%d-%04d", year, seq), nil
}

// Release 发布文档
func (r *DocumentRepository) Release(ctx context.Context, id string, releasedBy string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.Document{}).
		Where("id = ? AND status = ?", id, entity.DocumentStatusDraft).
		Updates(map[string]interface{}{
			"status":      entity.DocumentStatusReleased,
			"released_by": releasedBy,
			"released_at": now,
			"updated_at":  now,
		}).Error
}

// Obsolete 废弃文档
func (r *DocumentRepository) Obsolete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&entity.Document{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     entity.DocumentStatusObsolete,
			"updated_at": time.Now(),
		}).Error
}

// ============================================================
// 文档版本相关操作
// ============================================================

// CreateVersion 创建文档版本
func (r *DocumentRepository) CreateVersion(ctx context.Context, version *entity.DocumentVersion) error {
	return r.db.WithContext(ctx).Create(version).Error
}

// ListVersions 获取文档版本列表
func (r *DocumentRepository) ListVersions(ctx context.Context, documentID string) ([]entity.DocumentVersion, error) {
	var versions []entity.DocumentVersion
	err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Preload("Creator").
		Order("created_at DESC").
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// FindVersionByID 根据ID查找文档版本
func (r *DocumentRepository) FindVersionByID(ctx context.Context, id string) (*entity.DocumentVersion, error) {
	var version entity.DocumentVersion
	err := r.db.WithContext(ctx).
		Preload("Document").
		Preload("Creator").
		Where("id = ?", id).
		First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &version, nil
}

// GetLatestVersion 获取最新版本
func (r *DocumentRepository) GetLatestVersion(ctx context.Context, documentID string) (*entity.DocumentVersion, error) {
	var version entity.DocumentVersion
	err := r.db.WithContext(ctx).
		Where("document_id = ?", documentID).
		Order("created_at DESC").
		First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &version, nil
}

// ============================================================
// 文档分类相关操作
// ============================================================

// DocumentCategoryRepository 文档分类仓储
type DocumentCategoryRepository struct {
	db *gorm.DB
}

// NewDocumentCategoryRepository 创建文档分类仓储
func NewDocumentCategoryRepository(db *gorm.DB) *DocumentCategoryRepository {
	return &DocumentCategoryRepository{db: db}
}

// FindByID 根据ID查找分类
func (r *DocumentCategoryRepository) FindByID(ctx context.Context, id string) (*entity.DocumentCategory, error) {
	var cat entity.DocumentCategory
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

// FindByCode 根据编码查找分类
func (r *DocumentCategoryRepository) FindByCode(ctx context.Context, code string) (*entity.DocumentCategory, error) {
	var cat entity.DocumentCategory
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

// List 获取所有分类
func (r *DocumentCategoryRepository) List(ctx context.Context) ([]entity.DocumentCategory, error) {
	var cats []entity.DocumentCategory
	err := r.db.WithContext(ctx).
		Order("sort_order ASC").
		Find(&cats).Error
	if err != nil {
		return nil, err
	}
	return cats, nil
}

// ListTree 获取分类树
func (r *DocumentCategoryRepository) ListTree(ctx context.Context) ([]entity.DocumentCategory, error) {
	var cats []entity.DocumentCategory
	err := r.db.WithContext(ctx).
		Where("parent_id IS NULL OR parent_id = ''").
		Preload("Children").
		Order("sort_order ASC").
		Find(&cats).Error
	if err != nil {
		return nil, err
	}
	return cats, nil
}

// ============================================================
// 任务附件相关操作
// ============================================================

// TaskAttachmentRepository 任务附件仓储
type TaskAttachmentRepository struct {
	db *gorm.DB
}

// NewTaskAttachmentRepository 创建任务附件仓储
func NewTaskAttachmentRepository(db *gorm.DB) *TaskAttachmentRepository {
	return &TaskAttachmentRepository{db: db}
}

// Create 创建附件
func (r *TaskAttachmentRepository) Create(ctx context.Context, attachment *entity.TaskAttachment) error {
	return r.db.WithContext(ctx).Create(attachment).Error
}

// Delete 删除附件
func (r *TaskAttachmentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.TaskAttachment{}, "id = ?", id).Error
}

// ListByTask 获取任务的附件列表
func (r *TaskAttachmentRepository) ListByTask(ctx context.Context, taskID string) ([]entity.TaskAttachment, error) {
	var attachments []entity.TaskAttachment
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Preload("Uploader").
		Order("created_at DESC").
		Find(&attachments).Error
	if err != nil {
		return nil, err
	}
	return attachments, nil
}

// FindByID 根据ID查找附件
func (r *TaskAttachmentRepository) FindByID(ctx context.Context, id string) (*entity.TaskAttachment, error) {
	var attachment entity.TaskAttachment
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &attachment, nil
}

// ============================================================
// 操作日志相关操作
// ============================================================

// OperationLogRepository 操作日志仓储
type OperationLogRepository struct {
	db *gorm.DB
}

// NewOperationLogRepository 创建操作日志仓储
func NewOperationLogRepository(db *gorm.DB) *OperationLogRepository {
	return &OperationLogRepository{db: db}
}

// Create 创建日志
func (r *OperationLogRepository) Create(ctx context.Context, log *entity.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// List 获取日志列表
func (r *OperationLogRepository) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]entity.OperationLog, int64, error) {
	var logs []entity.OperationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.OperationLog{})

	// 应用过滤条件
	if userID, ok := filters["user_id"].(string); ok && userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if module, ok := filters["module"].(string); ok && module != "" {
		query = query.Where("module = ?", module)
	}
	if action, ok := filters["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}
	if targetType, ok := filters["target_type"].(string); ok && targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if targetID, ok := filters["target_id"].(string); ok && targetID != "" {
		query = query.Where("target_id = ?", targetID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ============================================================
// 系统配置相关操作
// ============================================================

// SystemConfigRepository 系统配置仓储
type SystemConfigRepository struct {
	db *gorm.DB
}

// NewSystemConfigRepository 创建系统配置仓储
func NewSystemConfigRepository(db *gorm.DB) *SystemConfigRepository {
	return &SystemConfigRepository{db: db}
}

// FindByKey 根据Key查找配置
func (r *SystemConfigRepository) FindByKey(ctx context.Context, key string) (*entity.SystemConfig, error) {
	var config entity.SystemConfig
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &config, nil
}

// Upsert 更新或创建配置
func (r *SystemConfigRepository) Upsert(ctx context.Context, config *entity.SystemConfig) error {
	return r.db.WithContext(ctx).
		Where("key = ?", config.Key).
		Assign(map[string]interface{}{
			"value":       config.Value,
			"value_type":  config.ValueType,
			"module":      config.Module,
			"description": config.Description,
			"is_public":   config.IsPublic,
			"updated_by":  config.UpdatedBy,
			"updated_at":  time.Now(),
		}).
		FirstOrCreate(config).Error
}

// List 获取配置列表
func (r *SystemConfigRepository) List(ctx context.Context, module string, publicOnly bool) ([]entity.SystemConfig, error) {
	var configs []entity.SystemConfig
	query := r.db.WithContext(ctx).Model(&entity.SystemConfig{})

	if module != "" {
		query = query.Where("module = ?", module)
	}
	if publicOnly {
		query = query.Where("is_public = ?", true)
	}

	err := query.Order("key ASC").Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}
