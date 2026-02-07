package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type ManufacturingHandler struct {
	svc *service.ManufacturingService
}

func NewManufacturingHandler(svc *service.ManufacturingService) *ManufacturingHandler {
	return &ManufacturingHandler{svc: svc}
}

func (h *ManufacturingHandler) Create(c *gin.Context) {
	var req service.CreateWorkOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	wo, err := h.svc.Create(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": wo})
}

func (h *ManufacturingHandler) Get(c *gin.Context) {
	wo, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "工单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": wo})
}

func (h *ManufacturingHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.WOListParams{
		Status:    c.Query("status"),
		ProductID: c.Query("product_id"),
		Keyword:   c.Query("keyword"),
		Page:      page,
		Size:      size,
	}
	wos, total, err := h.svc.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": wos, "total": total, "page": page, "size": size}})
}

func (h *ManufacturingHandler) Release(c *gin.Context) {
	if err := h.svc.Release(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ManufacturingHandler) Pick(c *gin.Context) {
	var req struct {
		WarehouseID string `json:"warehouse_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Pick(c.Param("id"), req.WarehouseID, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ManufacturingHandler) Report(c *gin.Context) {
	var req service.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Report(c.Param("id"), req, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ManufacturingHandler) Complete(c *gin.Context) {
	var req struct {
		WarehouseID string `json:"warehouse_id"`
	}
	c.ShouldBindJSON(&req)
	userID, _ := c.Get("user_id")
	if err := h.svc.Complete(c.Param("id"), req.WarehouseID, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
