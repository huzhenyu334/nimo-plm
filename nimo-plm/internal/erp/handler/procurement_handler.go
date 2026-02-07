package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type ProcurementHandler struct {
	svc *service.ProcurementService
}

func NewProcurementHandler(svc *service.ProcurementService) *ProcurementHandler {
	return &ProcurementHandler{svc: svc}
}

// --- PR ---

func (h *ProcurementHandler) CreatePR(c *gin.Context) {
	var req service.CreatePRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	pr, err := h.svc.CreatePR(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pr})
}

func (h *ProcurementHandler) ListPRs(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	prs, total, err := h.svc.ListPRs(status, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": prs, "total": total, "page": page, "size": size}})
}

func (h *ProcurementHandler) ApprovePR(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	if err := h.svc.ApprovePR(id, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// --- PO ---

func (h *ProcurementHandler) CreatePO(c *gin.Context) {
	var req service.CreatePORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	po, err := h.svc.CreatePO(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": po})
}

func (h *ProcurementHandler) GetPO(c *gin.Context) {
	id := c.Param("id")
	po, err := h.svc.GetPOByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "采购订单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": po})
}

func (h *ProcurementHandler) ListPOs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.POListParams{
		Status:     c.Query("status"),
		SupplierID: c.Query("supplier_id"),
		Keyword:    c.Query("keyword"),
		Page:       page,
		Size:       size,
	}
	pos, total, err := h.svc.ListPOs(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": pos, "total": total, "page": page, "size": size}})
}

func (h *ProcurementHandler) SubmitPO(c *gin.Context) {
	if err := h.svc.SubmitPO(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ProcurementHandler) ApprovePO(c *gin.Context) {
	userID, _ := c.Get("user_id")
	if err := h.svc.ApprovePO(c.Param("id"), userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ProcurementHandler) RejectPO(c *gin.Context) {
	if err := h.svc.RejectPO(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ProcurementHandler) SendPO(c *gin.Context) {
	if err := h.svc.SendPO(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *ProcurementHandler) ReceivePO(c *gin.Context) {
	var req struct {
		Items []service.ReceiveItemRequest `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	if err := h.svc.ReceivePO(c.Param("id"), req.Items, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
