package repository

import (
	"context"
	"errors"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// SamplingRepository 打样请求仓库
type SamplingRepository struct {
	db *gorm.DB
}

func NewSamplingRepository(db *gorm.DB) *SamplingRepository {
	return &SamplingRepository{db: db}
}

// Create 创建打样请求
func (r *SamplingRepository) Create(ctx context.Context, req *entity.SamplingRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

// FindByID 根据ID查询打样请求
func (r *SamplingRepository) FindByID(ctx context.Context, id string) (*entity.SamplingRequest, error) {
	var req entity.SamplingRequest
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

// FindByPRItemID 查询物料的打样记录列表
func (r *SamplingRepository) FindByPRItemID(ctx context.Context, prItemID string) ([]entity.SamplingRequest, error) {
	var items []entity.SamplingRequest
	err := r.db.WithContext(ctx).
		Where("pr_item_id = ?", prItemID).
		Order("round DESC, created_at DESC").
		Find(&items).Error
	return items, err
}

// Update 更新打样请求
func (r *SamplingRepository) Update(ctx context.Context, req *entity.SamplingRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

// FindByApprovalID 根据飞书审批实例ID查询打样请求
func (r *SamplingRepository) FindByApprovalID(ctx context.Context, approvalID string) (*entity.SamplingRequest, error) {
	var req entity.SamplingRequest
	err := r.db.WithContext(ctx).Where("approval_id = ?", approvalID).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

// GetMaxRound 获取物料的最大打样轮次
func (r *SamplingRepository) GetMaxRound(ctx context.Context, prItemID string) (int, error) {
	var maxRound int
	err := r.db.WithContext(ctx).
		Model(&entity.SamplingRequest{}).
		Select("COALESCE(MAX(round), 0)").
		Where("pr_item_id = ?", prItemID).
		Scan(&maxRound).Error
	return maxRound, err
}
