package service

import (
	"testing"

	"modul6/app/model"
	"modul6/helper"
)

func setupTestPermissionSet() *helper.PermissionSet {
	return helper.NewPermissionSet(map[string][]string{
		"admin": {
			"user:list", "user:read:any", "user:update:any", "user:delete", "role:assign",
			"student:list", "student:read:any", "student:create", "student:update:any", "student:delete",
		},
		"staff": {
			"user:list", "user:read:any",
			"student:list", "student:read:any", "student:create",
		},
		"user": {},
	})
}

func TestCanAccessStudent(t *testing.T) {
	perms := setupTestPermissionSet()

	tests := []struct {
		name          string
		current       model.AuthUser
		ownerID       int
		perms         *helper.PermissionSet
		anyPerm       string
		expectedAllow bool
	}{
		{
			name:          "Pemilik data sendiri (role user) boleh akses tanpa permission :any",
			current:       model.AuthUser{ID: 10, Username: "andi", Role: "user"},
			ownerID:       10,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: true,
		},
		{
			name:          "Pemilik data sendiri (role user) boleh update tanpa permission student:update:any",
			current:       model.AuthUser{ID: 10, Username: "andi", Role: "user"},
			ownerID:       10,
			perms:         perms,
			anyPerm:       "student:update:any",
			expectedAllow: true,
		},
		{
			name:          "Bukan pemilik (role user) DITOLAK saat membaca data orang lain",
			current:       model.AuthUser{ID: 10, Username: "andi", Role: "user"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: false,
		},
		{
			name:          "Bukan pemilik (role user) DITOLAK saat mengedit data orang lain",
			current:       model.AuthUser{ID: 10, Username: "andi", Role: "user"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:update:any",
			expectedAllow: false,
		},
		{
			name:          "Staff boleh membaca data mahasiswa milik orang lain (memiliki student:read:any)",
			current:       model.AuthUser{ID: 2, Username: "sari_staff", Role: "staff"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: true,
		},
		{
			name:          "Staff DITOLAK saat mengedit data mahasiswa milik orang lain (tidak memiliki student:update:any)",
			current:       model.AuthUser{ID: 2, Username: "sari_staff", Role: "staff"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:update:any",
			expectedAllow: false,
		},
		{
			name:          "Admin boleh membaca data mahasiswa mana pun (memiliki student:read:any)",
			current:       model.AuthUser{ID: 1, Username: "admin_budi", Role: "admin"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: true,
		},
		{
			name:          "Admin boleh mengubah data mahasiswa mana pun (memiliki student:update:any)",
			current:       model.AuthUser{ID: 1, Username: "admin_budi", Role: "admin"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:update:any",
			expectedAllow: true,
		},
		// Pengujian Fail-Closed:
		{
			name:          "Fail-closed: PermissionSet bernilai nil",
			current:       model.AuthUser{ID: 1, Username: "admin_budi", Role: "admin"},
			ownerID:       20,
			perms:         nil,
			anyPerm:       "student:read:any",
			expectedAllow: false,
		},
		{
			name:          "Fail-closed: current.ID tidak valid (<= 0)",
			current:       model.AuthUser{ID: 0, Username: "invalid", Role: "admin"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: false,
		},
		{
			name:          "Fail-closed: ownerID tidak valid (<= 0)",
			current:       model.AuthUser{ID: 1, Username: "admin", Role: "admin"},
			ownerID:       0,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: false,
		},
		{
			name:          "Fail-closed: permission string kosong",
			current:       model.AuthUser{ID: 1, Username: "admin", Role: "admin"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "",
			expectedAllow: false,
		},
		{
			name:          "Fail-closed: role tidak dikenal",
			current:       model.AuthUser{ID: 99, Username: "unknown", Role: "guest"},
			ownerID:       20,
			perms:         perms,
			anyPerm:       "student:read:any",
			expectedAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanAccessStudent(tt.current, tt.ownerID, tt.perms, tt.anyPerm)
			if got != tt.expectedAllow {
				t.Errorf("CanAccessStudent() = %v, expected %v", got, tt.expectedAllow)
			}
		})
	}
}
