// Package helper - security.go
// Berisi fungsi-fungsi keamanan: hashing password dengan bcrypt,
// verifikasi password, dan mitigasi timing attack.
package helper

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost adalah faktor beban komputasi untuk hashing password.
// Nilai 12 berarti 2^12 = 4096 iterasi untuk memitigasi brute force.
const BcryptCost = 12

// dummyHash digunakan saat email tidak ditemukan di database.
// Tujuannya agar waktu respon identik (mencegah timing attack / user enumeration).
const dummyHash = "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewJPHyDVSIQBLbPy"

// HashPassword mengenkripsi plaintext password menggunakan bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("gagal melakukan hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword memvalidasi kecocokan plaintext dengan hash tersimpan.
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// VerifyDummyPassword mengeksekusi perbandingan hash bcrypt ketika user
// tidak ditemukan untuk mencegah timing attack (user enumeration).
func VerifyDummyPassword() {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummy-password-prevent-enumeration"))
}
