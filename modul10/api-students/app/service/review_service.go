package service

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"modul10/app/model"
	"modul10/app/repository"
	"modul10/helper"
)

type ReviewService struct {
	reviews *repository.ReviewRepository
	courses *repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(reviews *repository.ReviewRepository, courses *repository.CourseRepository, perms *helper.PermissionSet) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	courseID, err := parsePositiveID(c.Params("id"), "id kelas")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateCourseError(err)
	}
	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 || limit > 50 {
		return helper.BadRequest("limit harus antara 1 dan 50")
	}
	reviews, err := s.reviews.FindByCourse(ctx, courseID, strings.TrimSpace(c.Query("tag")), c.Query("before"), limit)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar ulasan berhasil diambil", reviews)
}

func (s *ReviewService) Summary(c *fiber.Ctx) error {
	courseID, err := parsePositiveID(c.Params("id"), "id kelas")
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateCourseError(err)
	}
	summary, err := s.reviews.GetSummary(ctx, courseID)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ringkasan ulasan berhasil diambil", summary)
}

func (s *ReviewService) Create(c *fiber.Ctx) error {
	courseID, err := parsePositiveID(c.Params("id"), "id kelas")
	if err != nil {
		return err
	}
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token autentikasi diperlukan")
	}
	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(req); len(fields) > 0 {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateCourseError(err)
	}
	enrolled, err := s.courses.IsEnrolled(ctx, current.ID, courseID)
	if err != nil {
		return helper.Internal(err)
	}
	if !enrolled {
		return helper.Forbidden("ulasan hanya dapat dibuat oleh mahasiswa yang terdaftar di kelas")
	}
	review, err := s.reviews.Create(ctx, model.Review{
		CourseID: courseID, UserID: current.ID, Username: current.Username,
		Rating: req.Rating, Comment: strings.TrimSpace(req.Comment), Tags: req.Tags,
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Created(c, "ulasan berhasil dibuat", review, "/api/v1/reviews/"+review.ID)
}

func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	review, err := s.reviews.FindByID(ctx, c.Params("id"))
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan ditemukan", review)
}

func (s *ReviewService) Update(c *fiber.Ctx) error {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token autentikasi diperlukan")
	}
	var req model.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if req.Rating == nil && req.Comment == nil && req.Tags == nil {
		return helper.BadRequest("minimal satu field ulasan harus diubah")
	}
	if fields := helper.ValidateStruct(req); len(fields) > 0 {
		return helper.Validation(fields)
	}
	if req.Comment != nil {
		trimmed := strings.TrimSpace(*req.Comment)
		req.Comment = &trimmed
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	review, err := s.reviews.UpdateOwn(ctx, c.Params("id"), current.ID, req.Rating, req.Comment, req.Tags)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan berhasil diperbarui", review)
}

func (s *ReviewService) Delete(c *fiber.Ctx) error {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token autentikasi diperlukan")
	}
	moderator := s.perms != nil && s.perms.Can(current.Role, "review:moderate")
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if err := s.reviews.Delete(ctx, c.Params("id"), current.ID, moderator); err != nil {
		return translateReviewError(err)
	}
	return helper.NoContent(c)
}

func (s *ReviewService) AddReply(c *fiber.Ctx) error {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token autentikasi diperlukan")
	}
	var req model.AddReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(req); len(fields) > 0 {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	err := s.reviews.AddReply(ctx, c.Params("id"), model.ReviewReply{
		UserID: current.ID, Username: current.Username, Comment: strings.TrimSpace(req.Comment),
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "balasan berhasil ditambahkan", nil)
}

func (s *ReviewService) MarkHelpful(c *fiber.Ctx) error {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token autentikasi diperlukan")
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if err := s.reviews.MarkHelpful(ctx, c.Params("id"), current.ID); err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan ditandai membantu", nil)
}

func parsePositiveID(raw, label string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, helper.BadRequest(label + " harus berupa angka positif")
	}
	return id, nil
}
