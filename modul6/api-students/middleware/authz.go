package middleware

import (
	"github.com/gofiber/fiber/v2"

	"modul6/helper"
)

// RequirePermission memeriksa apakah pengguna yang telah terautentikasi memiliki permission yang disyaratkan.
//
// Middleware ini WAJIB selalu dipasang SETELAH RequireAuth:
// 1. Authentication (RequireAuth): "Siapa Anda?" -> Jika gagal = 401 Unauthorized
// 2. Authorization (RequirePermission): "Boleh melakukan apa?" -> Jika gagal = 403 Forbidden
//
// Menerapkan prinsip Fail-Closed:
// Jika perms bernilai nil, user tidak terdaftar, atau permission tidak terpenuhi,
// akses SELALU ditolak (403 Forbidden).
func RequirePermission(perms *helper.PermissionSet, requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Ambil identitas pengguna terautentikasi dari Locals context
		user, ok := helper.GetCurrentUser(c)
		if !ok || user == nil {
			return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
		}

		// 2. Periksa permission fail-closed
		if perms == nil || !perms.Can(user.Role, requiredPerm) {
			return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda tidak memiliki hak akses ("+requiredPerm+")")
		}

		// 3. Hak akses terverifikasi, lanjutkan request
		return c.Next()
	}
}
