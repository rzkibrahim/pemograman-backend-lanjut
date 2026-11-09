package service

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type ReviewService struct {
	reviews repository.ReviewRepository
	courses repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(reviews repository.ReviewRepository, courses repository.CourseRepository, perms *helper.PermissionSet) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

// ListByCourse godoc
// @Summary     Daftar ulasan kelas
// @Description Mengambil ulasan sebuah kelas dengan filter tag dan pagination cursor (before).
// @Tags        reviews
// @Security    BearerAuth
// @Produce     json
// @Param       id path int true "ID kelas"
// @Param       tag query string false "Saring urutan ulasan mencegat kursus"
// @Param       before query string false "Cursor ulasan sebelum (hex oid)"
// @Param       limit query int false "Jumlah item (maks: 50 / default: 10)"
// @Success     200 {object} model.WebResponse "data: []model.Review"
// @Failure     400 {object} model.WebResponse
// @Failure     404 {object} model.WebResponse
// @Router      /courses/{id}/reviews [get]
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

// Summary godoc
// @Summary      Ringkasan ulasan kelas
// @Description  Mendapatkan statistik rating, distribusi bintang, dan tag ulasan
// @Tags         reviews
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "ID kelas"
// @Success      200 {object} model.WebResponse{data=model.ReviewSummary}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /courses/{id}/reviews/summary [get]
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

// Create godoc
// @Summary     Buat ulasan kelas
// @Description Membuat ulasan untuk kelas yang sedang diikuti; setiap mahasiswa hanya boleh membuat satu ulasan per kelas.
// @Tags        reviews
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path int true "ID kelas"
// @Param       request body model.CreateReviewRequest true "Isi ulasan (rating 1..5, comment, tags)"
// @Success     201 {object} model.WebResponse "data: model.Review"
// @Failure     400 {object} model.WebResponse
// @Failure     403 {object} model.WebResponse "NOT_ENROLLED"
// @Failure     404 {object} model.WebResponse
// @Failure     409 {object} model.WebResponse "ALREADY_REVIEWED"
// @Failure     422 {object} model.WebResponse "VALIDATION_ERROR"
// @Router      /courses/{id}/reviews [post]
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
		return &helper.AppError{
			Status:  fiber.StatusForbidden,
			Code:    "NOT_ENROLLED",
			Message: "ulasan hanya dapat dibuat oleh mahasiswa yang terdaftar di kelas",
		}
	}
	review, err := s.reviews.Create(ctx, model.Review{
		CourseID: courseID, UserID: current.ID, Username: current.Username,
		Rating: req.Rating, Comment: strings.TrimSpace(req.Comment), Tags: NormalizeTags(req.Tags),
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Created(c, "ulasan berhasil dibuat", review, "/api/v1/reviews/"+review.ID)
}

// Get godoc
// @Summary      Detail ulasan
// @Description  Mendapatkan detail satu ulasan berdasarkan ID
// @Tags         reviews
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "ID ulasan (hex oid)"
// @Success      200 {object} model.WebResponse{data=model.Review}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /reviews/{id} [get]
func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	review, err := s.reviews.FindByID(ctx, c.Params("id"))
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan ditemukan", review)
}

// Update godoc
// @Summary      Memperbarui ulasan sendiri
// @Description  Mengubah rating, komentar, atau tags ulasan milik sendiri
// @Tags         reviews
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "ID ulasan (hex oid)"
// @Param        body body model.UpdateReviewRequest true "Data perubahan ulasan"
// @Success      200 {object} model.WebResponse{data=model.Review}
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse "NOT_REVIEW_OWNER"
// @Failure      404 {object} model.ErrorResponse
// @Failure      422 {object} model.ErrorResponse
// @Router       /reviews/{id} [patch]
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

// Delete godoc
// @Summary      Menghapus ulasan
// @Description  Pemilik ulasan atau moderator (admin) dapat menghapus ulasan
// @Tags         reviews
// @Security     BearerAuth
// @Param        id path string true "ID ulasan (hex oid)"
// @Success      204 "Berhasil menghapus; tanpa body"
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse "NOT_REVIEW_OWNER"
// @Failure      404 {object} model.ErrorResponse
// @Router       /reviews/{id} [delete]
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

// AddReply godoc
// @Summary      Menambahkan balasan ulasan
// @Description  Maksimal 50 balasan per ulasan
// @Tags         reviews
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "ID ulasan (hex oid)"
// @Param        body body model.AddReplyRequest true "Isi balasan"
// @Success      201 {object} model.WebResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Failure      409 {object} model.ErrorResponse "REPLY_LIMIT"
// @Failure      422 {object} model.ErrorResponse
// @Router       /reviews/{id}/replies [post]
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

// MarkHelpful godoc
// @Summary      Menandai ulasan bermanfaat (helpful)
// @Description  Satu user hanya dapat memberi satu helpful vote dan tidak bisa vote ulasan sendiri
// @Tags         reviews
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "ID ulasan (hex oid)"
// @Success      200 {object} model.WebResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Failure      409 {object} model.ErrorResponse "OWN_REVIEW / ALREADY_VOTED"
// @Router       /reviews/{id}/helpful [post]
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
