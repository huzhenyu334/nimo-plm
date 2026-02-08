package handler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler 文件上传处理器
type UploadHandler struct{}

// NewUploadHandler 创建文件上传处理器
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadedFile 上传文件信息
type UploadedFile struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Upload 处理文件上传
// POST /upload
func (h *UploadHandler) Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		BadRequest(c, "无法解析上传文件: "+err.Error())
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		// 也尝试获取单文件
		files = form.File["file"]
	}
	if len(files) == 0 {
		BadRequest(c, "没有上传文件")
		return
	}

	now := time.Now()
	dir := fmt.Sprintf("./uploads/%d/%02d", now.Year(), now.Month())

	// 创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		InternalError(c, "创建上传目录失败: "+err.Error())
		return
	}

	var uploaded []UploadedFile

	for _, fileHeader := range files {
		fileID := uuid.New().String()[:32]
		ext := filepath.Ext(fileHeader.Filename)
		savedName := fmt.Sprintf("%s_%s%s", fileID, fileHeader.Filename, "")
		if ext != "" {
			savedName = fmt.Sprintf("%s_%s", fileID, fileHeader.Filename)
		}
		savePath := filepath.Join(dir, savedName)

		// 打开源文件
		src, err := fileHeader.Open()
		if err != nil {
			InternalError(c, "读取上传文件失败: "+err.Error())
			return
		}

		// 创建目标文件
		dst, err := os.Create(savePath)
		if err != nil {
			src.Close()
			InternalError(c, "保存文件失败: "+err.Error())
			return
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			InternalError(c, "写入文件失败: "+err.Error())
			return
		}

		url := fmt.Sprintf("/uploads/%d/%02d/%s", now.Year(), now.Month(), savedName)

		uploaded = append(uploaded, UploadedFile{
			ID:          fileID,
			URL:         url,
			Filename:    fileHeader.Filename,
			Size:        fileHeader.Size,
			ContentType: fileHeader.Header.Get("Content-Type"),
		})
	}

	Success(c, uploaded)
}
