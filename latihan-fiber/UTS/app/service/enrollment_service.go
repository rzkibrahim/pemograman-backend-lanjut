// Package service - enrollment_service.go
// Mengelola logika bisnis pengambilan dan pembatalan KRS mahasiswa:
// - Pengecekan duplikasi mata kuliah pada tahun akademik yang sama (Error 409)
// - Row-level locking pada mata kuliah untuk mencegah overbooking kuota (Error 422)
// - Validasi batas maksimal SKS berdasarkan IPK terakhir mahasiswa (Error 422)
// - Otorisasi kepemilikan KRS (Error 403)
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

// EnrollmentService mengelola operasi KRS mahasiswa.
type EnrollmentService struct {
	pool           *pgxpool.Pool
	studentRepo    repository.StudentRepository
	courseRepo     repository.CourseRepository
	enrollmentRepo repository.EnrollmentRepository
}

// NewEnrollmentService membuat instance baru EnrollmentService.
func NewEnrollmentService(
	pool *pgxpool.Pool,
	studentRepo repository.StudentRepository,
	courseRepo repository.CourseRepository,
	enrollmentRepo repository.EnrollmentRepository,
) *EnrollmentService {
	return &EnrollmentService{
		pool:           pool,
		studentRepo:    studentRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

// Create memproses pengambilan mata kuliah (tambah KRS) oleh mahasiswa (Endpoint 9 UTS).
// Seluruh verifikasi aturan bisnis dijalankan dalam satu transaksi database atomik.
func (s *EnrollmentService) Create(
	ctx context.Context,
	currentUser *helper.JWTPayload,
	req model.CreateEnrollmentRequest,
) (*model.EnrollmentResponse, error) {
	// 1. Verifikasi peran pengguna
	if currentUser.Role != model.RoleMahasiswa {
		return nil, helper.Forbidden("akses ditolak: hanya mahasiswa yang dapat mengambil mata kuliah")
	}

	// 2. Ambil profil mahasiswa berdasarkan user ID login
	student, err := s.studentRepo.FindByUserID(ctx, currentUser.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrStudentNotFound) {
			return nil, helper.NotFound("profil mahasiswa tidak ditemukan atau telah dinonaktifkan")
		}
		return nil, helper.Internal(err)
	}

	// 3. Mulai Transaksi Database
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, helper.Internal(err)
	}
	defer tx.Rollback(ctx)

	// 4. Business Rule 2: Cek apakah mahasiswa sudah mengambil mata kuliah ini pada tahun akademik yang sama
	existing, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, tx, student.ID, req.CourseID, req.TahunAkademik)
	if err != nil {
		return nil, helper.Internal(err)
	}
	if existing != nil {
		return nil, helper.Conflict("mata kuliah ini sudah pernah Anda ambil pada tahun akademik yang sama")
	}

	// 5. Kunci baris mata kuliah (SELECT ... FOR UPDATE) untuk mencegah overbooking kuota akibat race condition
	course, err := s.courseRepo.FindByIDForUpdate(ctx, tx, req.CourseID)
	if err != nil {
		if errors.Is(err, repository.ErrCourseNotFound) {
			return nil, helper.NotFound("mata kuliah tidak ditemukan")
		}
		return nil, helper.Internal(err)
	}

	// 6. Business Rule 3: Cek apakah kuota mata kuliah masih tersedia
	enrolledCount, err := s.enrollmentRepo.GetCourseEnrollmentCount(ctx, tx, course.ID)
	if err != nil {
		return nil, helper.Internal(err)
	}
	if enrolledCount >= course.Kuota {
		return nil, helper.Validation(map[string][]string{
			"kuota": {fmt.Sprintf("kuota mata kuliah %s sudah penuh (kapasitas: %d)", course.NamaMK, course.Kuota)},
		})
	}

	// 7. Business Rule 1: Hitung batas maksimal SKS berdasarkan IPK terakhir
	// IPK >= 3.00 -> 24 SKS
	// IPK 2.50 - 2.99 -> 21 SKS
	// IPK < 2.50 -> 18 SKS
	batasSKS := model.HitungBatasSKS(student.IPKTerakhir)

	// Ambil total SKS yang telah diambil pada tahun akademik yang bersangkutan
	currentTotalSKS, err := s.enrollmentRepo.GetTotalSKSByStudent(ctx, tx, student.ID, req.TahunAkademik)
	if err != nil {
		return nil, helper.Internal(err)
	}

	if currentTotalSKS+course.SKS > batasSKS {
		sisaSKS := batasSKS - currentTotalSKS
		if sisaSKS < 0 {
			sisaSKS = 0
		}
		return nil, helper.Validation(map[string][]string{
			"sks": {fmt.Sprintf("Pengambilan mata kuliah melebihi batas SKS semester (%d SKS). Sisa SKS Anda saat ini adalah %d SKS, sedangkan mata kuliah %s membutuhkan %d SKS.", batasSKS, sisaSKS, course.NamaMK, course.SKS)},
		})
	}

	// 8. Simpan record enrollment baru
	createdEnrollment, err := s.enrollmentRepo.CreateWithTx(ctx, tx, student.ID, course.ID, req.TahunAkademik)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyEnrolled) {
			return nil, helper.Conflict("mata kuliah ini sudah pernah Anda ambil pada tahun akademik yang sama")
		}
		return nil, helper.Internal(err)
	}

	// 9. Komit transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, helper.Internal(err)
	}

	return &model.EnrollmentResponse{
		ID:            createdEnrollment.ID,
		StudentID:     student.ID,
		CourseID:      course.ID,
		KodeMK:        course.KodeMK,
		NamaMK:        course.NamaMK,
		SKS:           course.SKS,
		TahunAkademik: createdEnrollment.TahunAkademik,
		CreatedAt:     createdEnrollment.CreatedAt,
	}, nil
}

// Delete memproses pembatalan mata kuliah dari KRS mahasiswa (Endpoint 10 UTS).
// Menerapkan Business Rule 4: Mahasiswa hanya dapat membatalkan KRS miliknya sendiri.
func (s *EnrollmentService) Delete(ctx context.Context, currentUser *helper.JWTPayload, enrollmentID int) error {
	// 1. Verifikasi peran pengguna
	if currentUser.Role != model.RoleMahasiswa {
		return helper.Forbidden("akses ditolak: hanya mahasiswa yang dapat membatalkan KRS")
	}

	// 2. Ambil profil mahasiswa yang sedang login
	student, err := s.studentRepo.FindByUserID(ctx, currentUser.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrStudentNotFound) {
			return helper.NotFound("profil mahasiswa tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// 3. Ambil data enrollment target
	enrollment, err := s.enrollmentRepo.FindByID(ctx, enrollmentID)
	if err != nil {
		if errors.Is(err, repository.ErrEnrollmentNotFound) {
			return helper.NotFound("data KRS tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// 4. Verifikasi kepemilikan: Apakah record KRS ini milik mahasiswa yang bersangkutan?
	if enrollment.StudentID != student.ID {
		return helper.Forbidden("akses ditolak: Anda hanya dapat membatalkan mata kuliah dari KRS milik Anda sendiri")
	}

	// 5. Hapus record enrollment (kuota otomatis bertambah kembali)
	if err := s.enrollmentRepo.DeleteByID(ctx, enrollmentID); err != nil {
		return helper.Internal(err)
	}

	return nil
}
