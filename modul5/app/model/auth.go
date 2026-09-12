package model

import "time"

// RegisterRequest menampung data pendaftaran pengguna baru.
// PENTING: Field "role" SENGAJA TIDAK ADA di sini untuk menutup celah Mass Assignment.
// Nilai role selalu ditetapkan oleh server secara sepihak ("user").
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest menampung kredensial untuk masuk sistem.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RefreshTokenRequest menampung refresh token untuk memperoleh access token baru.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthResponse adalah payload respons sukses setelah login atau refresh token.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // dalam detik (misal 900 untuk 15 menit)
}

// UserResponse adalah representasi data profil pengguna tanpa field password.
type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshToken merepresentasikan entitas baris tabel refresh_tokens di basis data.
type RefreshToken struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}
