package handler

import (
	"github.com/bitfantasy/nimo/internal/srm/service"
	"github.com/gin-gonic/gin"
)

// SupplierHandler 供应商处理器
type SupplierHandler struct {
	svc *service.SupplierService
}

func NewSupplierHandler(svc *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{svc: svc}
}

// ListSuppliers 供应商列表
// GET /api/v1/srm/suppliers?search=xxx&category=xxx&level=xxx&status=xxx&page=1&page_size=20
func (h *SupplierHandler) ListSuppliers(c *gin.Context) {
	page, pageSize := GetPagination(c)
	filters := map[string]string{
		"search":   c.Query("search"),
		"category": c.Query("category"),
		"level":    c.Query("level"),
		"status":   c.Query("status"),
	}

	items, total, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, "获取供应商列表失败: "+err.Error())
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	Success(c, ListResponse{
		Items: items,
		Pagination: &Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      int(total),
			TotalPages: totalPages,
		},
	})
}

// GetSupplier 供应商详情
// GET /api/v1/srm/suppliers/:id
func (h *SupplierHandler) GetSupplier(c *gin.Context) {
	id := c.Param("id")
	supplier, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "供应商不存在")
		return
	}
	Success(c, supplier)
}

// CreateSupplier 创建供应商
// POST /api/v1/srm/suppliers
func (h *SupplierHandler) CreateSupplier(c *gin.Context) {
	var req service.CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := GetUserID(c)
	supplier, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		InternalError(c, "创建供应商失败: "+err.Error())
		return
	}

	Created(c, supplier)
}

// UpdateSupplier 更新供应商
// PUT /api/v1/srm/suppliers/:id
func (h *SupplierHandler) UpdateSupplier(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	supplier, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, "更新供应商失败: "+err.Error())
		return
	}

	Success(c, supplier)
}

// DeleteSupplier 删除供应商
// DELETE /api/v1/srm/suppliers/:id
func (h *SupplierHandler) DeleteSupplier(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		InternalError(c, "删除供应商失败: "+err.Error())
		return
	}
	Success(c, nil)
}

// ListContacts 供应商联系人列表
// GET /api/v1/srm/suppliers/:id/contacts
func (h *SupplierHandler) ListContacts(c *gin.Context) {
	supplierID := c.Param("id")
	contacts, err := h.svc.ListContacts(c.Request.Context(), supplierID)
	if err != nil {
		InternalError(c, "获取联系人列表失败: "+err.Error())
		return
	}
	Success(c, gin.H{"items": contacts})
}

// CreateContact 创建联系人
// POST /api/v1/srm/suppliers/:id/contacts
func (h *SupplierHandler) CreateContact(c *gin.Context) {
	supplierID := c.Param("id")
	var req service.CreateContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	contact, err := h.svc.CreateContact(c.Request.Context(), supplierID, &req)
	if err != nil {
		InternalError(c, "创建联系人失败: "+err.Error())
		return
	}

	Created(c, contact)
}

// DeleteContact 删除联系人
// DELETE /api/v1/srm/suppliers/:id/contacts/:contactId
func (h *SupplierHandler) DeleteContact(c *gin.Context) {
	contactID := c.Param("contactId")
	if err := h.svc.DeleteContact(c.Request.Context(), contactID); err != nil {
		InternalError(c, "删除联系人失败: "+err.Error())
		return
	}
	Success(c, nil)
}
