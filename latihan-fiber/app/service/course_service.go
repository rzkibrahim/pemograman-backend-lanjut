// Package service - course_service.go
// Menangani logika bisnis pengambilan daftar mata kuliah beserta kalkulasi sisa kuota.
package service

import (
	"context"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

// CourseService mengelola operasi mata kuliah.
type CourseService struct {
	courseRepo repository.CourseRepository
}

// NewCourseService membuat instance baru CourseService.
func NewCourseService(courseRepo repository.CourseRepository) *CourseService {
	return &CourseService{courseRepo: courseRepo}
}

// List mengambil daftar mata kuliah yang mendukung pencarian kode/nama, filter semester,
// dan opsi filter kuota tersedia (Endpoint 8 UTS).
func (s *CourseService) List(ctx context.Context, q model.CourseQuery) ([]model.CourseItemResponse, error) {
	courses, err := s.courseRepo.List(ctx, q)
	if err != nil {
		return nil, helper.Internal(err)
	}
	return courses, nil
}
