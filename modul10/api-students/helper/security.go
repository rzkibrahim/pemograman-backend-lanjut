package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost adalah faktor beban komputasi untuk hashing password.
// Nilai 12 berarti 2^12 = 4096 iterasi, dirancang cukup lambat untuk
// memitigasi serangan brute force dan rainbow table.
const BcryptCost = 12

// dummyHash adalah hash bcrypt valid yang digunakan saat username tidak ditemukan.
// Tujuannya agar waktu respon verifikasi identik dengan kasus username ditemukan,
// sehingga menutup celah timing attack (User Enumeration).
const dummyHash = "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewJPHyDVSIQBLbPy"

// HashPassword mengenkripsi plaintext password menggunakan algoritma bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("gagal melakukan hash password: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword memvalidasi kecocokan plaintext password dengan hash tersimpan.
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// VerifyDummyPassword mengeksekusi perbandingan hash bcrypt dengan beban komputasi sama
// ketika user tidak ditemukan di database untuk mencegah perbedaan latensi (timing attack).
func VerifyDummyPassword() {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummy-password-prevent-enumeration"))
}

// RandomToken menghasilkan string acak berkekuatan kriptografi sebanyak n byte (output 2*n hex char).
func RandomToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("gagal menghasilkan token acak: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SHA256Hex menghasilkan representasi heksadesimal dari hash SHA-256 sebuah string.
// Digunakan untuk menyimpan refresh token di basis data tanpa perlu bcrypt
// karena string token acak sudah memiliki entropi sangat tinggi.
func SHA256Hex(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
