// Package helper - response.go
// Berisi fungsi-fungsi pembantu untuk mengirim response JSON yang seragam.
// Semua response sukses menggunakan struktur WebResponse yang konsisten.
package helper

import (
	"github.com/gofiber/fiber/v2"

	"siakad-mini/app/model"
)

// Success mengirim response JSON sukses dengan status code dan data tertentu.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessList mengirim response JSON sukses dengan pagination metadata.
func SuccessList(c *fiber.Ctx, message string, data any, meta *model.PaginationMeta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim response JSON 201 Created dengan header Location.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengirim response 204 No Content (tanpa body).
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// RequestID mengekstrak request ID dari Locals Fiber context.
func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals("requestid").(string)
	return id
}
