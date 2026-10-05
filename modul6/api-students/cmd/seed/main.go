package main

import (
	"context"
	"fmt"
	"log"

	"modul6/database"
	"modul6/helper"
)

func main() {
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("koneksi database gagal: %v", err)
	}
	defer pool.Close()

	password := "Password123!"
	hashed, err := helper.HashPassword(password)
	if err != nil {
		log.Fatalf("hash password gagal: %v", err)
	}

	users := []struct {
		username string
		email    string
		role     string
	}{
		{"admin_budi", "admin@univ.ac.id", "admin"},
		{"staff_sari", "staff@univ.ac.id", "staff"},
		{"user_andi", "andi@student.ac.id", "user"},
		{"user_bambang", "bambang@student.ac.id", "user"},
	}

	for _, u := range users {
		query := `
			INSERT INTO users (username, email, password, role, is_active)
			VALUES ($1, $2, $3, $4, true)
			ON CONFLICT (LOWER(username)) DO UPDATE
			SET password = EXCLUDED.password, role = EXCLUDED.role, is_active = true
		`
		_, err := pool.Exec(context.Background(), query, u.username, u.email, hashed, u.role)
		if err != nil {
			log.Fatalf("gagal insert/update user %s: %v", u.username, err)
		}
		fmt.Printf("User %s (%s) siap dengan password valid.\n", u.username, u.role)
	}
}
