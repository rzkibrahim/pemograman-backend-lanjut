package service

import (
	"strings"

	"modul6/app/model"
)

// commonPasswords memuat daftar kata sandi lemah dan terlalu umum yang ditolak sistem.
var commonPasswords = map[string]bool{
	"password":    true,
	"password123": true,
	"12345678":    true,
	"123456789":   true,
	"admin123":    true,
	"qwerty123":   true,
	"indonesia":   true,
	"rahasia":     true,
	"administrator": true,
}

// ValidatePasswordStrength memeriksa kekuatan password secara murni.
// Mengembalikan pesan kesalahan jika tidak memenuhi syarat, atau string kosong jika valid.
func ValidatePasswordStrength(password string) string {
	if len(password) < 8 {
		return "minimal 8 karakter"
	}
	if commonPasswords[strings.ToLower(strings.TrimSpace(password))] {
		return "password terlalu umum"
	}
	return ""
}

// ValidateRegister memeriksa validitas payload pendaftaran akun baru.
func ValidateRegister(req model.RegisterRequest) map[string]string {
	errs := make(map[string]string)

	username := strings.TrimSpace(req.Username)
	if username == "" {
		errs["username"] = "wajib diisi"
	} else if len(username) < 3 {
		errs["username"] = "minimal 3 karakter"
	}

	email := strings.TrimSpace(req.Email)
	if email == "" {
		errs["email"] = "wajib diisi"
	} else if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		errs["email"] = "format email tidak valid"
	}

	if pwdErr := ValidatePasswordStrength(req.Password); pwdErr != "" {
		errs["password"] = pwdErr
	}

	return errs
}

// ValidateLogin memeriksa kelengkapan payload login.
func ValidateLogin(req model.LoginRequest) map[string]string {
	errs := make(map[string]string)

	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "wajib diisi"
	}
	if strings.TrimSpace(req.Password) == "" {
		errs["password"] = "wajib diisi"
	}

	return errs
}
