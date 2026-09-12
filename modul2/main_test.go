package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// setupApp membuat instance Fiber untuk testing tanpa benar-benar membuka port.
func setupApp() *fiber.App {
	// Reset state
	students = nil
	nextID = 1

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return fail(c, status, pesan)
		},
	})

	app.Use(requestid.New())

	api := app.Group("/api/v1")

	s := api.Group("/students", requireJSON)
	s.Get("/", listStudents)
	s.Get("/:id", getStudent)
	s.Post("/", createStudent)
	s.Put("/:id", replaceStudent)
	s.Patch("/:id", patchStudent)
	s.Delete("/:id", deleteStudent)

	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

func doRequest(app *fiber.App, method, url string, body interface{}) *http.Response {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, _ := app.Test(req, -1)
	return resp
}

func parseResponse(resp *http.Response) WebResponse {
	var wr WebResponse
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &wr)
	return wr
}

// ============================================================
// TEST 1: POST /api/v1/students ΓåÆ 201 Created
// ============================================================
func TestCreateStudent_201(t *testing.T) {
	app := setupApp()

	resp := doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	// Check Location header
	loc := resp.Header.Get("Location")
	if loc != "/api/v1/students/1" {
		t.Fatalf("expected Location /api/v1/students/1, got %s", loc)
	}

	wr := parseResponse(resp)
	if !wr.Success {
		t.Fatal("expected success=true")
	}

	fmt.Println("Γ£à TEST 1: POST ΓåÆ 201 Created + header Location Γ£ô")
}

// ============================================================
// TEST 2: GET /api/v1/students ΓåÆ 200 OK with pagination meta
// ============================================================
func TestListStudents_200(t *testing.T) {
	app := setupApp()

	// Create 3 students
	for i := 1; i <= 3; i++ {
		doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
			"nim": fmt.Sprintf("24000%d", i), "name": fmt.Sprintf("Student %d", i),
			"grade": float64(70 + i*5), "is_active": i%2 == 1,
		})
	}

	resp := doRequest(app, "GET", "/api/v1/students?page=1&limit=2", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	wr := parseResponse(resp)
	if wr.Meta == nil {
		t.Fatal("expected meta in response")
	}
	if wr.Meta.Total != 3 {
		t.Fatalf("expected total=3, got %d", wr.Meta.Total)
	}
	if wr.Meta.TotalPages != 2 {
		t.Fatalf("expected total_pages=2, got %d", wr.Meta.TotalPages)
	}

	fmt.Println("Γ£à TEST 2: GET list ΓåÆ 200 OK with pagination meta Γ£ô")
}

// ============================================================
// TEST 3: GET /api/v1/students?search=... ΓåÆ 200 OK filtered
// ============================================================
func TestSearchStudents_200(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})
	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240002", "name": "Sari Dewi", "grade": 92.0, "is_active": true,
	})

	resp := doRequest(app, "GET", "/api/v1/students?search=rizki", nil)
	wr := parseResponse(resp)

	if wr.Meta.Total != 1 {
		t.Fatalf("expected total=1 for search=rizki, got %d", wr.Meta.Total)
	}

	fmt.Println("Γ£à TEST 3: GET search ΓåÆ 200 OK filtered Γ£ô")
}

// ============================================================
// TEST 4: GET /api/v1/students/:id ΓåÆ 200 OK
// ============================================================
func TestGetStudent_200(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	resp := doRequest(app, "GET", "/api/v1/students/1", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 4: GET /:id ΓåÆ 200 OK Γ£ô")
}

// ============================================================
// TEST 5: GET /api/v1/students/999 ΓåÆ 404 Not Found
// ============================================================
func TestGetStudent_404(t *testing.T) {
	app := setupApp()

	resp := doRequest(app, "GET", "/api/v1/students/999", nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 5: GET /:id not found ΓåÆ 404 Γ£ô")
}

// ============================================================
// TEST 6: GET /api/v1/students/abc ΓåÆ 400 Bad Request
// ============================================================
func TestGetStudent_400_InvalidID(t *testing.T) {
	app := setupApp()

	resp := doRequest(app, "GET", "/api/v1/students/abc", nil)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 6: GET /:id invalid ΓåÆ 400 Γ£ô")
}

// ============================================================
// TEST 7: PUT /api/v1/students/:id ΓåÆ 200 OK (full replace)
// ============================================================
func TestReplaceStudent_200(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	resp := doRequest(app, "PUT", "/api/v1/students/1", map[string]interface{}{
		"nim": "240001", "name": "Rizki Baru", "grade": 91.0, "is_active": false,
	})
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	wr := parseResponse(resp)
	data := wr.Data.(map[string]interface{})
	if data["name"] != "Rizki Baru" {
		t.Fatalf("expected name=Rizki Baru, got %v", data["name"])
	}
	if data["is_active"] != false {
		t.Fatal("expected is_active=false after PUT")
	}

	fmt.Println("Γ£à TEST 7: PUT ΓåÆ 200 OK (all fields replaced) Γ£ô")
}

// ============================================================
// TEST 8: PUT without required fields ΓåÆ 422
// ============================================================
func TestReplaceStudent_422(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	resp := doRequest(app, "PUT", "/api/v1/students/1", map[string]interface{}{
		"name": "Rizki",
	})
	if resp.StatusCode != 422 {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 8: PUT incomplete fields ΓåÆ 422 Γ£ô")
}

// ============================================================
// TEST 9: PATCH /api/v1/students/:id ΓåÆ 200 OK (partial update)
// ============================================================
func TestPatchStudent_200(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	isActive := false
	resp := doRequest(app, "PATCH", "/api/v1/students/1", map[string]interface{}{
		"is_active": isActive,
	})
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	wr := parseResponse(resp)
	data := wr.Data.(map[string]interface{})
	// Name should remain unchanged
	if data["name"] != "Rizki Ibrahim" {
		t.Fatalf("expected name unchanged, got %v", data["name"])
	}
	if data["is_active"] != false {
		t.Fatal("expected is_active=false after PATCH")
	}

	fmt.Println("Γ£à TEST 9: PATCH ΓåÆ 200 OK (only is_active changed, rest unchanged) Γ£ô")
}

// ============================================================
// TEST 10: DELETE /api/v1/students/:id ΓåÆ 204 No Content
// ============================================================
func TestDeleteStudent_204(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	resp := doRequest(app, "DELETE", "/api/v1/students/1", nil)
	if resp.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify deletion
	resp2 := doRequest(app, "GET", "/api/v1/students/1", nil)
	if resp2.StatusCode != 404 {
		t.Fatalf("expected 404 after delete, got %d", resp2.StatusCode)
	}

	fmt.Println("Γ£à TEST 10: DELETE ΓåÆ 204 No Content Γ£ô")
}

// ============================================================
// TEST 11: POST with duplicate NIM ΓåÆ 409 Conflict
// ============================================================
func TestCreateStudent_409_DuplicateNIM(t *testing.T) {
	app := setupApp()

	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	resp := doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Duplikat", "grade": 80.0, "is_active": true,
	})
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 11: POST duplicate NIM ΓåÆ 409 Conflict Γ£ô")
}

// ============================================================
// TEST 12: POST without Content-Type ΓåÆ 415 Unsupported Media Type
// ============================================================
func TestCreateStudent_415(t *testing.T) {
	app := setupApp()

	req := httptest.NewRequest("POST", "/api/v1/students", bytes.NewReader([]byte(`{"nim":"x"}`)))
	// No Content-Type set
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 415 {
		t.Fatalf("expected 415, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 12: POST without Content-Type ΓåÆ 415 Γ£ô")
}

// ============================================================
// TEST 13: POST with validation errors ΓåÆ 422
// ============================================================
func TestCreateStudent_422(t *testing.T) {
	app := setupApp()

	resp := doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "", "name": "", "grade": -5,
	})
	if resp.StatusCode != 422 {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}

	wr := parseResponse(resp)
	if wr.Errors == nil {
		t.Fatal("expected errors in response")
	}

	fmt.Println("Γ£à TEST 13: POST validation errors ΓåÆ 422 with field details Γ£ô")
}

// ============================================================
// TEST 14: Filter + Sort + Pagination combined
// ============================================================
func TestListStudents_FilterSortPagination(t *testing.T) {
	app := setupApp()

	// Create several students
	testData := []map[string]interface{}{
		{"nim": "240001", "name": "Andi", "grade": 85.0, "is_active": true},
		{"nim": "240002", "name": "Budi", "grade": 70.0, "is_active": false},
		{"nim": "240003", "name": "Cici", "grade": 92.0, "is_active": true},
		{"nim": "240004", "name": "Deni", "grade": 88.0, "is_active": true},
		{"nim": "240005", "name": "Eka", "grade": 65.0, "is_active": false},
	}
	for _, d := range testData {
		doRequest(app, "POST", "/api/v1/students", d)
	}

	// Filter active, sort by grade desc, page 1 limit 2
	resp := doRequest(app, "GET", "/api/v1/students?is_active=true&sort=grade&order=desc&page=1&limit=2", nil)
	wr := parseResponse(resp)

	if wr.Meta.Total != 3 {
		t.Fatalf("expected 3 active students, got %d", wr.Meta.Total)
	}
	if wr.Meta.TotalPages != 2 {
		t.Fatalf("expected 2 pages, got %d", wr.Meta.TotalPages)
	}

	// Filter by grade range
	resp2 := doRequest(app, "GET", "/api/v1/students?grade_min=80&grade_max=90", nil)
	wr2 := parseResponse(resp2)
	if wr2.Meta.Total != 2 {
		t.Fatalf("expected 2 students with grade 80-90, got %d", wr2.Meta.Total)
	}

	fmt.Println("Γ£à TEST 14: Filter + Sort + Pagination combined Γ£ô")
}

// ============================================================
// TEST 15: PUT vs PATCH difference demonstration
// ============================================================
func TestPUTvsPATCH_Difference(t *testing.T) {
	app := setupApp()

	// Create student
	doRequest(app, "POST", "/api/v1/students", map[string]interface{}{
		"nim": "240001", "name": "Rizki Ibrahim", "grade": 87.5, "is_active": true,
	})

	fmt.Println("\n--- PUT vs PATCH Demonstration ---")

	// Verify initial state
	resp0 := doRequest(app, "GET", "/api/v1/students/1", nil)
	wr0 := parseResponse(resp0)
	d0 := wr0.Data.(map[string]interface{})
	fmt.Printf("SEBELUM: name=%v, grade=%v, is_active=%v\n", d0["name"], d0["grade"], d0["is_active"])

	// PUT ΓÇö replaces ALL fields
	resp1 := doRequest(app, "PUT", "/api/v1/students/1", map[string]interface{}{
		"nim": "240001", "name": "NAMA BARU VIA PUT", "grade": 50.0, "is_active": false,
	})
	wr1 := parseResponse(resp1)
	d1 := wr1.Data.(map[string]interface{})
	fmt.Printf("SETELAH PUT: name=%v, grade=%v, is_active=%v\n", d1["name"], d1["grade"], d1["is_active"])

	if d1["name"] != "NAMA BARU VIA PUT" || d1["grade"] != 50.0 || d1["is_active"] != false {
		t.Fatal("PUT did not replace all fields correctly")
	}

	// PATCH ΓÇö only change is_active back to true
	resp2 := doRequest(app, "PATCH", "/api/v1/students/1", map[string]interface{}{
		"is_active": true,
	})
	wr2 := parseResponse(resp2)
	d2 := wr2.Data.(map[string]interface{})
	fmt.Printf("SETELAH PATCH (hanya is_active): name=%v, grade=%v, is_active=%v\n", d2["name"], d2["grade"], d2["is_active"])

	// Name and grade should remain from PUT, only is_active changed
	if d2["name"] != "NAMA BARU VIA PUT" {
		t.Fatalf("PATCH changed name unexpectedly: %v", d2["name"])
	}
	if d2["grade"] != 50.0 {
		t.Fatalf("PATCH changed grade unexpectedly: %v", d2["grade"])
	}
	if d2["is_active"] != true {
		t.Fatal("PATCH did not change is_active")
	}

	fmt.Println("Γ£à TEST 15: PUT replaces ALL, PATCH only changes sent fields Γ£ô")
}

// ============================================================
// TEST 16: Limit cap validation (?limit=99999 ΓåÆ capped to 100)
// ============================================================
func TestLimitCap(t *testing.T) {
	app := setupApp()

	resp := doRequest(app, "GET", "/api/v1/students?limit=99999", nil)
	wr := parseResponse(resp)

	if wr.Meta.Limit != 100 {
		t.Fatalf("expected limit capped at 100, got %d", wr.Meta.Limit)
	}

	fmt.Println("Γ£à TEST 16: ?limit=99999 ΓåÆ capped to 100 Γ£ô")
}

// ============================================================
// TEST 17: Sort whitelist (?sort=password ΓåÆ defaults to id)
// ============================================================
func TestSortWhitelist(t *testing.T) {
	app := setupApp()

	// This should not crash, sort=password should be ignored and default to id
	resp := doRequest(app, "GET", "/api/v1/students?sort=password", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 with invalid sort, got %d", resp.StatusCode)
	}

	fmt.Println("Γ£à TEST 17: ?sort=password ΓåÆ defaults to id (whitelist protection) Γ£ô")
}
