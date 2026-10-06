package helper

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"modul7/app/model"
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

// CurrentUser adalah alias praktis untuk GetCurrentAuthUser.
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	return GetCurrentAuthUser(c)
}

// RequestContext membungkus c.UserContext() dengan timeout default 5 detik.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}
