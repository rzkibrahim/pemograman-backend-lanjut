// Package model - user.go
// Mendefinisikan entitas user, request DTO login, dan response DTO autentikasi.
package model

import "time"

// Konstanta role pengguna dalam sistem SIAKAD Mini
const (
	RoleAdmin     = "admin"
	RoleMahasiswa = "mahasiswa"
)

// User merepresentasikan record pada tabel `users`.
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Tidak boleh diserialisasi ke JSON
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserDataResponse adalah representasi data ringkas user pada response JSON.
type UserDataResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// LoginRequest adalah DTO request body untuk endpoint POST /api/v1/auth/login.
// Menggunakan validasi deklaratif tag go-playground/validator.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginResponse adalah DTO response sukses login sesuai spesifikasi UTS.
type LoginResponse struct {
	AccessToken string           `json:"access_token"`
	TokenType   string           `json:"token_type"`
	ExpiresIn   int              `json:"expires_in"` // Durasi dalam detik
	User        UserDataResponse `json:"user"`
}

// AuthMeResponse adalah DTO untuk endpoint GET /api/v1/auth/me.
// Jika role adalah mahasiswa, field Student akan menyertakan data profil mahasiswa.
type AuthMeResponse struct {
	ID      int                    `json:"id"`
	Email   string                 `json:"email"`
	Role    string                 `json:"role"`
	Student *StudentProfileSummary `json:"student,omitempty"`
}

// StudentProfileSummary memuat profil singkat mahasiswa untuk disisipkan pada GET /auth/me.
type StudentProfileSummary struct {
	ID          int     `json:"id"`
	NIM         string  `json:"nim"`
	Nama        string  `json:"nama"`
	Prodi       string  `json:"prodi"`
	Angkatan    int     `json:"angkatan"`
	IPKTerakhir float64 `json:"ipk_terakhir"`
}
