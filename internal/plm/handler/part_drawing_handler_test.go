package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/entity"
	"github.com/bitfantasy/nimo/internal/plm/repository"
	"github.com/bitfantasy/nimo/internal/plm/testutil"
	"gorm.io/gorm"
)

func setupPartDrawingTest(t *testing.T) (*testutil.TestEnv, *PartDrawingHandler) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	router := testutil.SetupRouter()

	repos := repository.NewRepositories(db)
	handler := NewPartDrawingHandler(repos.PartDrawing, repos.ProjectBOM)

	api := testutil.AuthGroup(router, "/api/v1")
	api.GET("/projects/:id/bom-items/:itemId/drawings", handler.ListDrawings)
	api.POST("/projects/:id/bom-items/:itemId/drawings", handler.UploadDrawing)
	api.DELETE("/projects/:id/bom-items/:itemId/drawings/:drawingId", handler.DeleteDrawing)
	api.GET("/projects/:id/bom-items/:itemId/drawings/:drawingId/download", handler.DownloadDrawing)
	api.GET("/projects/:id/boms/:bomId/drawings", handler.ListDrawingsByBOM)

	return &testutil.TestEnv{DB: db, Router: router, T: t}, handler
}

func seedBOMAndItem(t *testing.T, db *gorm.DB, userID string) (*entity.ProjectBOM, *entity.ProjectBOMItem) {
	t.Helper()
	// Seed a project first
	project := &entity.Project{
		ID: "proj-draw-001", Code: "PRJ-D001", Name: "Drawing Test Project",
		Status: "active", Phase: "evt", ManagerID: userID, CreatedBy: userID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	db.Create(project)

	bom := &entity.ProjectBOM{
		ID: "bom-draw-001", ProjectID: "proj-draw-001", BOMType: "PBOM",
		Name: "Test PBOM", Version: "v1.0", Status: "draft", CreatedBy: userID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	db.Create(bom)

	item := &entity.ProjectBOMItem{
		ID: "item-draw-001", BOMID: "bom-draw-001", Name: "Test Part",
		Quantity: 1, Unit: "pcs", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	db.Create(item)

	return bom, item
}

func TestPartDrawingUploadAndList(t *testing.T) {
	env, _ := setupPartDrawingTest(t)
	token := testutil.DefaultTestToken()
	testutil.SeedTestUser(t, env.DB, "test-user-001", "Test Admin", "admin@test.com")
	_, item := seedBOMAndItem(t, env.DB, "test-user-001")

	// Upload 2D drawing
	w := testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings",
		map[string]interface{}{
			"drawing_type":       "2D",
			"file_id":            "file-001",
			"file_name":          "part-2d-v1.dwg",
			"file_size":          12345,
			"change_description": "初版图纸",
		}, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["version"] != "v1" {
		t.Errorf("Expected version v1, got %v", data["version"])
	}
	drawingID := data["id"].(string)

	// Upload another 2D version
	w2 := testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings",
		map[string]interface{}{
			"drawing_type":       "2D",
			"file_id":            "file-002",
			"file_name":          "part-2d-v2.dwg",
			"change_description": "修正尺寸",
		}, token)
	if w2.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w2.Code, w2.Body.String())
	}
	resp2 := testutil.ParseResponse(w2)
	data2 := resp2["data"].(map[string]interface{})
	if data2["version"] != "v2" {
		t.Errorf("Expected version v2, got %v", data2["version"])
	}

	// Upload 3D drawing
	testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings",
		map[string]interface{}{
			"drawing_type": "3D",
			"file_id":      "file-003",
			"file_name":    "part-3d.step",
		}, token)

	// List drawings for item
	w3 := testutil.DoRequest(env.Router, "GET",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings", nil, token)
	if w3.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w3.Code, w3.Body.String())
	}
	resp3 := testutil.ParseResponse(w3)
	data3 := resp3["data"].(map[string]interface{})
	drawings2D := data3["2D"].([]interface{})
	drawings3D := data3["3D"].([]interface{})
	if len(drawings2D) != 2 {
		t.Errorf("Expected 2 2D drawings, got %d", len(drawings2D))
	}
	if len(drawings3D) != 1 {
		t.Errorf("Expected 1 3D drawing, got %d", len(drawings3D))
	}

	// Delete first drawing
	w4 := testutil.DoRequest(env.Router, "DELETE",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings/"+drawingID, nil, token)
	if w4.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w4.Code, w4.Body.String())
	}

	// Verify deletion
	w5 := testutil.DoRequest(env.Router, "GET",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings", nil, token)
	resp5 := testutil.ParseResponse(w5)
	data5 := resp5["data"].(map[string]interface{})
	if len(data5["2D"].([]interface{})) != 1 {
		t.Errorf("Expected 1 2D drawing after delete, got %d", len(data5["2D"].([]interface{})))
	}
}

func TestPartDrawingListByBOM(t *testing.T) {
	env, _ := setupPartDrawingTest(t)
	token := testutil.DefaultTestToken()
	testutil.SeedTestUser(t, env.DB, "test-user-001", "Test Admin", "admin@test.com")
	bom, item := seedBOMAndItem(t, env.DB, "test-user-001")

	// Upload a drawing
	testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-draw-001/bom-items/"+item.ID+"/drawings",
		map[string]interface{}{
			"drawing_type": "2D",
			"file_id":      "file-bom-001",
			"file_name":    "bom-part.dwg",
		}, token)

	// List by BOM
	w := testutil.DoRequest(env.Router, "GET",
		"/api/v1/projects/proj-draw-001/boms/"+bom.ID+"/drawings", nil, token)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	itemDrawings := data[item.ID].(map[string]interface{})
	if len(itemDrawings["2D"].([]interface{})) != 1 {
		t.Errorf("Expected 1 2D drawing for item, got %d", len(itemDrawings["2D"].([]interface{})))
	}
}

func TestPartDrawingBadRequest(t *testing.T) {
	env, _ := setupPartDrawingTest(t)
	token := testutil.DefaultTestToken()
	testutil.SeedTestUser(t, env.DB, "test-user-001", "Test Admin", "admin@test.com")

	// Missing required fields
	w := testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-001/bom-items/item-001/drawings",
		map[string]interface{}{
			"drawing_type": "2D",
		}, token)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing fields, got %d", w.Code)
	}

	// Invalid drawing type
	w2 := testutil.DoRequest(env.Router, "POST",
		"/api/v1/projects/proj-001/bom-items/item-001/drawings",
		map[string]interface{}{
			"drawing_type": "4D",
			"file_id":      "f1",
			"file_name":    "test.dwg",
		}, token)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid type, got %d", w2.Code)
	}
}

func TestPartDrawingDeleteNotFound(t *testing.T) {
	env, _ := setupPartDrawingTest(t)
	token := testutil.DefaultTestToken()
	testutil.SeedTestUser(t, env.DB, "test-user-001", "Test Admin", "admin@test.com")

	w := testutil.DoRequest(env.Router, "DELETE",
		"/api/v1/projects/proj-001/bom-items/item-001/drawings/nonexistent", nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestPartDrawingDownloadNotFound(t *testing.T) {
	env, _ := setupPartDrawingTest(t)
	token := testutil.DefaultTestToken()
	testutil.SeedTestUser(t, env.DB, "test-user-001", "Test Admin", "admin@test.com")

	w := testutil.DoRequest(env.Router, "GET",
		"/api/v1/projects/proj-001/bom-items/item-001/drawings/nonexistent/download", nil, token)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}
