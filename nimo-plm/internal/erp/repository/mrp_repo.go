package repository

import (
	"github.com/bitfantasy/nimo-plm/internal/erp/entity"
	"gorm.io/gorm"
)

type MRPRepository struct {
	db *gorm.DB
}

func NewMRPRepository(db *gorm.DB) *MRPRepository {
	return &MRPRepository{db: db}
}

func (r *MRPRepository) CreateRun(run *entity.MRPRun) error {
	return r.db.Create(run).Error
}

func (r *MRPRepository) GetRunByID(id string) (*entity.MRPRun, error) {
	var run entity.MRPRun
	err := r.db.Where("id = ?", id).First(&run).Error
	return &run, err
}

func (r *MRPRepository) UpdateRun(run *entity.MRPRun) error {
	return r.db.Save(run).Error
}

func (r *MRPRepository) GetLatestRun() (*entity.MRPRun, error) {
	var run entity.MRPRun
	err := r.db.Order("created_at DESC").First(&run).Error
	return &run, err
}

func (r *MRPRepository) ListRuns(page, size int) ([]entity.MRPRun, int64, error) {
	var total int64
	r.db.Model(&entity.MRPRun{}).Count(&total)
	if page <= 0 { page = 1 }
	if size <= 0 { size = 20 }
	var runs []entity.MRPRun
	err := r.db.Order("created_at DESC").Offset((page-1)*size).Limit(size).Find(&runs).Error
	return runs, total, err
}

func (r *MRPRepository) BatchCreateResults(results []entity.MRPResult) error {
	if len(results) == 0 {
		return nil
	}
	return r.db.Create(&results).Error
}

func (r *MRPRepository) GetResultsByRunID(runID string) ([]entity.MRPResult, error) {
	var results []entity.MRPResult
	err := r.db.Where("mrp_run_id = ?", runID).Order("material_code").Find(&results).Error
	return results, err
}

func (r *MRPRepository) GetUnappliedResults(runID string) ([]entity.MRPResult, error) {
	var results []entity.MRPResult
	err := r.db.Where("mrp_run_id = ? AND applied = false AND net_requirement > 0", runID).Find(&results).Error
	return results, err
}

func (r *MRPRepository) MarkResultsApplied(runID string) error {
	return r.db.Model(&entity.MRPResult{}).Where("mrp_run_id = ?", runID).Update("applied", true).Error
}

// --- Finance ---

func (r *MRPRepository) CreateFinanceRecord(record *entity.FinanceRecord) error {
	return r.db.Create(record).Error
}

func (r *MRPRepository) UpdateFinanceRecord(record *entity.FinanceRecord) error {
	return r.db.Save(record).Error
}

func (r *MRPRepository) GetFinanceRecordByID(id string) (*entity.FinanceRecord, error) {
	var record entity.FinanceRecord
	err := r.db.Where("id = ?", id).First(&record).Error
	return &record, err
}

type FinanceListParams struct {
	RecordType string
	Status     string
	Page       int
	Size       int
}

func (r *MRPRepository) ListFinanceRecords(params FinanceListParams) ([]entity.FinanceRecord, int64, error) {
	query := r.db.Model(&entity.FinanceRecord{})
	if params.RecordType != "" {
		query = query.Where("record_type = ?", params.RecordType)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	var total int64
	query.Count(&total)
	if params.Page <= 0 { params.Page = 1 }
	if params.Size <= 0 { params.Size = 20 }
	var records []entity.FinanceRecord
	err := query.Order("created_at DESC").Offset((params.Page-1)*params.Size).Limit(params.Size).Find(&records).Error
	return records, total, err
}

func (r *MRPRepository) DB() *gorm.DB {
	return r.db
}
