package model

import "time"

// User merepresentasikan entitas pengguna dalam sistem.
// Tag json:"-" pada field Password menjamin password hash tidak pernah
// ikut terkirim dalam serialisasi respons JSON (mencegah kebocoran kredensial).
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateUserRequest menampung data perubahan profil pengguna.
type UpdateUserRequest struct {
	Email string `json:"email"`
}

// AssignRoleRequest menampung role baru yang akan diberikan kepada pengguna.
type AssignRoleRequest struct {
	Role string `json:"role"`
}
