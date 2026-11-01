package helper

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenExpired menandakan access token sudah melewati batas kedaluwarsa (exp).
	ErrTokenExpired = errors.New("access token telah kedaluwarsa")
	// ErrTokenInvalid menandakan signature salah, klaim rusak, atau algoritma tidak sah.
	ErrTokenInvalid = errors.New("access token tidak valid")
)

// JWTPayload merepresentasikan payload klaim kustom pada Access Token JWT.
type JWTPayload struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken membuat JWT access token bertanda tangan HMAC-SHA256 (HS256).
func GenerateAccessToken(userID int, username, role, issuer, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTPayload{
		UserID:   userID,
		Username: username,
		Role:     role,
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

// ParseAccessToken memvalidasi token JWT, signature, masa berlaku, dan algoritma penandatanganan.
func ParseAccessToken(tokenString, secret string) (*JWTPayload, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTPayload{}, func(t *jwt.Token) (any, error) {
		// PENCEGAHAN ALGORITHM CONFUSION (misal alg: "none" atau public key as HMAC):
		// Pastikan algoritma yang digunakan benar-benar turunan SigningMethodHMAC.
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
