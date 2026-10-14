// Package helper - jwt.go
// Mengelola pembuatan dan verifikasi JSON Web Token (JWT) untuk autentikasi.
// Menggunakan algoritma HMAC-SHA256 (HS256) untuk penandatanganan token.
package helper

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenExpired menandakan access token sudah melewati batas kedaluwarsa.
	ErrTokenExpired = errors.New("access token telah kedaluwarsa")
	// ErrTokenInvalid menandakan signature salah atau klaim rusak.
	ErrTokenInvalid = errors.New("access token tidak valid")
)

// JWTPayload merepresentasikan payload klaim kustom pada JWT.
// Berisi informasi identitas pengguna yang terautentikasi.
type JWTPayload struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken membuat JWT access token bertanda tangan HMAC-SHA256.
// Token berisi user_id, email, role, dan masa berlaku (TTL).
func GenerateAccessToken(userID int, email, role, issuer, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTPayload{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("gagal menandatangani token: %w", err)
	}

	return tokenString, nil
}

// ParseAccessToken memvalidasi dan mem-parse token JWT.
// Melakukan pengecekan signature, algoritma, dan masa berlaku.
func ParseAccessToken(tokenString, secret string) (*JWTPayload, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTPayload{}, func(t *jwt.Token) (any, error) {
		// Pencegahan Algorithm Confusion Attack:
		// Pastikan algoritma benar-benar HMAC, bukan "none" atau RSA
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: algoritma %v tidak didukung", ErrTokenInvalid, t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*JWTPayload)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
