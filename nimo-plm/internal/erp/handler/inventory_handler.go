package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	svc *service.InventoryService
}

func NewInventoryHandler(svc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

func (h *InventoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.InventoryListParams{
		MaterialID:    c.Query("material_id"),
		WarehouseID:   c.Query("warehouse_id"),
		InventoryType: c.Query("inventory_type"),
		Keyword:       c.Query("keyword"),
		LowStock:      c.Query("low_stock") == "true",
		Page:          page,
		Size:          size,
	}
	items, total, err := h.svc.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": items, "total": total, "page": page, "size": size}})
}

func (h *InventoryHandler) GetByMaterial(c *gin.Context) {
	materialID := c.Param("material_id")
	items, err := h.svc.GetByMaterial(materialID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

func (h *InventoryHandler) Inbound(c *gin.Context) {
	var req service.InboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Inbound(req, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *InventoryHandler) Outbound(c *gin.Context) {
	var req service.OutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Outbound(req, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *InventoryHandler) Adjust(c *gin.Context) {
	var req service.AdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.Adjust(req, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *InventoryHandler) Alerts(c *gin.Context) {
	alerts, err := h.svc.GetAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": alerts})
}

func (h *InventoryHandler) Transactions(c *gin.Context) {
	materialID := c.Query("material_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	txs, total, err := h.svc.ListTransactions(materialID, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": txs, "total": total, "page": page, "size": size}})
}
