package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 看板处理器
type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetSamplingProgress 打样进度
// GET /api/v1/srm/dashboard/sampling-progress?project_id=xxx
func (h *DashboardHandler) GetSamplingProgress(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		BadRequest(c, "project_id不能为空")
		return
	}

	progress, err := h.svc.GetSamplingProgress(c.Request.Context(), projectID)
	if err != nil {
		InternalError(c, "获取打样进度失败: "+err.Error())
		return
	}

	Success(c, progress)
}
