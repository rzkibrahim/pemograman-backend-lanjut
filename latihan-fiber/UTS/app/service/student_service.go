// Package service - student_service.go
// Mengelola logika bisnis data mahasiswa: pendaftaran mahasiswa & akun user secara transaksional,
// otorisasi akses profil sendiri, pagination list, kalkulasi batas SKS, dan soft delete.
package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

// StudentService mengelola operasi mahasiswa.
type StudentService struct {
	pool           *pgxpool.Pool
	userRepo       repository.UserRepository
	studentRepo    repository.StudentRepository
	enrollmentRepo repository.EnrollmentRepository
}

// NewStudentService membuat instance baru StudentService.
func NewStudentService(
	pool *pgxpool.Pool,
	userRepo repository.UserRepository,
	studentRepo repository.StudentRepository,
	enrollmentRepo repository.EnrollmentRepository,
) *StudentService {
	return &StudentService{
		pool:           pool,
		userRepo:       userRepo,
		studentRepo:    studentRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

// List mengambil daftar mahasiswa dengan pagination dan filter.
func (s *StudentService) List(ctx context.Context, q model.StudentListQuery) ([]model.StudentListItem, *model.PaginationMeta, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PerPage < 1 || q.PerPage > 50 {
		q.PerPage = 10
	}

	items, total, err := s.studentRepo.List(ctx, q)
	if err != nil {
		return nil, nil, helper.Internal(err)
	}

	lastPage := 1
	if total > 0 {
		lastPage = int(math.Ceil(float64(total) / float64(q.PerPage)))
	}

	meta := &model.PaginationMeta{
		CurrentPage: q.Page,
		PerPage:     q.PerPage,
		Total:       total,
		LastPage:    lastPage,
	}

	return items, meta, nil
}

// Create mendaftarkan mahasiswa baru sekaligus membuat akun pengguna (users)
// dalam satu transaksi database atomik (Endpoint 4 UTS).
// Password awal diset sama dengan NIM yang di-hash menggunakan bcrypt.
func (s *StudentService) Create(ctx context.Context, req model.CreateStudentRequest) (*model.Student, error) {
	// Cek apakah email sudah terdaftar sebelumnya
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, helper.Validation(map[string][]string{
			"email": {"Email sudah terdaftar"},
		})
	}

	// Cek apakah NIM sudah terdaftar sebelumnya
	existingStudent, err := s.studentRepo.FindByNIM(ctx, req.NIM)
	if err == nil && existingStudent != nil {
		return nil, helper.Validation(map[string][]string{
			"nim": {"NIM sudah terdaftar"},
		})
	}

	// Hash password awal (default = NIM)
	hashedPassword, err := helper.HashPassword(req.NIM)
	if err != nil {
		return nil, helper.Internal(fmt.Errorf("gagal hashing password default: %w", err))
	}

	// Mulai transaksi database
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, helper.Internal(err)
	}
	defer tx.Rollback(ctx)

	// 1. Simpan ke tabel `users` dengan role 'mahasiswa'
	createdUser, err := s.userRepo.CreateWithTx(ctx, tx, req.Email, hashedPassword, model.RoleMahasiswa)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExist) {
			return nil, helper.Validation(map[string][]string{
				"email": {"Email sudah terdaftar"},
			})
		}
		return nil, helper.Internal(err)
	}

	ipk := 0.0
	if req.IPKTerakhir != nil {
		ipk = *req.IPKTerakhir
	}

	// 2. Simpan ke tabel `students` dengan relasi user_id
	newStudent := &model.Student{
		UserID:      createdUser.ID,
		NIM:         req.NIM,
		Nama:        req.Nama,
		Prodi:       req.Prodi,
		Angkatan:    req.Angkatan,
		IPKTerakhir: ipk,
	}

	createdStudent, err := s.studentRepo.CreateWithTx(ctx, tx, newStudent)
	if err != nil {
		if errors.Is(err, repository.ErrNIMAlreadyExist) {
			return nil, helper.Validation(map[string][]string{
				"nim": {"NIM sudah terdaftar"},
			})
		}
		return nil, helper.Internal(err)
	}

	// Komit transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, helper.Internal(err)
	}

	return createdStudent, nil
}

// GetDetail mengambil detail data mahasiswa beserta daftar mata kuliah yang diambil (KRS),
// total SKS, dan batas maksimal SKS (Endpoint 5 UTS).
// Menerapkan otorisasi: Mahasiswa hanya boleh melihat data profil miliknya sendiri.
func (s *StudentService) GetDetail(ctx context.Context, id int, currentUser *helper.JWTPayload) (*model.StudentDetailResponse, error) {
	// Ambil profil mahasiswa target
	student, err := s.studentRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrStudentNotFound) {
			return nil, helper.NotFound("data mahasiswa tidak ditemukan atau telah dinonaktifkan")
		}
		return nil, helper.Internal(err)
	}

	// Cek otorisasi jika pengguna adalah mahasiswa
	if currentUser.Role == model.RoleMahasiswa {
		myStudent, err := s.studentRepo.FindByUserID(ctx, currentUser.UserID)
		if err != nil || myStudent == nil || myStudent.ID != student.ID {
			return nil, helper.Forbidden("akses ditolak: Anda hanya dapat mengakses data mahasiswa milik Anda sendiri")
		}
	}

	// Ambil daftar mata kuliah yang diambil (KRS)
	courses, err := s.enrollmentRepo.ListByStudentID(ctx, student.ID)
	if err != nil {
		return nil, helper.Internal(err)
	}

	// Hitung total SKS yang diambil
	totalSKS := 0
	for _, c := range courses {
		totalSKS += c.SKS
	}

	// Hitung batas SKS berdasarkan IPK terakhir
	batasSKS := model.HitungBatasSKS(student.IPKTerakhir)

	return &model.StudentDetailResponse{
		ID:          student.ID,
		NIM:         student.NIM,
		Nama:        student.Nama,
		Prodi:       student.Prodi,
		Angkatan:    student.Angkatan,
		IPKTerakhir: student.IPKTerakhir,
		TotalSKS:    totalSKS,
		BatasSKS:    batasSKS,
		Courses:     courses,
	}, nil
}

// Update memperbarui data nama, prodi, angkatan, dan IPK mahasiswa (Endpoint 6 UTS).
// NIM tidak boleh diubah sesuai spesifikasi.
func (s *StudentService) Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, error) {
	updated, err := s.studentRepo.Update(ctx, id, req)
	if err != nil {
		if errors.Is(err, repository.ErrStudentNotFound) {
			return nil, helper.NotFound("data mahasiswa tidak ditemukan")
		}
		return nil, helper.Internal(err)
	}
	return updated, nil
}

// Delete melakukan soft delete mahasiswa (mengisi deleted_at) (Endpoint 7 UTS).
func (s *StudentService) Delete(ctx context.Context, id int) error {
	err := s.studentRepo.SoftDelete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrStudentNotFound) {
			return helper.NotFound("data mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}
	return nil
}
