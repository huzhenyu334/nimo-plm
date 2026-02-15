package repository

import (
	"context"
	"errors"

	"github.com/bitfantasy/nimo/internal/srm/entity"
	"gorm.io/gorm"
)

// EvaluationRepository 评估仓库
type EvaluationRepository struct {
	db *gorm.DB
}

func NewEvaluationRepository(db *gorm.DB) *EvaluationRepository {
	return &EvaluationRepository{db: db}
}

// FindAll 查询评估列表
func (r *EvaluationRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string) ([]entity.SupplierEvaluation, int64, error) {
	var items []entity.SupplierEvaluation
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.SupplierEvaluation{})

	if supplierID := filters["supplier_id"]; supplierID != "" {
		query = query.Where("supplier_id = ?", supplierID)
	}
	if status := filters["status"]; status != "" {
		query = query.Where("status = ?", status)
	}
	if evalType := filters["eval_type"]; evalType != "" {
		query = query.Where("eval_type = ?", evalType)
	}
	if period := filters["period"]; period != "" {
		query = query.Where("period = ?", period)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Supplier").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// FindByID 根据ID查找评估
func (r *EvaluationRepository) FindByID(ctx context.Context, id string) (*entity.SupplierEvaluation, error) {
	var eval entity.SupplierEvaluation
	err := r.db.WithContext(ctx).
		Preload("Supplier").
		Where("id = ?", id).
		First(&eval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &eval, nil
}

// Create 创建评估
func (r *EvaluationRepository) Create(ctx context.Context, eval *entity.SupplierEvaluation) error {
	return r.db.WithContext(ctx).Create(eval).Error
}

// Update 更新评估
func (r *EvaluationRepository) Update(ctx context.Context, eval *entity.SupplierEvaluation) error {
	return r.db.WithContext(ctx).Save(eval).Error
}

// FindBySupplier 查询某供应商的评估历史
func (r *EvaluationRepository) FindBySupplier(ctx context.Context, supplierID string) ([]entity.SupplierEvaluation, error) {
	var items []entity.SupplierEvaluation
	err := r.db.WithContext(ctx).
		Where("supplier_id = ?", supplierID).
		Order("period DESC").
		Find(&items).Error
	return items, err
}

// FindSupplierAvgScores 获取供应商平均评分
func (r *EvaluationRepository) FindSupplierAvgScores(ctx context.Context, supplierID string) (quality, delivery, price, service, total float64, err error) {
	var result struct {
		AvgQuality  float64
		AvgDelivery float64
		AvgPrice    float64
		AvgService  float64
		AvgTotal    float64
	}
	err = r.db.WithContext(ctx).
		Model(&entity.SupplierEvaluation{}).
		Select(`COALESCE(AVG(quality_score), 0) as avg_quality,
			COALESCE(AVG(delivery_score), 0) as avg_delivery,
			COALESCE(AVG(price_score), 0) as avg_price,
			COALESCE(AVG(service_score), 0) as avg_service,
			COALESCE(AVG(total_score), 0) as avg_total`).
		Where("supplier_id = ? AND status = ?", supplierID, entity.EvalStatusApproved).
		Scan(&result).Error
	return result.AvgQuality, result.AvgDelivery, result.AvgPrice, result.AvgService, result.AvgTotal, err
}
