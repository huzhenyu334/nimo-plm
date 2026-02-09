package service

import (
	"context"

	"gorm.io/gorm"
)

// DashboardService 看板服务
type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// SamplingProgress 打样进度
type SamplingProgress struct {
	ProjectID    string `json:"project_id"`
	TotalItems   int    `json:"total_items"`
	OrderedItems int    `json:"ordered_items"`
	ReceivedItems int   `json:"received_items"`
	InspectedItems int  `json:"inspected_items"`
	PassedItems  int    `json:"passed_items"`
	ProgressPct  float64 `json:"progress_pct"`
}

// GetSamplingProgress 获取打样进度
func (s *DashboardService) GetSamplingProgress(ctx context.Context, projectID string) (*SamplingProgress, error) {
	progress := &SamplingProgress{
		ProjectID: projectID,
	}

	// 统计该项目下所有PR行项
	row := s.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN i.status IN ('ordered','received','inspected','completed') THEN 1 END) as ordered,
			COUNT(CASE WHEN i.status IN ('received','inspected','completed') THEN 1 END) as received,
			COUNT(CASE WHEN i.status IN ('inspected','completed') THEN 1 END) as inspected,
			COUNT(CASE WHEN i.status = 'completed' THEN 1 END) as passed
		FROM srm_pr_items i
		JOIN srm_purchase_requests pr ON pr.id = i.pr_id
		WHERE pr.project_id = ?
	`, projectID).Row()

	if err := row.Scan(
		&progress.TotalItems,
		&progress.OrderedItems,
		&progress.ReceivedItems,
		&progress.InspectedItems,
		&progress.PassedItems,
	); err != nil {
		return progress, nil // 没有数据时返回空进度
	}

	if progress.TotalItems > 0 {
		progress.ProgressPct = float64(progress.PassedItems) / float64(progress.TotalItems) * 100
	}

	return progress, nil
}
