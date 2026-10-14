// Package config berisi fungsi-fungsi konfigurasi aplikasi SIAKAD Mini.
// File ini menangani pembacaan variabel environment dari berkas .env.
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv memuat variabel dari berkas .env ke environment proses.
// Jika file .env tidak ditemukan, aplikasi tetap berjalan menggunakan
// variabel environment sistem operasi.
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("peringatan: berkas .env tidak ditemukan, memakai environment sistem")
	}
}

// GetEnv mengambil nilai environment variable berdasarkan key.
// Bila key tidak ada atau kosong, kembalikan nilai fallback.
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// GetEnvInt sama seperti GetEnv, tetapi mengembalikan hasil berupa int.
// Jika konversi gagal, nilai fallback yang dikembalikan.
func GetEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("peringatan: %s bukan angka (%q), memakai bawaan %d", key, value, fallback)
		return fallback
	}
	return parsed
}
