package middleware

import (
	"github.com/gofiber/fiber/v2"

	"modul7/helper"
)

func RequirePermission(perms *helper.PermissionSet, requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := helper.GetCurrentUser(c)
		if !ok || user == nil {
			return helper.Unauthorized("token autentikasi diperlukan")
		}

		if perms == nil || !perms.Can(user.Role, requiredPerm) {
			return helper.Forbidden("akses ditolak: Anda tidak memiliki hak akses (" + requiredPerm + ")")
		}

		return c.Next()
	}
}