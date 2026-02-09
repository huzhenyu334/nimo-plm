package handler

import (
	"strconv"

	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// Handlers SRM处理器集合
type Handlers struct {
	Supplier   *SupplierHandler
	PR         *PRHandler
	PO         *POHandler
	Inspection *InspectionHandler
	Dashboard  *DashboardHandler
}

// NewHandlers 创建SRM处理器集合
func NewHandlers(
	supplierSvc *service.SupplierService,
	procurementSvc *service.ProcurementService,
	inspectionSvc *service.InspectionService,
	dashboardSvc *service.DashboardService,
	poRepo POItemReceiver,
) *Handlers {
	return &Handlers{
		Supplier:   NewSupplierHandler(supplierSvc),
		PR:         NewPRHandler(procurementSvc),
		PO:         NewPOHandler(procurementSvc, poRepo),
		Inspection: NewInspectionHandler(inspectionSvc),
		Dashboard:  NewDashboardHandler(dashboardSvc),
	}
}

// === 响应辅助函数（与PLM保持一致） ===

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ListResponse struct {
	Items      interface{} `json:"items"`
	Pagination *Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(201, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	statusCode := code / 100
	if statusCode < 100 || statusCode > 599 {
		statusCode = 500
	}
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, 40000, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, 40400, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, 40300, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, 50000, message)
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

func GetPagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	return page, pageSize
}
