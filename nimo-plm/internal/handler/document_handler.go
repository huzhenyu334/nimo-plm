package handler

import (
	"io"
	"strconv"

	"github.com/bitfantasy/nimo-plm/internal/service"
	"github.com/gin-gonic/gin"
)

// DocumentHandler 文档处理器
type DocumentHandler struct {
	svc *service.DocumentService
}

// NewDocumentHandler 创建文档处理器
func NewDocumentHandler(svc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// List 获取文档列表
func (h *DocumentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filters := map[string]interface{}{
		"keyword":      c.Query("keyword"),
		"category_id":  c.Query("category_id"),
		"status":       c.Query("status"),
		"related_type": c.Query("related_type"),
		"related_id":   c.Query("related_id"),
		"uploaded_by":  c.Query("uploaded_by"),
	}

	result, err := h.svc.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// Get 获取文档详情
func (h *DocumentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	doc, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Document not found")
		return
	}

	Success(c, doc)
}

// Upload 上传文档
func (h *DocumentHandler) Upload(c *gin.Context) {
	// 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "File is required: "+err.Error())
		return
	}
	defer file.Close()

	// 解析请求参数
	req := &service.UploadDocumentRequest{
		Title:       c.PostForm("title"),
		CategoryID:  c.PostForm("category_id"),
		RelatedType: c.PostForm("related_type"),
		RelatedID:   c.PostForm("related_id"),
		Description: c.PostForm("description"),
	}

	if req.Title == "" {
		req.Title = header.Filename
	}

	userID := GetUserID(c)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	doc, err := h.svc.Upload(c.Request.Context(), userID, req, file, header.Filename, header.Size, contentType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Created(c, doc)
}

// Update 更新文档信息
func (h *DocumentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	var req service.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	doc, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, doc)
}

// UploadNewVersion 上传新版本
func (h *DocumentHandler) UploadNewVersion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	// 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "File is required: "+err.Error())
		return
	}
	defer file.Close()

	changeSummary := c.PostForm("change_summary")

	userID := GetUserID(c)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	doc, err := h.svc.UploadNewVersion(c.Request.Context(), id, userID, file, header.Filename, header.Size, contentType, changeSummary)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, doc)
}

// Delete 删除文档
func (h *DocumentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, nil)
}

// Release 发布文档
func (h *DocumentHandler) Release(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	userID := GetUserID(c)
	doc, err := h.svc.Release(c.Request.Context(), id, userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, doc)
}

// Obsolete 废弃文档
func (h *DocumentHandler) Obsolete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	doc, err := h.svc.Obsolete(c.Request.Context(), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, doc)
}

// Download 下载文档
func (h *DocumentHandler) Download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	reader, doc, err := h.svc.Download(c.Request.Context(), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename="+doc.FileName)
	c.Header("Content-Type", doc.MimeType)
	c.Header("Content-Length", strconv.FormatInt(doc.FileSize, 10))

	io.Copy(c.Writer, reader)
}

// ListVersions 获取文档版本列表
func (h *DocumentHandler) ListVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Document ID is required")
		return
	}

	versions, err := h.svc.ListVersions(c.Request.Context(), id)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, versions)
}

// DownloadVersion 下载指定版本
func (h *DocumentHandler) DownloadVersion(c *gin.Context) {
	versionID := c.Param("versionId")
	if versionID == "" {
		BadRequest(c, "Version ID is required")
		return
	}

	reader, version, err := h.svc.DownloadVersion(c.Request.Context(), versionID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename="+version.FileName)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(version.FileSize, 10))

	io.Copy(c.Writer, reader)
}

// ListByRelated 获取关联对象的文档列表
func (h *DocumentHandler) ListByRelated(c *gin.Context) {
	relatedType := c.Query("related_type")
	relatedID := c.Query("related_id")

	if relatedType == "" || relatedID == "" {
		BadRequest(c, "related_type and related_id are required")
		return
	}

	docs, err := h.svc.ListByRelated(c.Request.Context(), relatedType, relatedID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, docs)
}

// ListCategories 获取文档分类列表
func (h *DocumentHandler) ListCategories(c *gin.Context) {
	tree := c.Query("tree") == "true"

	var cats interface{}
	var err error

	if tree {
		cats, err = h.svc.ListCategoryTree(c.Request.Context())
	} else {
		cats, err = h.svc.ListCategories(c.Request.Context())
	}

	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, cats)
}

// GetCategory 获取分类详情
func (h *DocumentHandler) GetCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequest(c, "Category ID is required")
		return
	}

	cat, err := h.svc.GetCategory(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Category not found")
		return
	}

	Success(c, cat)
}
