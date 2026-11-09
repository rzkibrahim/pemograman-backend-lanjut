package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type UserService struct {
	repo  repository.UserRepository
	perms *helper.PermissionSet
}

func NewUserService(repo repository.UserRepository, perms *helper.PermissionSet) *UserService {
	return &UserService{
		repo:  repo,
		perms: perms,
	}
}

// List godoc
// @Summary      Daftar user
// @Description  Mengambil daftar user dengan cursor pagination atau format CSV
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Param        cursor query string false "Cursor untuk pagination"
// @Param        limit query int false "Jumlah data per halaman" default(10)
// @Success      200 {object} model.WebResponse{data=[]model.User}
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Router       /users [get]
func (s *UserService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	if format == helper.FormatCSV {
		return helper.WriteUsersCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar user berhasil diambil", rows, meta)
}

// Get godoc
// @Summary      Detail satu user
// @Description  Mengambil data user berdasarkan ID
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "ID user"
// @Success      200 {object} model.WebResponse{data=model.User}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /users/{id} [get]
func (s *UserService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if !CanAccessUser(current, id, s.perms, "user:read:any") {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "user ditemukan", user)
}

// Update godoc
// @Summary      Memperbarui profil user
// @Description  Memperbarui email user
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "ID user"
// @Param        body body model.UpdateUserRequest true "Data update user"
// @Success      200 {object} model.WebResponse{data=model.UserResponse}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Failure      422 {object} model.ErrorResponse
// @Router       /users/{id} [put]
func (s *UserService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	if !CanAccessUser(current, id, s.perms, "user:update:any") {
		return helper.Forbidden("tidak berhak mengakses data user lain")
	}

	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return helper.Validation(map[string]string{"email": "format email tidak valid"})
	}

	updated, err := s.repo.Update(ctx, model.User{
		ID:    id,
		Email: req.Email,
	})
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "profil user berhasil diperbarui", model.UserResponse{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		Role:      updated.Role,
		CreatedAt: updated.CreatedAt,
	})
}

// AssignRole godoc
// @Summary      Mengubah role user (admin)
// @Description  Memerlukan permission role:assign
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "ID user"
// @Param        body body model.AssignRoleRequest true "Role baru"
// @Success      200 {object} model.WebResponse{data=model.UserResponse}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /users/{id}/role [patch]
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	updated, err := s.repo.UpdateRole(ctx, id, strings.ToLower(strings.TrimSpace(req.Role)))
	if err != nil {
		return translateError(err, "user")
	}

	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", model.UserResponse{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		Role:      updated.Role,
		CreatedAt: updated.CreatedAt,
	})
}

// Delete godoc
// @Summary      Menghapus user
// @Description  Memerlukan permission user:delete
// @Tags         users
// @Security     BearerAuth
// @Param        id path int true "ID user"
// @Success      204 "Berhasil menghapus; tanpa body"
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /users/{id} [delete]
func (s *UserService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "user")
	}

	return helper.NoContent(c)
}

func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("username sudah dipakai")
	default:
		return helper.Internal(err)
	}
}
