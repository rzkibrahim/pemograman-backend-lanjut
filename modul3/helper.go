package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"modul3/app/model"
)

// === Helper Respons ===

// ok mengembalikan 200 OK -- permintaan berhasil dan ada isi yang dikirim.
func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

// okList mengembalikan 200 OK dengan metadata paginasi.
func okList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

// created mengembalikan 201 Created -- sumber daya baru berhasil dibuat.
// Header Location dikirimkan agar pengguna tahu di mana sumber daya baru berada.
func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

// noContent mengembalikan 204 No Content -- berhasil, tanpa body.
// Umumnya dipakai untuk DELETE.
func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// fail mengembalikan respons gagal dengan status dan pesan tertentu.
func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{Success: false, Message: message})
}

// failValidation mengembalikan 422 Unprocessable Entity.
func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

// failConflict mengembalikan 409 Conflict.
func failConflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(model.WebResponse{
		Success: false, Message: message,
	})
}

// === Helper Context ===

// reqCtx memberi batas waktu untuk setiap operasi basis data.
func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// === Helper Parameter ===

// paramID membaca parameter :id dari URL dan memvalidasinya.
func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// === Daftar Putih dan Parser Query String ===

var allowedSort = map[string]bool{
	"id":         true,
	"nim":        true,
	"name":       true,
	"grade":      true,
	"created_at": true,
}

// parseListQuery membaca query string dan memberi nilai bawaan yang aman.
func parseListQuery(c *fiber.Ctx) model.ListQuery {
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

