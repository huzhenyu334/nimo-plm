package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/bitfantasy/nimo/internal/plm/service"
	"github.com/bitfantasy/nimo/internal/plm/testutil"
	"github.com/gin-gonic/gin"
)

func setupDocumentTest(t *testing.T) (*gin.Engine, *testutil.TestEnv) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	// Create document tables via raw SQL (AutoMigrate has issues with default:1.0 tag)
	db.Exec(`CREATE TABLE IF NOT EXISTS document_categories (
		id VARCHAR(32) PRIMARY KEY,
		code VARCHAR(32) NOT NULL UNIQUE,
		name VARCHAR(64) NOT NULL,
		parent_id VARCHAR(32),
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS documents (
		id VARCHAR(32) PRIMARY KEY,
		code VARCHAR(64) NOT NULL UNIQUE,
		title VARCHAR(256) NOT NULL,
		category_id VARCHAR(32),
		related_type VARCHAR(32),
		related_id VARCHAR(32),
		status VARCHAR(16) NOT NULL DEFAULT 'draft',
		version VARCHAR(16) NOT NULL DEFAULT '1.0',
		description TEXT,
		file_name VARCHAR(256) NOT NULL,
		file_path VARCHAR(512) NOT NULL,
		file_size BIGINT NOT NULL,
		mime_type VARCHAR(128),
		feishu_doc_token VARCHAR(64),
		feishu_doc_url VARCHAR(512),
		uploaded_by VARCHAR(32) NOT NULL,
		released_by VARCHAR(32),
		released_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents(deleted_at)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS document_versions (
		id VARCHAR(32) PRIMARY KEY,
		document_id VARCHAR(32) NOT NULL,
		version VARCHAR(16) NOT NULL,
		file_name VARCHAR(256) NOT NULL,
		file_path VARCHAR(512) NOT NULL,
		file_size BIGINT NOT NULL,
		change_summary TEXT,
		created_by VARCHAR(32) NOT NULL,
		created_at TIMESTAMPTZ
	)`)

	// Create the document_code_seq sequence
	db.Exec("CREATE SEQUENCE IF NOT EXISTS document_code_seq START 1")

	// Seed test user
	testutil.SeedTestUser(t, db, "test-user-001", "Test Admin", "admin@test.com")

	// Seed a document category
	db.Create(&entity.DocumentCategory{
		ID:   "dcat_design",
		Code: "DESIGN",
		Name: "设计文档",
	})

	docRepo := repository.NewDocumentRepository(db)
	catRepo := repository.NewDocumentCategoryRepository(db)
	docSvc := service.NewDocumentService(docRepo, catRepo, nil, "")
	docHandler := NewDocumentHandler(docSvc)

	router := testutil.SetupRouter()
	api := testutil.AuthGroup(router, "/api/v1")

	docs := api.Group("/documents")
	docs.GET("", docHandler.List)
	docs.POST("", docHandler.Upload)
	docs.GET("/:id", docHandler.Get)
	docs.PUT("/:id", docHandler.Update)
	docs.DELETE("/:id", docHandler.Delete)
	docs.POST("/:id/release", docHandler.Release)
	docs.POST("/:id/obsolete", docHandler.Obsolete)
	docs.GET("/:id/versions", docHandler.ListVersions)
	docs.POST("/:id/versions", docHandler.UploadNewVersion)
	docs.GET("/:id/versions/:versionId/download", docHandler.DownloadVersion)

	api.GET("/document-categories", docHandler.ListCategories)

	return router, &testutil.TestEnv{DB: db, Router: router, T: t}
}

func uploadDocument(t *testing.T, router *gin.Engine, token, title, filename, content string) map[string]interface{} {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	io.Copy(part, strings.NewReader(content))
	writer.WriteField("title", title)
	writer.WriteField("category_id", "dcat_design")
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/documents", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["data"].(map[string]interface{})
}

func TestDocumentUpload(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Test Document", "test.pdf", "PDF content")

	if doc["id"] == nil || doc["id"] == "" {
		t.Error("Expected non-empty id")
	}
	if doc["title"] != "Test Document" {
		t.Errorf("Expected title 'Test Document', got %v", doc["title"])
	}
	if doc["version"] != "1.0" {
		t.Errorf("Expected version '1.0', got %v", doc["version"])
	}
	if doc["status"] != "draft" {
		t.Errorf("Expected status 'draft', got %v", doc["status"])
	}
	code, ok := doc["code"].(string)
	if !ok || !strings.HasPrefix(code, "DOC-") {
		t.Errorf("Expected code starting with 'DOC-', got %v", doc["code"])
	}
}

func TestDocumentList(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	// Upload two documents
	uploadDocument(t, router, token, "Doc 1", "doc1.pdf", "content 1")
	uploadDocument(t, router, token, "Doc 2", "doc2.pdf", "content 2")

	// List all
	w := testutil.DoRequest(router, "GET", "/api/v1/documents", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	total := data["total"].(float64)
	if total < 2 {
		t.Errorf("Expected at least 2 documents, got %v", total)
	}
}

func TestDocumentGet(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Get Test", "gettest.pdf", "content")
	docID := doc["id"].(string)

	w := testutil.DoRequest(router, "GET", "/api/v1/documents/"+docID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["title"] != "Get Test" {
		t.Errorf("Expected title 'Get Test', got %v", data["title"])
	}
}

func TestDocumentUpdate(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Old Title", "update.pdf", "content")
	docID := doc["id"].(string)

	w := testutil.DoRequest(router, "PUT", "/api/v1/documents/"+docID,
		map[string]string{"title": "New Title"}, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["title"] != "New Title" {
		t.Errorf("Expected title 'New Title', got %v", data["title"])
	}
}

func TestDocumentUploadNewVersion(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	// Upload initial document
	doc := uploadDocument(t, router, token, "Version Test", "v1.pdf", "v1 content")
	docID := doc["id"].(string)

	// Upload V2
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "v2.pdf")
	io.Copy(part, strings.NewReader("v2 content"))
	writer.WriteField("change_summary", "Updated to v2")
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/documents/%s/versions", docID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["version"] != "1.1" {
		t.Errorf("Expected version '1.1', got %v", data["version"])
	}

	// Upload V3
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	part2, _ := writer2.CreateFormFile("file", "v3.pdf")
	io.Copy(part2, strings.NewReader("v3 content"))
	writer2.WriteField("change_summary", "Updated to v3")
	writer2.Close()

	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/documents/%s/versions", docID), body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	req2.Header.Set("Authorization", "Bearer "+token)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp2 := testutil.ParseResponse(w2)
	data2 := resp2["data"].(map[string]interface{})
	if data2["version"] != "1.2" {
		t.Errorf("Expected version '1.2', got %v", data2["version"])
	}
}

func TestDocumentListVersions(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	// Upload document (creates V1.0)
	doc := uploadDocument(t, router, token, "Versions List Test", "initial.pdf", "content")
	docID := doc["id"].(string)

	// Upload V2
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "update.pdf")
	io.Copy(part, strings.NewReader("v2 content"))
	writer.WriteField("change_summary", "Version 2")
	writer.Close()

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/documents/%s/versions", docID), body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	wUp := httptest.NewRecorder()
	router.ServeHTTP(wUp, req)

	// List versions
	w := testutil.DoRequest(router, "GET", fmt.Sprintf("/api/v1/documents/%s/versions", docID), nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data to be array, got %T", resp["data"])
	}
	if len(data) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(data))
	}

	// Verify version order (newest first)
	if len(data) >= 2 {
		v1 := data[0].(map[string]interface{})
		v2 := data[1].(map[string]interface{})
		if v1["version"] != "1.1" {
			t.Errorf("Expected first version to be '1.1', got %v", v1["version"])
		}
		if v2["version"] != "1.0" {
			t.Errorf("Expected second version to be '1.0', got %v", v2["version"])
		}
		// Check change_summary on the v2 entry
		if v1["change_summary"] != "Version 2" {
			t.Errorf("Expected change_summary 'Version 2', got %v", v1["change_summary"])
		}
	}
}

func TestDocumentRelease(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Release Test", "release.pdf", "content")
	docID := doc["id"].(string)

	w := testutil.DoRequest(router, "POST", "/api/v1/documents/"+docID+"/release", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["status"] != "released" {
		t.Errorf("Expected status 'released', got %v", data["status"])
	}
}

func TestDocumentObsolete(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Obsolete Test", "obsolete.pdf", "content")
	docID := doc["id"].(string)

	w := testutil.DoRequest(router, "POST", "/api/v1/documents/"+docID+"/obsolete", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["status"] != "obsolete" {
		t.Errorf("Expected status 'obsolete', got %v", data["status"])
	}
}

func TestDocumentDelete(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	doc := uploadDocument(t, router, token, "Delete Test", "delete.pdf", "content")
	docID := doc["id"].(string)

	w := testutil.DoRequest(router, "DELETE", "/api/v1/documents/"+docID, nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get should fail after delete
	w2 := testutil.DoRequest(router, "GET", "/api/v1/documents/"+docID, nil, token)
	if w2.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", w2.Code)
	}
}

func TestDocumentListCategories(t *testing.T) {
	router, _ := setupDocumentTest(t)
	token := testutil.DefaultTestToken()

	w := testutil.DoRequest(router, "GET", "/api/v1/document-categories", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data, ok := resp["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data to be array, got %T", resp["data"])
	}
	if len(data) < 1 {
		t.Error("Expected at least 1 category")
	}
}
