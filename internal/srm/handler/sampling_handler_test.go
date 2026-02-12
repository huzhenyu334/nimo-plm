package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/bitfantasy/nimo/internal/plm/testutil"
	"github.com/bitfantasy/nimo/internal/srm/entity"
	"github.com/bitfantasy/nimo/internal/srm/repository"
	"github.com/bitfantasy/nimo/internal/srm/service"
)

func setupSamplingTest(t *testing.T) (*testutil.TestEnv, *SamplingHandler) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	if err := db.AutoMigrate(
		&entity.Supplier{},
		&entity.PurchaseRequest{},
		&entity.PRItem{},
		&entity.SamplingRequest{},
		&entity.ActivityLog{},
	); err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}

	samplingRepo := repository.NewSamplingRepository(db)
	prRepo := repository.NewPRRepository(db)
	supplierRepo := repository.NewSupplierRepository(db)
	activityLogRepo := repository.NewActivityLogRepository(db)

	svc := service.NewSamplingService(samplingRepo, prRepo, supplierRepo, activityLogRepo, db)
	handler := NewSamplingHandler(svc)

	router := testutil.SetupRouter()
	api := testutil.AuthGroup(router, "/api/v1/srm")
	api.POST("/pr-items/:itemId/sampling", handler.CreateSamplingRequest)
	api.GET("/pr-items/:itemId/sampling", handler.ListSamplingRequests)
	api.PUT("/sampling/:id/status", handler.UpdateSamplingStatus)

	return &testutil.TestEnv{DB: db, Router: router, T: t}, handler
}

func seedSamplingTestData(t *testing.T, env *testutil.TestEnv) (prItemID, supplierID string) {
	t.Helper()

	// Create supplier
	supplier := &entity.Supplier{
		ID:        "sup-sample-001",
		Code:      "SUP-S001",
		Name:      "打样供应商A",
		Category:  "structural",
		Status:    "active",
		CreatedBy: "test-user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := env.DB.Create(supplier).Error; err != nil {
		t.Fatalf("Failed to seed supplier: %v", err)
	}

	// Create PR
	pr := &entity.PurchaseRequest{
		ID:          "pr-sample-001",
		PRCode:      "PR-2026-TEST",
		Title:       "打样测试PR",
		Type:        "sample",
		Status:      "approved",
		RequestedBy: "test-user",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := env.DB.Create(pr).Error; err != nil {
		t.Fatalf("Failed to seed PR: %v", err)
	}

	// Create PR item in pending status
	item := &entity.PRItem{
		ID:           "item-sample-001",
		PRID:         "pr-sample-001",
		MaterialName: "测试外壳",
		MaterialCode: "MAT-001",
		Specification: "ABS+PC, 100x50x20mm",
		Category:     "结构件",
		Quantity:     100,
		Unit:         "pcs",
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := env.DB.Create(item).Error; err != nil {
		t.Fatalf("Failed to seed PR item: %v", err)
	}

	return item.ID, supplier.ID
}

// TestSamplingCreateAndList tests creating a sampling request and listing sampling records
func TestSamplingCreateAndList(t *testing.T) {
	env, _ := setupSamplingTest(t)
	token := testutil.DefaultTestToken()

	itemID, supplierID := seedSamplingTestData(t, env)

	// Create sampling request
	body := map[string]interface{}{
		"supplier_id": supplierID,
		"sample_qty":  5,
		"notes":       "首次打样",
	}
	w := testutil.DoRequest(env.Router, http.MethodPost, "/api/v1/srm/pr-items/"+itemID+"/sampling", body, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	data := resp["data"].(map[string]interface{})
	if data["round"].(float64) != 1 {
		t.Fatalf("expected round 1, got %v", data["round"])
	}
	if data["status"] != "preparing" {
		t.Fatalf("expected status preparing, got %v", data["status"])
	}
	samplingID := data["id"].(string)

	// Verify PR item status changed to sampling
	var item entity.PRItem
	env.DB.Where("id = ?", itemID).First(&item)
	if item.Status != "sampling" {
		t.Fatalf("expected PR item status 'sampling', got '%s'", item.Status)
	}

	// List sampling records
	w2 := testutil.DoRequest(env.Router, http.MethodGet, "/api/v1/srm/pr-items/"+itemID+"/sampling", nil, token)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	resp2 := testutil.ParseResponse(w2)
	data2 := resp2["data"].(map[string]interface{})
	items := data2["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 sampling record, got %d", len(items))
	}

	// Update sampling status: preparing → shipping
	w3 := testutil.DoRequest(env.Router, http.MethodPut, "/api/v1/srm/sampling/"+samplingID+"/status",
		map[string]interface{}{"status": "shipping"}, token)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for status update, got %d: %s", w3.Code, w3.Body.String())
	}

	resp3 := testutil.ParseResponse(w3)
	data3 := resp3["data"].(map[string]interface{})
	if data3["status"] != "shipping" {
		t.Fatalf("expected status 'shipping', got %v", data3["status"])
	}

	// Update: shipping → arrived
	w4 := testutil.DoRequest(env.Router, http.MethodPut, "/api/v1/srm/sampling/"+samplingID+"/status",
		map[string]interface{}{"status": "arrived"}, token)
	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w4.Code, w4.Body.String())
	}

	resp4 := testutil.ParseResponse(w4)
	data4 := resp4["data"].(map[string]interface{})
	if data4["status"] != "arrived" {
		t.Fatalf("expected status 'arrived', got %v", data4["status"])
	}
	if data4["arrived_at"] == nil {
		t.Fatal("expected arrived_at to be set")
	}
}

// TestSamplingInvalidStatus tests that invalid status transitions are rejected
func TestSamplingInvalidStatus(t *testing.T) {
	env, _ := setupSamplingTest(t)
	token := testutil.DefaultTestToken()

	itemID, supplierID := seedSamplingTestData(t, env)

	// Create sampling
	body := map[string]interface{}{
		"supplier_id": supplierID,
		"sample_qty":  3,
	}
	w := testutil.DoRequest(env.Router, http.MethodPost, "/api/v1/srm/pr-items/"+itemID+"/sampling", body, token)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := testutil.ParseResponse(w)
	samplingID := resp["data"].(map[string]interface{})["id"].(string)

	// Try invalid transition: preparing → arrived (should fail, must go through shipping)
	w2 := testutil.DoRequest(env.Router, http.MethodPut, "/api/v1/srm/sampling/"+samplingID+"/status",
		map[string]interface{}{"status": "arrived"}, token)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid transition, got %d: %s", w2.Code, w2.Body.String())
	}
}

// TestSamplingMultipleRounds tests that multiple sampling rounds increment correctly
func TestSamplingMultipleRounds(t *testing.T) {
	env, _ := setupSamplingTest(t)
	token := testutil.DefaultTestToken()

	itemID, supplierID := seedSamplingTestData(t, env)

	// Create first round
	body := map[string]interface{}{
		"supplier_id": supplierID,
		"sample_qty":  5,
	}
	w1 := testutil.DoRequest(env.Router, http.MethodPost, "/api/v1/srm/pr-items/"+itemID+"/sampling", body, token)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w1.Code, w1.Body.String())
	}
	resp1 := testutil.ParseResponse(w1)
	round1 := resp1["data"].(map[string]interface{})["round"].(float64)
	if round1 != 1 {
		t.Fatalf("expected round 1, got %v", round1)
	}

	// Create second round (item is now in sampling status, which allows re-sampling)
	w2 := testutil.DoRequest(env.Router, http.MethodPost, "/api/v1/srm/pr-items/"+itemID+"/sampling", body, token)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for second round, got %d: %s", w2.Code, w2.Body.String())
	}
	resp2 := testutil.ParseResponse(w2)
	round2 := resp2["data"].(map[string]interface{})["round"].(float64)
	if round2 != 2 {
		t.Fatalf("expected round 2, got %v", round2)
	}

	// List should show 2 records
	w3 := testutil.DoRequest(env.Router, http.MethodGet, "/api/v1/srm/pr-items/"+itemID+"/sampling", nil, token)
	resp3 := testutil.ParseResponse(w3)
	items := resp3["data"].(map[string]interface{})["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected 2 sampling records, got %d", len(items))
	}
}

// TestPRItemStatusTransitions tests the new sampling/quoting status transitions
func TestPRItemStatusTransitions(t *testing.T) {
	// Test that ValidPRItemTransitions includes the new states
	transitions := entity.ValidPRItemTransitions

	// pending → sampling
	pendingTargets := transitions["pending"]
	hasSampling := false
	for _, s := range pendingTargets {
		if s == "sampling" {
			hasSampling = true
		}
	}
	if !hasSampling {
		t.Fatal("expected pending → sampling transition to be valid")
	}

	// sampling → quoting
	samplingTargets := transitions["sampling"]
	hasQuoting := false
	for _, s := range samplingTargets {
		if s == "quoting" {
			hasQuoting = true
		}
	}
	if !hasQuoting {
		t.Fatal("expected sampling → quoting transition to be valid")
	}

	// quoting → sourcing
	quotingTargets := transitions["quoting"]
	hasSourcing := false
	for _, s := range quotingTargets {
		if s == "sourcing" {
			hasSourcing = true
		}
	}
	if !hasSourcing {
		t.Fatal("expected quoting → sourcing transition to be valid")
	}
}
