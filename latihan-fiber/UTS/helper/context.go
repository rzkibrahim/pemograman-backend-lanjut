// Package helper - context.go
// Berisi helper untuk manajemen context request, timeout, pengambilan user login,
// dan parsing parameter ID dari URL.
package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// DefaultTimeout adalah batas waktu maksimum eksekusi query ke database per-request
const DefaultTimeout = 5 * time.Second

// RequestContext membuat context.Context dengan batas waktu timeout 5 detik
// untuk mencegah goroutine leak dan koneksi DB menggantung tanpa batas.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), DefaultTimeout)
}

// CurrentUser mengambil data JWT payload dari Locals context Fiber
// yang sebelumnya telah disuntikkan oleh RequireAuth middleware.
func CurrentUser(c *fiber.Ctx) (*JWTPayload, error) {
	user, ok := c.Locals("user").(*JWTPayload)
	if !ok || user == nil {
		return nil, Unauthorized("token autentikasi diperlukan atau sesi tidak valid")
	}
	return user, nil
}

// ParamID membaca dan memvalidasi path parameter ID agar berupa bilangan bulat positif.
// Mengembalikan error BadRequest jika format tidak valid atau <= 0.
func ParamID(c *fiber.Ctx, name string) (int, error) {
	val := c.Params(name)
	id, err := strconv.Atoi(val)
	if err != nil || id <= 0 {
		return 0, BadRequest("parameter " + name + " harus berupa bilangan bulat positif")
	}
	return id, nil
}
