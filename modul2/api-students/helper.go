package main

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// === Helper Respons ===
// Setiap fungsi di bawah membungkus respons dalam amplop WebResponse
// dan mengembalikan status HTTP yang tepat sesuai situasi.

// ok mengembalikan 200 OK ΓÇö permintaan berhasil dan ada isi yang dikirim.
func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

// okList mengembalikan 200 OK dengan metadata paginasi.
func okList(c *fiber.Ctx, message string, data any, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

// created mengembalikan 201 Created ΓÇö sumber daya baru berhasil dibuat.
// Header Location dikirimkan agar pengguna tahu di mana sumber daya baru berada.
func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

// noContent mengembalikan 204 No Content ΓÇö berhasil, tanpa body.
// Umumnya dipakai untuk DELETE.
func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// fail mengembalikan respons gagal dengan status dan pesan tertentu.
func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(WebResponse{Success: false, Message: message})
}

// failValidation mengembalikan 422 Unprocessable Entity ΓÇö bentuknya benar,
// tetapi isinya tidak lolos validasi. Field yang bermasalah dirinci satu per satu
// agar pengguna dapat menandai kolom yang salah pada formulir.
func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

// failConflict mengembalikan 409 Conflict ΓÇö bertentangan dengan data yang ada.
// Dipakai misalnya ketika NIM yang dikirim sudah terdaftar.
func failConflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(WebResponse{
		Success: false, Message: message,
	})
}

// === Daftar Putih dan Parser Query String ===

// allowedSort adalah daftar putih field yang boleh dipakai untuk mengurutkan.
// Menerima nama field apa pun dari pengguna lalu menempelkannya ke query basis data
// adalah jalan mudah menuju SQL injection ΓÇö meskipun saat ini data masih di memori,
// kebiasaan baiknya dibentuk sekarang.
var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

// parseListQuery membaca query string dan memberi nilai bawaan yang aman.
// Setiap masukan dari pengguna divalidasi: page minimal 1, limit dibatasi 1ΓÇô100,
// sort harus ada di daftar putih, dan order hanya boleh asc atau desc.
func parseListQuery(c *fiber.Ctx) ListQuery {
	q := ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}

	// Validasi page ΓÇö minimal 1
	if q.Page < 1 {
		q.Page = 1
	}

	// Validasi limit ΓÇö minimal 1, maksimal 100.
	// Batas atas 100 mencegah pengguna mengirim ?limit=99999999 yang bisa
	// membuat server kehabisan memori saat data sudah banyak.
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	// Validasi sort ΓÇö hanya field yang ada di daftar putih yang diterima
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}

	// Validasi order ΓÇö hanya asc atau desc
	if q.Order != "desc" {
		q.Order = "asc"
	}

	// Filter is_active ΓÇö opsional
	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	// Filter grade_min ΓÇö opsional, untuk menyaring berdasarkan rentang nilai
	if raw := c.Query("grade_min"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMin = &v
		}
	}

	// Filter grade_max ΓÇö opsional
	if raw := c.Query("grade_max"); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			q.GradeMax = &v
		}
	}

	return q
}
