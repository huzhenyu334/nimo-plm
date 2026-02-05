package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/bitfantasy/nimo-plm/internal/model/entity"
	"github.com/bitfantasy/nimo-plm/internal/repository"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// DocumentService 文档服务
type DocumentService struct {
	docRepo     *repository.DocumentRepository
	catRepo     *repository.DocumentCategoryRepository
	minioClient *minio.Client
	bucketName  string
}

// NewDocumentService 创建文档服务
func NewDocumentService(
	docRepo *repository.DocumentRepository,
	catRepo *repository.DocumentCategoryRepository,
	minioClient *minio.Client,
	bucketName string,
) *DocumentService {
	return &DocumentService{
		docRepo:     docRepo,
		catRepo:     catRepo,
		minioClient: minioClient,
		bucketName:  bucketName,
	}
}

// CreateDocumentRequest 创建文档请求
type CreateDocumentRequest struct {
	Title       string `json:"title" binding:"required"`
	CategoryID  string `json:"category_id"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	Description string `json:"description"`
}

// UpdateDocumentRequest 更新文档请求
type UpdateDocumentRequest struct {
	Title       string `json:"title"`
	CategoryID  string `json:"category_id"`
	Description string `json:"description"`
}

// UploadDocumentRequest 上传文档请求
type UploadDocumentRequest struct {
	Title       string `json:"title" binding:"required"`
	CategoryID  string `json:"category_id"`
	RelatedType string `json:"related_type"`
	RelatedID   string `json:"related_id"`
	Description string `json:"description"`
}

// DocumentListResult 文档列表结果
type DocumentListResult struct {
	Items      []entity.Document `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// List 获取文档列表
func (s *DocumentService) List(ctx context.Context, page, pageSize int, filters map[string]interface{}) (*DocumentListResult, error) {
	docs, total, err := s.docRepo.List(ctx, page, pageSize, filters)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &DocumentListResult{
		Items:      docs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Get 获取文档详情
func (s *DocumentService) Get(ctx context.Context, id string) (*entity.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}
	return doc, nil
}

// Upload 上传文档
func (s *DocumentService) Upload(ctx context.Context, userID string, req *UploadDocumentRequest, reader io.Reader, fileName string, fileSize int64, contentType string) (*entity.Document, error) {
	// 生成文档编码
	code, err := s.docRepo.GenerateCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	// 生成存储路径
	objectName := fmt.Sprintf("documents/%s/%s%s", time.Now().Format("2006/01/02"), uuid.New().String()[:8], filepath.Ext(fileName))

	// 上传到MinIO
	if s.minioClient != nil {
		_, err = s.minioClient.PutObject(ctx, s.bucketName, objectName, reader, fileSize, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return nil, fmt.Errorf("upload file: %w", err)
		}
	}

	now := time.Now()
	doc := &entity.Document{
		ID:          uuid.New().String()[:32],
		Code:        code,
		Title:       req.Title,
		CategoryID:  req.CategoryID,
		RelatedType: req.RelatedType,
		RelatedID:   req.RelatedID,
		Status:      entity.DocumentStatusDraft,
		Version:     "1.0",
		Description: req.Description,
		FileName:    fileName,
		FilePath:    objectName,
		FileSize:    fileSize,
		MimeType:    contentType,
		UploadedBy:  userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	// 创建初始版本记录
	version := &entity.DocumentVersion{
		ID:         uuid.New().String()[:32],
		DocumentID: doc.ID,
		Version:    "1.0",
		FileName:   fileName,
		FilePath:   objectName,
		FileSize:   fileSize,
		CreatedBy:  userID,
		CreatedAt:  now,
	}

	if err := s.docRepo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}

	return doc, nil
}

// Update 更新文档信息
func (s *DocumentService) Update(ctx context.Context, id string, req *UpdateDocumentRequest) (*entity.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}

	if req.Title != "" {
		doc.Title = req.Title
	}
	if req.CategoryID != "" {
		doc.CategoryID = req.CategoryID
	}
	if req.Description != "" {
		doc.Description = req.Description
	}

	doc.UpdatedAt = time.Now()

	if err := s.docRepo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	return doc, nil
}

// UploadNewVersion 上传新版本
func (s *DocumentService) UploadNewVersion(ctx context.Context, id string, userID string, reader io.Reader, fileName string, fileSize int64, contentType string, changeSummary string) (*entity.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}

	// 计算新版本号
	latestVersion, _ := s.docRepo.GetLatestVersion(ctx, id)
	newVersion := "1.0"
	if latestVersion != nil {
		// 简单的版本号递增逻辑
		var major, minor int
		fmt.Sscanf(latestVersion.Version, "%d.%d", &major, &minor)
		newVersion = fmt.Sprintf("%d.%d", major, minor+1)
	}

	// 生成存储路径
	objectName := fmt.Sprintf("documents/%s/%s%s", time.Now().Format("2006/01/02"), uuid.New().String()[:8], filepath.Ext(fileName))

	// 上传到MinIO
	if s.minioClient != nil {
		_, err = s.minioClient.PutObject(ctx, s.bucketName, objectName, reader, fileSize, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return nil, fmt.Errorf("upload file: %w", err)
		}
	}

	// 创建版本记录
	version := &entity.DocumentVersion{
		ID:            uuid.New().String()[:32],
		DocumentID:    doc.ID,
		Version:       newVersion,
		FileName:      fileName,
		FilePath:      objectName,
		FileSize:      fileSize,
		ChangeSummary: changeSummary,
		CreatedBy:     userID,
		CreatedAt:     time.Now(),
	}

	if err := s.docRepo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}

	// 更新文档主记录
	doc.Version = newVersion
	doc.FileName = fileName
	doc.FilePath = objectName
	doc.FileSize = fileSize
	doc.MimeType = contentType
	doc.UpdatedAt = time.Now()

	if err := s.docRepo.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("update document: %w", err)
	}

	return doc, nil
}

// Delete 删除文档
func (s *DocumentService) Delete(ctx context.Context, id string) error {
	return s.docRepo.Delete(ctx, id)
}

// Release 发布文档
func (s *DocumentService) Release(ctx context.Context, id string, userID string) (*entity.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}

	if doc.Status != entity.DocumentStatusDraft {
		return nil, fmt.Errorf("document is not in draft status")
	}

	if err := s.docRepo.Release(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("release document: %w", err)
	}

	return s.docRepo.FindByID(ctx, id)
}

// Obsolete 废弃文档
func (s *DocumentService) Obsolete(ctx context.Context, id string) (*entity.Document, error) {
	if err := s.docRepo.Obsolete(ctx, id); err != nil {
		return nil, fmt.Errorf("obsolete document: %w", err)
	}
	return s.docRepo.FindByID(ctx, id)
}

// ListVersions 获取文档版本列表
func (s *DocumentService) ListVersions(ctx context.Context, documentID string) ([]entity.DocumentVersion, error) {
	return s.docRepo.ListVersions(ctx, documentID)
}

// GetVersion 获取文档版本详情
func (s *DocumentService) GetVersion(ctx context.Context, versionID string) (*entity.DocumentVersion, error) {
	return s.docRepo.FindVersionByID(ctx, versionID)
}

// Download 下载文档
func (s *DocumentService) Download(ctx context.Context, id string) (io.ReadCloser, *entity.Document, error) {
	doc, err := s.docRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("find document: %w", err)
	}

	if s.minioClient == nil {
		return nil, doc, fmt.Errorf("storage not configured")
	}

	object, err := s.minioClient.GetObject(ctx, s.bucketName, doc.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get object: %w", err)
	}

	return object, doc, nil
}

// DownloadVersion 下载指定版本
func (s *DocumentService) DownloadVersion(ctx context.Context, versionID string) (io.ReadCloser, *entity.DocumentVersion, error) {
	version, err := s.docRepo.FindVersionByID(ctx, versionID)
	if err != nil {
		return nil, nil, fmt.Errorf("find version: %w", err)
	}

	if s.minioClient == nil {
		return nil, version, fmt.Errorf("storage not configured")
	}

	object, err := s.minioClient.GetObject(ctx, s.bucketName, version.FilePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get object: %w", err)
	}

	return object, version, nil
}

// ListByRelated 获取关联对象的文档列表
func (s *DocumentService) ListByRelated(ctx context.Context, relatedType, relatedID string) ([]entity.Document, error) {
	return s.docRepo.ListByRelated(ctx, relatedType, relatedID)
}

// ListCategories 获取文档分类列表
func (s *DocumentService) ListCategories(ctx context.Context) ([]entity.DocumentCategory, error) {
	return s.catRepo.List(ctx)
}

// ListCategoryTree 获取文档分类树
func (s *DocumentService) ListCategoryTree(ctx context.Context) ([]entity.DocumentCategory, error) {
	return s.catRepo.ListTree(ctx)
}

// GetCategory 获取分类详情
func (s *DocumentService) GetCategory(ctx context.Context, id string) (*entity.DocumentCategory, error) {
	return s.catRepo.FindByID(ctx, id)
}
