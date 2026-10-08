// Package middleware - ratelimit.go
// Mengelola rate limiting untuk percobaan login yang gagal:
// Jika suatu IP gagal login lebih dari 5 kali dalam kurun waktu 1 menit,
// sistem akan menolak permintaan dengan status HTTP 429 Too Many Requests.
package middleware

import (
	"sync"
	"time"

	"siakad-mini/helper"
)

// LoginFailureTracker menyimpan riwayat waktu kegagalan login per alamat IP
type LoginFailureTracker struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

var loginTracker = &LoginFailureTracker{
	failures: make(map[string][]time.Time),
}

const (
	MaxLoginFailures = 5
	WindowDuration   = 1 * time.Minute
)

// CheckLoginRateLimit memeriksa apakah IP telah melebihi batas 5 kali gagal login per menit
func CheckLoginRateLimit(ip string) error {
	loginTracker.mu.Lock()
	defer loginTracker.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-WindowDuration)

	// Filter hanya kegagalan dalam 1 menit terakhir
	recent := make([]time.Time, 0)
	for _, t := range loginTracker.failures[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	loginTracker.failures[ip] = recent

	if len(recent) >= MaxLoginFailures {
		return helper.TooManyRequests("terlalu banyak percobaan login yang gagal (lebih dari 5 kali per menit). Silakan coba lagi nanti.")
	}

	return nil
}

// RecordLoginFailure mencatat satu kali kegagalan login untuk IP tertentu
func RecordLoginFailure(ip string) {
	loginTracker.mu.Lock()
	defer loginTracker.mu.Unlock()

	loginTracker.failures[ip] = append(loginTracker.failures[ip], time.Now())
}

// ResetLoginFailure menghapus riwayat kegagalan setelah login berhasil
func ResetLoginFailure(ip string) {
	loginTracker.mu.Lock()
	defer loginTracker.mu.Unlock()

	delete(loginTracker.failures, ip)
}
