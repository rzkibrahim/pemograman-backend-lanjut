package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"modul5/app/model"
	"modul5/app/repository"
	"modul5/helper"
)

var (
	// ErrInvalidCredentials digunakan untuk username atau password salah (pesan identik anti enumeration).
	ErrInvalidCredentials = errors.New("username atau password salah")
	// ErrUserExists menandakan username atau email sudah digunakan.
	ErrUserExists = errors.New("username atau email sudah terdaftar")
	// ErrTokenRevoked menandakan token yang dikirim telah dicabut.
	ErrTokenRevoked = errors.New("refresh token telah dicabut")
	// ErrRefreshTokenExpired menandakan refresh token telah kedaluwarsa.
	ErrRefreshTokenExpired = errors.New("refresh token telah kedaluwarsa")
)

// AuthService mendefinisikan seluruh usecase autentikasi dan otorisasi dasar.
type AuthService interface {
	Register(ctx context.Context, req model.RegisterRequest) (model.UserResponse, map[string]string, error)
	Login(ctx context.Context, req model.LoginRequest) (model.AuthResponse, map[string]string, error)
	RefreshToken(ctx context.Context, req model.RefreshTokenRequest) (model.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID int) (model.UserResponse, error)
}

type authService struct {
	userRepo   repository.UserRepository
	tokenRepo  repository.TokenRepository
	jwtSecret  string
	jwtIssuer  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService membuat instansiasi AuthService baru.
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtSecret string,
	jwtIssuer string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) AuthService {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	return &authService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  jwtSecret,
		jwtIssuer:  jwtIssuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (model.UserResponse, map[string]string, error) {
	// 1. Validasi input
	if errs := ValidateRegister(req); len(errs) > 0 {
		return model.UserResponse{}, errs, errors.New("validasi gagal")
	}

	// 2. Periksa keunikan username
	if _, err := s.userRepo.FindByUsername(ctx, req.Username); err == nil {
		return model.UserResponse{}, map[string]string{"username": "sudah dipakai"}, ErrUserExists
	}

	// 3. Periksa keunikan email
	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return model.UserResponse{}, map[string]string{"email": "sudah dipakai"}, ErrUserExists
	}

	// 4. Hash password dengan bcrypt cost factor 12
	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return model.UserResponse{}, nil, fmt.Errorf("hashing password: %w", err)
	}

	// 5. Simpan pengguna baru (role SELALU dipaksa 'user' oleh server)
	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user", // Anti mass-assignment
		IsActive: true,
	}

	createdUser, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return model.UserResponse{}, nil, fmt.Errorf("menyimpan pengguna: %w", err)
	}

	return model.UserResponse{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		Role:      createdUser.Role,
		CreatedAt: createdUser.CreatedAt,
	}, nil, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (model.AuthResponse, map[string]string, error) {
	// 1. Validasi input login
	if errs := ValidateLogin(req); len(errs) > 0 {
		return model.AuthResponse{}, errs, errors.New("validasi gagal")
	}

	// 2. Cari user di basis data
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		// PENCEGAHAN USER ENUMERATION & TIMING ATTACK:
		// Jalankan komputasi dummy hash agar latensi waktu sama persis
		helper.VerifyDummyPassword()
		return model.AuthResponse{}, nil, ErrInvalidCredentials
	}

	// 3. Verifikasi password hash
	if !helper.VerifyPassword(user.Password, req.Password) {
		return model.AuthResponse{}, nil, ErrInvalidCredentials
	}

	// 4. Buat Access Token JWT (15 menit)
	accessToken, err := helper.GenerateAccessToken(user.ID, user.Username, user.Role, s.jwtIssuer, s.jwtSecret, s.accessTTL)
	if err != nil {
		return model.AuthResponse{}, nil, fmt.Errorf("membuat access token: %w", err)
	}

	// 5. Buat Refresh Token acak (64 hex characters)
	rawRefreshToken, err := helper.RandomToken(32)
	if err != nil {
		return model.AuthResponse{}, nil, fmt.Errorf("membuat refresh token: %w", err)
	}

	// 6. Simpan hash SHA-256 dari refresh token di database
	tokenHash := helper.SHA256Hex(rawRefreshToken)
	refreshTokenEntity := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if _, err := s.tokenRepo.Create(ctx, refreshTokenEntity); err != nil {
		return model.AuthResponse{}, nil, fmt.Errorf("menyimpan refresh token: %w", err)
	}

	return model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil, nil
}

// RefreshToken, Logout, dan Me akan diimplementasikan penuh pada commit berikutnya.
func (s *authService) RefreshToken(ctx context.Context, req model.RefreshTokenRequest) (model.AuthResponse, error) {
	return model.AuthResponse{}, errors.New("not implemented")
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	return errors.New("not implemented")
}

func (s *authService) Me(ctx context.Context, userID int) (model.UserResponse, error) {
	return model.UserResponse{}, errors.New("not implemented")
}
