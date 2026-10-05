package service

import (
	"testing"

	"modul6/app/model"
)

func TestCanAccessUser(t *testing.T) {
	perms := setupTestPermissionSet()

	tests := []struct {
		name          string
		current       model.AuthUser
		targetUserID  int
		anyPerm       string
		expectedAllow bool
	}{
		{
			name:          "User mengakses profil sendiri selalu diizinkan",
			current:       model.AuthUser{ID: 5, Username: "andi", Role: "user"},
			targetUserID:  5,
			anyPerm:       "user:read:any",
			expectedAllow: true,
		},
		{
			name:          "User mengakses profil orang lain ditolak",
			current:       model.AuthUser{ID: 5, Username: "andi", Role: "user"},
			targetUserID:  8,
			anyPerm:       "user:read:any",
			expectedAllow: false,
		},
		{
			name:          "Staff membaca profil orang lain diizinkan (user:read:any)",
			current:       model.AuthUser{ID: 2, Username: "sari", Role: "staff"},
			targetUserID:  8,
			anyPerm:       "user:read:any",
			expectedAllow: true,
		},
		{
			name:          "Staff mengubah profil orang lain ditolak (tidak punya user:update:any)",
			current:       model.AuthUser{ID: 2, Username: "sari", Role: "staff"},
			targetUserID:  8,
			anyPerm:       "user:update:any",
			expectedAllow: false,
		},
		{
			name:          "Admin mengubah profil orang lain diizinkan (user:update:any)",
			current:       model.AuthUser{ID: 1, Username: "budi", Role: "admin"},
			targetUserID:  8,
			anyPerm:       "user:update:any",
			expectedAllow: true,
		},
		// Fail-closed
		{
			name:          "Fail-closed: ID tidak valid",
			current:       model.AuthUser{ID: 0, Username: "anon", Role: "admin"},
			targetUserID:  8,
			anyPerm:       "user:read:any",
			expectedAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanAccessUser(tt.current, tt.targetUserID, perms, tt.anyPerm)
			if got != tt.expectedAllow {
				t.Errorf("CanAccessUser() = %v, expected %v", got, tt.expectedAllow)
			}
		})
	}
}
