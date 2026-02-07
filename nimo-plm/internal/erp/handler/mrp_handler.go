package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type MRPHandler struct {
	svc *service.MRPService
}

func NewMRPHandler(svc *service.MRPService) *MRPHandler {
	return &MRPHandler{svc: svc}
}

func (h *MRPHandler) Run(c *gin.Context) {
	var req service.RunMRPRequest
	c.ShouldBindJSON(&req)
	userID, _ := c.Get("user_id")
	run, err := h.svc.Run(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": run})
}

func (h *MRPHandler) GetResult(c *gin.Context) {
	runID := c.Query("run_id")
	if runID == "" {
		// 获取最近一次MRP运行
		run, err := h.svc.GetLatestRun()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "没有MRP运行记录"})
			return
		}
		runID = run.ID
	}
	results, err := h.svc.GetResults(runID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": results})
}

func (h *MRPHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	runs, total, err := h.svc.ListRuns(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": runs, "total": total, "page": page, "size": size}})
}

func (h *MRPHandler) Apply(c *gin.Context) {
	var req struct {
		RunID string `json:"run_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Apply(req.RunID, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
