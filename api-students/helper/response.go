package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students/app/model"
)

// Success mengembalikan respons sukses dengan status kode dan pesan tertentu.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessList mengembalikan 200 OK dengan metadata paginasi.
func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created mengirim 201 Created sekaligus memasang header Location.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// NoContent mengembalikan 204 No Content untuk operasi tanpa body (misal: DELETE).
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Fail mengembalikan respons gagal dengan status HTTP dan pesan tertentu.
func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}

// FailValidation mengembalikan 422 Unprocessable Entity beserta detail error tiap field.
func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false,
		Message: "validasi gagal",
		Errors:  errs,
	})
}

// FailConflict mengembalikan 409 Conflict bila terjadi duplikasi data unik.
func FailConflict(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}
