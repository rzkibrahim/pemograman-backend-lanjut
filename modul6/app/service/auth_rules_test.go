package service

import (
	"testing"

	"modul6/app/model"
)

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{
			name:     "password valid kuat",
			password: "SuperSecretPassword#2026",
			wantErr:  "",
		},
		{
			name:     "password terlalu pendek (< 8)",
			password: "short",
			wantErr:  "minimal 8 karakter",
		},
		{
			name:     "password terlalu umum (password123)",
			password: "password123",
			wantErr:  "password terlalu umum",
		},
		{
			name:     "password terlalu umum (admin123)",
			password: "admin123",
			wantErr:  "password terlalu umum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePasswordStrength(tt.password)
			if got != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestValidateRegister(t *testing.T) {
	tests := []struct {
		name       string
		req        model.RegisterRequest
		expectErrs []string
	}{
		{
			name: "register valid",
			req: model.RegisterRequest{
				Username: "sari_rahma",
				Email:    "sari@example.com",
				Password: "SariSecurePass2026!",
			},
			expectErrs: []string{},
		},
		{
			name: "username kosong dan email rusak",
			req: model.RegisterRequest{
				Username: "",
				Email:    "bukan-email",
				Password: "ValidPassword123!",
			},
			expectErrs: []string{"username", "email"},
		},
		{
			name: "password terlalu umum",
			req: model.RegisterRequest{
				Username: "budi_santoso",
				Email:    "budi@example.com",
				Password: "password123",
			},
			expectErrs: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateRegister(tt.req)
			for _, field := range tt.expectErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("ValidateRegister() expected error on field %v, got none", field)
				}
			}
			if len(tt.expectErrs) == 0 && len(errs) > 0 {
				t.Errorf("ValidateRegister() expected no errors, got %v", errs)
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name       string
		req        model.LoginRequest
		expectErrs []string
	}{
		{
			name: "login valid",
			req: model.LoginRequest{
				Username: "sari",
				Password: "passwordRahasia123!",
			},
			expectErrs: []string{},
		},
		{
			name: "username dan password kosong",
			req: model.LoginRequest{
				Username: "",
				Password: "",
			},
			expectErrs: []string{"username", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateLogin(tt.req)
			for _, field := range tt.expectErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("ValidateLogin() expected error on field %v, got none", field)
				}
			}
		})
	}
}
