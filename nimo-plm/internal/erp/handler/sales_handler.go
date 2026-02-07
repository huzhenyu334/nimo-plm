package handler

import (
	"net/http"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/erp/repository"
	"github.com/bitfantasy/nimo-plm/internal/erp/service"
	"github.com/gin-gonic/gin"
)

type SalesHandler struct {
	svc *service.SalesService
}

func NewSalesHandler(svc *service.SalesService) *SalesHandler {
	return &SalesHandler{svc: svc}
}

// --- Customer ---

func (h *SalesHandler) CreateCustomer(c *gin.Context) {
	var req service.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	customer, err := h.svc.CreateCustomer(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": customer})
}

func (h *SalesHandler) GetCustomer(c *gin.Context) {
	customer, err := h.svc.GetCustomerByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "客户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": customer})
}

func (h *SalesHandler) ListCustomers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.CustomerListParams{
		Status:  c.Query("status"),
		Type:    c.Query("type"),
		Keyword: c.Query("keyword"),
		Page:    page,
		Size:    size,
	}
	customers, total, err := h.svc.ListCustomers(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": customers, "total": total, "page": page, "size": size}})
}

func (h *SalesHandler) DeleteCustomer(c *gin.Context) {
	if err := h.svc.DeleteCustomer(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// --- Sales Order ---

func (h *SalesHandler) CreateSO(c *gin.Context) {
	var req service.CreateSORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	so, err := h.svc.CreateSO(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": so})
}

func (h *SalesHandler) GetSO(c *gin.Context) {
	so, err := h.svc.GetSOByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "销售订单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": so})
}

func (h *SalesHandler) ListSOs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.SOListParams{
		Status:     c.Query("status"),
		CustomerID: c.Query("customer_id"),
		Channel:    c.Query("channel"),
		Keyword:    c.Query("keyword"),
		Page:       page,
		Size:       size,
	}
	sos, total, err := h.svc.ListSOs(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": sos, "total": total, "page": page, "size": size}})
}

func (h *SalesHandler) ConfirmSO(c *gin.Context) {
	if err := h.svc.ConfirmSO(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *SalesHandler) ShipSO(c *gin.Context) {
	var req struct {
		TrackingNo string `json:"tracking_no"`
	}
	c.ShouldBindJSON(&req)
	userID, _ := c.Get("user_id")
	if err := h.svc.ShipSO(c.Param("id"), req.TrackingNo, userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *SalesHandler) CancelSO(c *gin.Context) {
	if err := h.svc.CancelSO(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// --- Service Order ---

func (h *SalesHandler) CreateServiceOrder(c *gin.Context) {
	var req service.CreateServiceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	so, err := h.svc.CreateServiceOrder(req, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": so})
}

func (h *SalesHandler) GetServiceOrder(c *gin.Context) {
	so, err := h.svc.GetServiceOrderByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 10002, "message": "服务工单不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": so})
}

func (h *SalesHandler) ListServiceOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	params := repository.ServiceOrderListParams{
		Status:      c.Query("status"),
		ServiceType: c.Query("service_type"),
		CustomerID:  c.Query("customer_id"),
		AssigneeID:  c.Query("assignee_id"),
		Keyword:     c.Query("keyword"),
		Page:        page,
		Size:        size,
	}
	orders, total, err := h.svc.ListServiceOrders(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50001, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"items": orders, "total": total, "page": page, "size": size}})
}

func (h *SalesHandler) AssignServiceOrder(c *gin.Context) {
	var req struct {
		AssigneeID   string `json:"assignee_id" binding:"required"`
		AssigneeName string `json:"assignee_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10001, "message": err.Error()})
		return
	}
	if err := h.svc.AssignServiceOrder(c.Param("id"), req.AssigneeID, req.AssigneeName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *SalesHandler) CompleteServiceOrder(c *gin.Context) {
	var req struct {
		Solution string `json:"solution"`
	}
	c.ShouldBindJSON(&req)
	if err := h.svc.CompleteServiceOrder(c.Param("id"), req.Solution); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 10004, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}
