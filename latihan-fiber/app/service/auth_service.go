// Package service - auth_service.go
// Menangani alur bisnis autentikasi: login dengan mitigasi timing attack,
// pembuatan access token JWT, dan pemuatan profil diri pengguna (me).
package service

import (
	"context"
	"errors"
	"time"

	"siakad-mini/app/model"
	"siakad-mini/app/repository"
	"siakad-mini/helper"
)

// AuthService mendefinisikan interface logika bisnis autentikasi.
type AuthService interface {
	Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error)
	Me(ctx context.Context, userID int) (*model.AuthMeResponse, error)
}

type authService struct {
	userRepo    repository.UserRepository
	studentRepo repository.StudentRepository
	jwtSecret   string
	jwtIssuer   string
	tokenTTL    time.Duration
}

// NewAuthService membuat instance baru AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	studentRepo repository.StudentRepository,
	jwtSecret string,
	jwtIssuer string,
	tokenTTLMinutes int,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		studentRepo: studentRepo,
		jwtSecret:   jwtSecret,
		jwtIssuer:   jwtIssuer,
		tokenTTL:    time.Duration(tokenTTLMinutes) * time.Minute,
	}
}

// Login memproses autentikasi pengguna dengan kredensial email dan password.
// Mengimplementasikan mitigasi timing attack menggunakan dummy hash perbandingan.
func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Jalankan dummy bcrypt verification untuk menyamarkan waktu respon (mencegah user enumeration)
			helper.VerifyDummyPassword()
			return nil, helper.Unauthorized("email atau password salah")
		}
		return nil, helper.Internal(err)
	}

	// Verifikasi kecocokan password dengan hash bcrypt
	if !helper.VerifyPassword(user.Password, req.Password) {
		return nil, helper.Unauthorized("email atau password salah")
	}

	// Jika role adalah mahasiswa, pastikan mahasiswa belum di-soft delete
	if user.Role == model.RoleMahasiswa {
		student, err := s.studentRepo.FindByUserID(ctx, user.ID)
		if err != nil {
			if errors.Is(err, repository.ErrStudentNotFound) {
				return nil, helper.Unauthorized("akun mahasiswa ini telah dinonaktifkan")
			}
			return nil, helper.Internal(err)
		}
		if student.DeletedAt != nil {
			return nil, helper.Unauthorized("akun mahasiswa ini telah dinonaktifkan")
		}
	}

	// Generate JWT Access Token
	tokenString, err := helper.GenerateAccessToken(
		user.ID, user.Email, user.Role, s.jwtIssuer, s.jwtSecret, s.tokenTTL,
	)
	if err != nil {
		return nil, helper.Internal(err)
	}

	return &model.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.tokenTTL.Seconds()),
		User: model.UserDataResponse{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

// Me mengambil data profil pengguna yang sedang login beserta data mahasiswa jika role-nya mahasiswa.
func (s *authService) Me(ctx context.Context, userID int) (*model.AuthMeResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, helper.NotFound("pengguna tidak ditemukan")
		}
		return nil, helper.Internal(err)
	}

	resp := &model.AuthMeResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}

	// Jika pengguna adalah mahasiswa, sisipkan data profil mahasiswa
	if user.Role == model.RoleMahasiswa {
		student, err := s.studentRepo.FindByUserID(ctx, user.ID)
		if err == nil && student != nil {
			resp.Student = &model.StudentProfileSummary{
				ID:          student.ID,
				NIM:         student.NIM,
				Nama:        student.Nama,
				Prodi:       student.Prodi,
				Angkatan:    student.Angkatan,
				IPKTerakhir: student.IPKTerakhir,
			}
		}
	}

	return resp, nil
}
