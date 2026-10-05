package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"modul6/app/model"
	"modul6/app/repository"
	"modul6/helper"
)

// UserService mengelola operasi CRUD data pengguna dan penugasan peran (role assignment).
type UserService struct {
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

// NewUserService membuat instansiasi UserService baru.
func NewUserService(repo repository.UserRepository, perms *helper.PermissionSet) *UserService {
	return &UserService{
		repo:  repo,
		perms: perms,
	}
}

// List menampilkan seluruh daftar pengguna.
// Endpoint dilindungi oleh RequirePermission("user:list") di route.go.
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	users, err := s.repo.FindAll(ctx)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar user")
	}

	responses := make([]model.UserResponse, len(users))
	for i, u := range users {
		responses[i] = model.UserResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		}
	}

	return helper.Success(c, fiber.StatusOK, "daftar pengguna berhasil diambil", responses)
}

// Get menampilkan detail seorang pengguna.
// Ownership Check: Pemilik akun boleh melihat datanya sendiri.
// Pengguna lain WAJIB memiliki permission "user:read:any".
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	// Evaluasi otorisasi
	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda bukan pemilik akun ini dan tidak memiliki izin user:read:any")
	}

	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data user")
	}

	return helper.Success(c, fiber.StatusOK, "data user ditemukan", model.UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	})
}

// Update memperbarui data profil pengguna (misalnya email).
// Ownership Check: Pemilik akun boleh mengubah profilnya sendiri.
// Pengguna lain WAJIB memiliki permission "user:update:any".
func (s *UserService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	current, ok := helper.GetCurrentAuthUser(c)
	if !ok {
		return helper.Fail(c, fiber.StatusUnauthorized, "token autentikasi diperlukan")
	}

	// Evaluasi otorisasi
	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: Anda bukan pemilik akun ini dan tidak memiliki izin user:update:any")
	}

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return helper.Fail(c, fiber.StatusBadRequest, "format email tidak valid")
	}

	updated, err := s.repo.Update(ctx, model.User{
		ID:    id,
		Email: req.Email,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
		}
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Fail(c, fiber.StatusConflict, "email sudah digunakan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui user")
	}

	return helper.Success(c, fiber.StatusOK, "profil user berhasil diperbarui", model.UserResponse{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		Role:      updated.Role,
		CreatedAt: updated.CreatedAt,
	})
}

// AssignRole mengubah role seorang pengguna.
// Endpoint dilindungi oleh RequirePermission("role:assign") di route.go (Admin only).
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role != "admin" && req.Role != "staff" && req.Role != "user" {
		return helper.Fail(c, fiber.StatusBadRequest, "role tidak valid: hanya boleh 'admin', 'staff', atau 'user'")
	}

	updated, err := s.repo.UpdateRole(ctx, id, req.Role)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui role user")
	}

	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", model.UserResponse{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		Role:      updated.Role,
		CreatedAt: updated.CreatedAt,
	})
}

// Delete menghapus seorang pengguna dari sistem.
// Endpoint dilindungi oleh RequirePermission("user:delete") di route.go (Admin only).
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Fail(c, fiber.StatusNotFound, "user tidak ditemukan")
		}
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus user")
	}

	return helper.NoContent(c)
}
