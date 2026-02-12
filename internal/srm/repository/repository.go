package repository

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("record not found")
)

// Repositories SRM仓库集合
type Repositories struct {
	Supplier         *SupplierRepository
	PR               *PRRepository
	PO               *PORepository
	Inspection       *InspectionRepository
	Project          *ProjectRepository
	ActivityLog      *ActivityLogRepository
	DelayRequest     *DelayRequestRepository
	Settlement       *SettlementRepository
	CorrectiveAction *CorrectiveActionRepository
	Evaluation       *EvaluationRepository
	Equipment        *EquipmentRepository
	RFQ              *RFQRepository
	Sampling         *SamplingRepository
}

// NewRepositories 创建SRM仓库集合
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Supplier:         NewSupplierRepository(db),
		PR:               NewPRRepository(db),
		PO:               NewPORepository(db),
		Inspection:       NewInspectionRepository(db),
		Project:          NewProjectRepository(db),
		ActivityLog:      NewActivityLogRepository(db),
		DelayRequest:     NewDelayRequestRepository(db),
		Settlement:       NewSettlementRepository(db),
		CorrectiveAction: NewCorrectiveActionRepository(db),
		Evaluation:       NewEvaluationRepository(db),
		Equipment:        NewEquipmentRepository(db),
		RFQ:              NewRFQRepository(db),
		Sampling:         NewSamplingRepository(db),
	}
}
