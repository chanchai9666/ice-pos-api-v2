package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"ice-pos-api-v2/internal/domain"
)

func TestBarcodeScanAndRegistration(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Login as Admin to get auth token
	loginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin1234",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %v, status: %d", err, resp.StatusCode)
	}

	var loginRes struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&loginRes)
	adminToken := loginRes.Data.AccessToken

	// 2. Create Category
	// Category is preloaded or can be created
	// 3. Create Product A with Barcode "885000000001"
	prodA := map[string]interface{}{
		"code":       "P-SCAN-01",
		"barcode":    "885999000001",
		"name":       "Green Tea 500ml",
		"price":      2500, // 25.00 Baht
		"cost_price": 1800,
	}
	bodyA, _ := json.Marshal(prodA)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(bodyA))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create product A: %v, status: %d", err, resp.StatusCode)
	}

	var createResA struct {
		Data domain.Product `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createResA)
	prodAID := createResA.Data.ID

	// 4. Test Exact Barcode Scan (สแกนเพื่อตรวจสอบข้อมูล / ดึงลงตะกร้า)
	t.Run("Scan exact barcode succeeds", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/barcode/885999000001", nil)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
		}

		var res struct {
			Success bool           `json:"success"`
			Data    domain.Product `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		if !res.Success || res.Data.Barcode != "885999000001" || res.Data.Name != "Green Tea 500ml" {
			t.Errorf("unexpected product data: %+v", res.Data)
		}
	})

	// 5. Test Barcode Scan with spaces / URL encoding (กล้องอ่านติด whitespace)
	t.Run("Scan barcode with surrounding whitespace succeeds", func(t *testing.T) {
		// fiber handles url decoding: %20885999000001%20
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/barcode/%20885999000001%20", nil)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK after trimming whitespace, got: %d", resp.StatusCode)
		}

		var res struct {
			Success bool           `json:"success"`
			Data    domain.Product `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		if res.Data.ID != prodAID {
			t.Errorf("expected product ID %s, got %s", prodAID, res.Data.ID)
		}
	})

	// 6. Test Barcode Scan Not Found
	t.Run("Scan non-existent barcode returns 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/products/barcode/999999999999", nil)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got: %d", resp.StatusCode)
		}
	})

	// 7. Test Query Parameter Barcode Search: GET /api/v1/products?barcode=...
	t.Run("List products with exact barcode query parameter", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/products?barcode=885999000001", nil)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got: %d", resp.StatusCode)
		}

		var res struct {
			Success bool             `json:"success"`
			Data    []domain.Product `json:"data"`
			Total   int64            `json:"total"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		if res.Total != 1 || len(res.Data) != 1 || res.Data[0].ID != prodAID {
			t.Errorf("expected 1 product with matching barcode, got total=%d, len=%d", res.Total, len(res.Data))
		}
	})

	// 8. Test Duplicate Barcode on CreateProduct (สแกนลงทะเบียนซ้ำ)
	t.Run("Create product with duplicate barcode returns 400 Bad Request", func(t *testing.T) {
		prodDup := map[string]interface{}{
			"code":    "P-SCAN-02",
			"barcode": "885999000001", // Duplicate!
			"name":    "Another Green Tea",
			"price":   3000,
		}
		bodyDup, _ := json.Marshal(prodDup)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(bodyDup))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for duplicate barcode, got: %d", resp.StatusCode)
		}
	})

	// 9. Create Product B without barcode, then register barcode via PUT /api/v1/products/:id
	t.Run("Update product to assign barcode via PUT /products/:id", func(t *testing.T) {
		prodB := map[string]interface{}{
			"code":    "P-SCAN-03",
			"barcode": "",
			"name":    "Mineral Water 600ml",
			"price":   1000,
		}
		bodyB, _ := json.Marshal(prodB)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewBuffer(bodyB))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req)
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to create product B: %v, status: %d", err, resp.StatusCode)
		}

		var createResB struct {
			Data domain.Product `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&createResB)
		prodBID := createResB.Data.ID

		// Update product B to assign barcode "885999000002"
		updatePayload, _ := json.Marshal(map[string]interface{}{
			"name":    "Mineral Water 600ml Premium",
			"barcode": "885999000002",
			"price":   1200,
		})
		req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/products/%s", prodBID), bytes.NewBuffer(updatePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err = app.Test(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from PUT /products/:id, got: %d", resp.StatusCode)
		}

		// Now scan with new barcode "885999000002"
		req, _ = http.NewRequest(http.MethodGet, "/api/v1/products/barcode/885999000002", nil)
		resp, err = app.Test(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK after assigning barcode, got: %d", resp.StatusCode)
		}

		// Verify duplicate check on update: try updating product B with product A's barcode
		dupUpdatePayload, _ := json.Marshal(map[string]interface{}{
			"barcode": "885999000001",
		})
		req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/products/%s", prodBID), bytes.NewBuffer(dupUpdatePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err = app.Test(req)
		if err != nil || resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request when updating with existing barcode, got: %d", resp.StatusCode)
		}
	})
}
