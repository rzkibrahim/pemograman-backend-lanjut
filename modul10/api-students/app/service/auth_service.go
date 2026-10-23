package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"modul10/app/model"
	"modul10/app/repository"
	"modul10/helper"
)

var (
	ErrInvalidCredentials  = errors.New("username atau password salah")
	ErrUserExists          = errors.New("username atau email sudah terdaftar")
	ErrTokenRevoked        = errors.New("refresh token telah dicabut")
	ErrRefreshTokenExpired = errors.New("refresh token telah kedaluwarsa")
)

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
	if errs := helper.ValidateStruct(req); errs != nil {
		return model.UserResponse{}, errs, errors.New("validasi gagal")
	}

	if _, err := s.userRepo.FindByUsername(ctx, req.Username); err == nil {
		return model.UserResponse{}, map[string]string{"username": "sudah dipakai"}, ErrUserExists
	}

	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return model.UserResponse{}, map[string]string{"email": "sudah dipakai"}, ErrUserExists
	}

	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return model.UserResponse{}, nil, fmt.Errorf("hashing password: %w", err)
	}

	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "user",
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
	if errs := helper.ValidateStruct(req); errs != nil {
		return model.AuthResponse{}, errs, errors.New("validasi gagal")
	}

	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		helper.VerifyDummyPassword()
		return model.AuthResponse{}, nil, ErrInvalidCredentials
	}

	if !helper.VerifyPassword(user.Password, req.Password) {
		return model.AuthResponse{}, nil, ErrInvalidCredentials
	}

	accessToken, err := helper.GenerateAccessToken(user.ID, user.Username, user.Role, s.jwtIssuer, s.jwtSecret, s.accessTTL)
	if err != nil {
		return model.AuthResponse{}, nil, fmt.Errorf("membuat access token: %w", err)
	}

	rawRefreshToken, err := helper.RandomToken(32)
	if err != nil {
		return model.AuthResponse{}, nil, fmt.Errorf("membuat refresh token: %w", err)
	}

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

func (s *authService) RefreshToken(ctx context.Context, req model.RefreshTokenRequest) (model.AuthResponse, error) {
	if req.RefreshToken == "" {
		return model.AuthResponse{}, errors.New("refresh token wajib disertakan")
	}

	tokenHash := helper.SHA256Hex(req.RefreshToken)

	tokenEntity, err := s.tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return model.AuthResponse{}, errors.New("refresh token tidak valid")
	}

	if tokenEntity.RevokedAt != nil {
		return model.AuthResponse{}, ErrTokenRevoked
	}

	if time.Now().After(tokenEntity.ExpiresAt) {
		return model.AuthResponse{}, ErrRefreshTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, tokenEntity.UserID)
	if err != nil {
		return model.AuthResponse{}, errors.New("user pemilik token tidak ditemukan")
	}

	if err := s.tokenRepo.Revoke(ctx, tokenHash); err != nil {
		return model.AuthResponse{}, fmt.Errorf("gagal mencabut token lama: %w", err)
	}

	newAccessToken, err := helper.GenerateAccessToken(user.ID, user.Username, user.Role, s.jwtIssuer, s.jwtSecret, s.accessTTL)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("membuat access token baru: %w", err)
	}

	newRawRefreshToken, err := helper.RandomToken(32)
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("membuat refresh token baru: %w", err)
	}

	newTokenHash := helper.SHA256Hex(newRawRefreshToken)
	newRefreshTokenEntity := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if _, err := s.tokenRepo.Create(ctx, newRefreshTokenEntity); err != nil {
		return model.AuthResponse{}, fmt.Errorf("menyimpan refresh token baru: %w", err)
	}

	return model.AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	tokenHash := helper.SHA256Hex(refreshToken)
	_ = s.tokenRepo.Revoke(ctx, tokenHash)
	return nil
}

func (s *authService) Me(ctx context.Context, userID int) (model.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return model.UserResponse{}, errors.New("user tidak ditemukan")
	}

	return model.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}
