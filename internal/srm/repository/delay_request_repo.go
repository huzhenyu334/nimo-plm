package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// DelayRequestRepository 延期审批仓库
type DelayRequestRepository struct {
	db *gorm.DB
}

func NewDelayRequestRepository(db *gorm.DB) *DelayRequestRepository {
	return &DelayRequestRepository{db: db}
}

// FindAll 查询延期申请列表
func (r *DelayRequestRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.DelayRequest, int64, error) {
	var items []entity.DelayRequest
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.DelayRequest{})

	if projectID := filters["srm_project_id"]; projectID != "" {
		query = query.Where("srm_project_id = ?", projectID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}

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

// FindByID 根据ID查找延期申请
func (r *DelayRequestRepository) FindByID(ctx context.Context, id string) (*entity.DelayRequest, error) {
	var req entity.DelayRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

// FindByProject 根据采购项目ID查找延期申请
func (r *DelayRequestRepository) FindByProject(ctx context.Context, srmProjectID string) ([]entity.DelayRequest, error) {
	var items []entity.DelayRequest
	err := r.db.WithContext(ctx).
		Where("srm_project_id = ?", srmProjectID).
		Order("created_at DESC").
		Find(&items).Error
	return items, err
}

// Create 创建延期申请
func (r *DelayRequestRepository) Create(ctx context.Context, req *entity.DelayRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

// Update 更新延期申请
func (r *DelayRequestRepository) Update(ctx context.Context, req *entity.DelayRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

// GenerateCode 生成延期审批编码 DLY-{year}-{4位}
func (r *DelayRequestRepository) GenerateCode(ctx context.Context) (string, error) {
	year := time.Now().Format("2006")
	prefix := fmt.Sprintf("DLY-%s-", year)

	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&entity.DelayRequest{}).
		Select("COALESCE(MAX(code), '')").
		Where("code LIKE ?", prefix+"%").
		Scan(&maxCode).Error
	if err != nil {
		return "", err
	}

	var seq int
	if maxCode != "" {
		fmt.Sscanf(maxCode, "DLY-"+year+"-%04d", &seq)
	}
	seq++
	return fmt.Sprintf("DLY-%s-%04d", year, seq), nil
}
