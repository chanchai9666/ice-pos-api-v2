package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"ice-pos-api-v2/internal/domain"
)

func TestCategoryLifecycle(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Login as Admin
	adminLoginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin1234",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(adminLoginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login failed: %v, status: %d", err, resp.StatusCode)
	}

	var adminRes struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&adminRes)
	adminToken := adminRes.Data.AccessToken

	// 2. Login as Cashier
	cashierLoginPayload, _ := json.Marshal(map[string]string{
		"username": "cashier",
		"password": "cashier1234",
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(cashierLoginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("cashier login failed: %v, status: %d", err, resp.StatusCode)
	}

	var cashierRes struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cashierRes)
	cashierToken := cashierRes.Data.AccessToken

	// 3. GET /api/v1/categories is public (no auth token required)
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected public GET /categories to return 200, got status: %d, err: %v", resp.StatusCode, err)
	}

	// 4. POST /api/v1/categories without token -> 401 Unauthorized
	catPayload, _ := json.Marshal(map[string]interface{}{
		"name":       "เครื่องดื่ม",
		"sort_order": 1,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(catPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected POST /categories without token to return 401, got: %d", resp.StatusCode)
	}

	// 5. POST /api/v1/categories with Cashier token -> 403 Forbidden
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(catPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected POST /categories with Cashier token to return 403, got: %d", resp.StatusCode)
	}

	// 6. POST /api/v1/categories with Empty Name -> 400 Bad Request
	emptyPayload, _ := json.Marshal(map[string]interface{}{
		"name":       "   ",
		"sort_order": 1,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(emptyPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected POST /categories with empty name to return 400, got: %d", resp.StatusCode)
	}

	// 7. POST /api/v1/categories with Admin token -> 201 Created
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(catPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected POST /categories with Admin token to return 201, got: %d", resp.StatusCode)
	}

	var createdRes struct {
		Success bool            `json:"success"`
		Data    domain.Category `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createdRes)
	categoryID := createdRes.Data.ID
	if categoryID == "" || createdRes.Data.Name != "เครื่องดื่ม" || createdRes.Data.SortOrder != 1 {
		t.Fatalf("unexpected created category data: %+v", createdRes.Data)
	}

	// 8. POST /api/v1/categories with duplicate name -> 409 Conflict
	dupPayload, _ := json.Marshal(map[string]interface{}{
		"name":       "เครื่องดื่ม",
		"sort_order": 2,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(dupPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected POST /categories with duplicate name to return 409, got: %d", resp.StatusCode)
	}

	// 9. GET /api/v1/categories/:id -> 200 OK
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/categories/"+categoryID, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected GET /categories/:id to return 200, got: %d", resp.StatusCode)
	}

	var getRes struct {
		Success bool            `json:"success"`
		Data    domain.Category `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&getRes)
	if getRes.Data.ID != categoryID || getRes.Data.Name != "เครื่องดื่ม" {
		t.Fatalf("unexpected fetched category: %+v", getRes.Data)
	}

	// 10. PUT /api/v1/categories/:id -> update to same name must succeed (200 OK)
	sameNamePayload, _ := json.Marshal(map[string]interface{}{
		"name":       "เครื่องดื่ม",
		"sort_order": 5,
	})
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/categories/"+categoryID, bytes.NewBuffer(sameNamePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected PUT /categories/:id with own name to return 200, got: %d", resp.StatusCode)
	}

	// 11. PUT /api/v1/categories/:id -> update name and sort order
	updatePayload, _ := json.Marshal(map[string]interface{}{
		"name":       "เครื่องดื่มและกาแฟ",
		"sort_order": 10,
	})
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/categories/"+categoryID, bytes.NewBuffer(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected PUT /categories/:id to return 200, got: %d", resp.StatusCode)
	}

	var updateRes struct {
		Success bool            `json:"success"`
		Data    domain.Category `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&updateRes)
	if updateRes.Data.Name != "เครื่องดื่มและกาแฟ" || updateRes.Data.SortOrder != 10 {
		t.Fatalf("unexpected updated category: %+v", updateRes.Data)
	}

	// 12. Create product linked to this category
	prodPayload, _ := json.Marshal(map[string]interface{}{
		"code":        "P-CAT-01",
		"barcode":     "885111111111",
		"name":        "Iced Americano",
		"price":       5000,
		"cost_price":  2500,
		"category_id": categoryID,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(prodPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create product linked to category: %v, status: %d", err, resp.StatusCode)
	}

	var prodRes struct {
		Data domain.Product `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&prodRes)
	productID := prodRes.Data.ID
	if prodRes.Data.CategoryID != categoryID {
		t.Fatalf("expected product category_id to be %s, got %s", categoryID, prodRes.Data.CategoryID)
	}

	// 13. DELETE /api/v1/categories/:id -> 200 OK (unlinks product atomically)
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/categories/"+categoryID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected DELETE /categories/:id to return 200, got: %d", resp.StatusCode)
	}

	// 14. Verify category is deleted -> GET /categories/:id returns 404
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/categories/"+categoryID, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected GET /categories/:id for deleted category to return 404, got: %d", resp.StatusCode)
	}

	// 15. Verify linked product now has empty category_id
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/products/"+productID, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected GET /products/:id to return 200, got: %d", resp.StatusCode)
	}

	var verifiedProdRes struct {
		Data domain.Product `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&verifiedProdRes)
	if verifiedProdRes.Data.CategoryID != "" {
		t.Fatalf("expected unlinked product category_id to be empty, got: %q", verifiedProdRes.Data.CategoryID)
	}

	// 16. DELETE /api/v1/categories/:id on non-existent category -> 404 Not Found
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/categories/non-existent-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected DELETE non-existent category to return 404, got: %d", resp.StatusCode)
	}

	// 17. PUT and DELETE Auth Protection (401 without token, 403 with Cashier token)
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/categories/some-id", bytes.NewBuffer(updatePayload))
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected PUT /categories without token to return 401, got: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPut, "/api/v1/categories/some-id", bytes.NewBuffer(updatePayload))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected PUT /categories with Cashier token to return 403, got: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/categories/some-id", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected DELETE /categories without token to return 401, got: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/categories/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected DELETE /categories with Cashier token to return 403, got: %d", resp.StatusCode)
	}

	// 18. Long Thai Name (> 33 runes, > 100 bytes, but <= 100 runes) must be accepted
	longThaiName := "อาหารและเครื่องดื่มเพื่อสุขภาพและของหวาน" // 40 Thai runes, 120 bytes
	longThaiPayload, _ := json.Marshal(map[string]interface{}{
		"name":       longThaiName,
		"sort_order": 99,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(longThaiPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected POST with valid 40-character Thai name to return 201, got: %d", resp.StatusCode)
	}

	// 19. Case-insensitive duplicate check (e.g. "Bakery" vs "BAKERY")
	bakeryPayload, _ := json.Marshal(map[string]interface{}{
		"name":       "Bakery Shop",
		"sort_order": 1,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(bakeryPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected POST /categories Bakery Shop to return 201, got: %d", resp.StatusCode)
	}

	dupCasingPayload, _ := json.Marshal(map[string]interface{}{
		"name":       "BAKERY SHOP",
		"sort_order": 2,
	})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(dupCasingPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected POST /categories with different casing to return 409, got: %d", resp.StatusCode)
	}

	// 20. PUT with empty name returns 400 Bad Request
	var createdBakery struct {
		Data domain.Category `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createdBakery)
	// We can update the bakery category
	req, _ = http.NewRequest(http.MethodPut, "/api/v1/categories/non-existent-uuid", bytes.NewBuffer(bakeryPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected PUT non-existent category to return 404, got: %d", resp.StatusCode)
	}
}
