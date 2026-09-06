package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// RequestContext memberi batas waktu (timeout) 5 detik untuk setiap operasi basis data.
// Timeout ini mencegah query lambat atau deadlock menggantung selamanya.
// Pemanggil WAJIB memanggil cancel() melalui defer agar resources dibebaskan.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID membaca parameter :id dari URL dan memastikan nilainya berupa integer positif.
// Mengembalikan (id, true) jika valid, atau (0, false) jika bukan angka atau kurang dari 1.
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// allowedSort adalah daftar putih (whitelist) kolom yang diizinkan untuk pengurutan.
// Hal ini mencegah SQL injection melalui parameter sort karena nama kolom tidak
// bisa di-parameterize ($1, $2, ...) dalam prepared statement.
var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

// ParseListQuery membaca query string dari URL dan memberikan nilai bawaan yang aman.
// Fungsi ini menormalisasi dan memvalidasi seluruh parameter paginasi, pengurutan,
// pencarian, dan filter sebelum diteruskan ke lapisan repository.
func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	if raw := c.Query("grade_min"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMin = &v
		}
	}
	if raw := c.Query("grade_max"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMax = &v
		}
	}

	return q
}
