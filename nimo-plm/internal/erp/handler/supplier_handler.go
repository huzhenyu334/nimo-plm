package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct {
	svc *service.SupplierService
}

func NewSupplierHandler(svc *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

func (h *SupplierHandler) Create(c *gin.Context) {
	var req service.CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": "参数校验失败: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	supplier, err := h.svc.Create(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": supplier})
}

func (h *SupplierHandler) Get(c *gin.Context) {
	id := c.Param("id")
	supplier, err := h.svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "供应商不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": supplier})
}

func (h *SupplierHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": "参数校验失败: " + err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	supplier, err := h.svc.Update(id, req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": supplier})
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *SupplierHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	params := repository.SupplierListParams{
		Status:  c.Query("status"),
		Type:    c.Query("type"),
		Rating:  c.Query("rating"),
		Keyword: c.Query("keyword"),
		Page:    page,
		Size:    size,
	}

	suppliers, total, err := h.svc.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": suppliers,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

func (h *SupplierHandler) UpdateScore(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": "参数校验失败: " + err.Error()})
		return
	}

	if err := h.svc.UpdateScore(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
