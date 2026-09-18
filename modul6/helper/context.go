package helper

import (
	"github.com/gofiber/fiber/v2"

	"modul6/app/model"
)

const userContextKey = "currentUser"

// SetCurrentUser menyimpan data payload user terautentikasi ke dalam Locals context Fiber.
func SetCurrentUser(c *fiber.Ctx, user *JWTPayload) {
	c.Locals(userContextKey, user)
}

// GetCurrentUser mengambil data payload user terautentikasi dari Locals context Fiber.
func GetCurrentUser(c *fiber.Ctx) (*JWTPayload, bool) {
	val := c.Locals(userContextKey)
	if val == nil {
		return nil, false
	}
	user, ok := val.(*JWTPayload)
	return user, ok
}

// GetCurrentAuthUser mengekstrak model.AuthUser dari Locals context Fiber.
func GetCurrentAuthUser(c *fiber.Ctx) (model.AuthUser, bool) {
	payload, ok := GetCurrentUser(c)
	if !ok || payload == nil {
		return model.AuthUser{}, false
	}
	return model.AuthUser{
		ID:       payload.UserID,
		Username: payload.Username,
		Role:     payload.Role,
	}, true
}

