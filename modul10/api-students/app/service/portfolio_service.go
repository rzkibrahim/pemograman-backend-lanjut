package service

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"modul10/app/model"
	"modul10/app/repository"
	"modul10/helper"
)

type PortfolioService struct {
	portfolios *repository.PortfolioRepository
	students   repository.StudentRepository
	perms      *helper.PermissionSet
}

func NewPortfolioService(portfolios *repository.PortfolioRepository, students repository.StudentRepository, perms *helper.PermissionSet) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students, perms: perms}
}

func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, err := s.authorizeStudent(c, ctx, "student:read:any")
	if err != nil {
		return err
	}
	portfolio, err := s.portfolios.FindByStudentID(ctx, studentID)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil diambil", portfolio)
}

func (s *PortfolioService) Upsert(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, err := s.authorizeStudent(c, ctx, "student:update:any")
	if err != nil {
		return err
	}
	var req model.UpsertPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(req); len(fields) > 0 {
		return helper.Validation(fields)
	}
	portfolio, err := s.portfolios.Upsert(ctx, studentID, strings.TrimSpace(req.Headline), strings.TrimSpace(req.Bio))
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", portfolio)
}

func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, err := s.authorizeStudent(c, ctx, "student:update:any")
	if err != nil {
		return err
	}
	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(req); len(fields) > 0 {
		return helper.Validation(fields)
	}
	projectID, err := s.portfolios.AddProject(ctx, studentID, model.PortfolioProject{
		Title: strings.TrimSpace(req.Title), Technologies: req.Technologies,
		Link: strings.TrimSpace(req.Link), Year: req.Year,
	})
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "proyek berhasil ditambahkan", fiber.Map{"project_id": projectID})
}

func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, err := s.authorizeStudent(c, ctx, "student:update:any")
	if err != nil {
		return err
	}
	if err := s.portfolios.DeleteProject(ctx, studentID, c.Params("projectId")); err != nil {
		return translatePortfolioError(err)
	}
	return helper.NoContent(c)
}

func (s *PortfolioService) TechnologyStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	stats, err := s.portfolios.GetTechnologyStats(ctx)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi berhasil diambil", stats)
}

func (s *PortfolioService) authorizeStudent(c *fiber.Ctx, ctx context.Context, permission string) (int, error) {
	studentID, valid := helper.ParamID(c)
	if !valid {
		return 0, helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	current, ok := helper.CurrentUser(c)
	if !ok {
		return 0, helper.Unauthorized("token autentikasi diperlukan")
	}
	student, err := s.students.FindByID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return 0, helper.NotFound("mahasiswa tidak ditemukan")
		}
		return 0, helper.Internal(err)
	}
	if !CanAccessStudent(current, student.OwnerID, s.perms, permission) {
		return 0, helper.Forbidden("Anda tidak berhak mengakses portofolio mahasiswa ini")
	}
	return studentID, nil
}
