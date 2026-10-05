package helper

import (
	"strings"
	"sync"
)

// PermissionSet menyimpan seluruh pemetaan role -> permissions di dalam memori.
// Dirancang thread-safe untuk pembacaan bersama (read-concurrent) berkecepatan tinggi O(1).
type PermissionSet struct {
	mu          sync.RWMutex
	permissions map[string]map[string]bool
}

// NewPermissionSet membuat instansiasi PermissionSet baru dari mapping raw role -> []permission.
func NewPermissionSet(rawMapping map[string][]string) *PermissionSet {
	ps := &PermissionSet{
		permissions: make(map[string]map[string]bool),
	}

	for role, perms := range rawMapping {
		normalizedRole := strings.ToLower(strings.TrimSpace(role))
		if ps.permissions[normalizedRole] == nil {
			ps.permissions[normalizedRole] = make(map[string]bool)
		}
		for _, perm := range perms {
			normalizedPerm := strings.ToLower(strings.TrimSpace(perm))
			if normalizedPerm != "" {
				ps.permissions[normalizedRole][normalizedPerm] = true
			}
		}
	}

	return ps
}

// Can memeriksa apakah suatu role memiliki hak akses terhadap permission tertentu.
//
// PRINSIP FAIL-CLOSED:
// Fungsi ini SELALU mengembalikan false jika ada keraguan:
// 1. ps == nil (PermissionSet belum diinisialisasi)
// 2. role kosong ("") atau permission kosong ("")
// 3. role tidak terdaftar di basis data
// 4. permission tidak terdaftar untuk role tersebut
//
// Tidak ada satu pun cabang kode yang mengembalikan true secara kebetulan atau default!
func (ps *PermissionSet) Can(role, permission string) bool {
	if ps == nil {
		return false
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	normalizedPerm := strings.ToLower(strings.TrimSpace(permission))

	if normalizedRole == "" || normalizedPerm == "" {
		return false
	}

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	rolePerms, exists := ps.permissions[normalizedRole]
	if !exists || rolePerms == nil {
		return false
	}

	return rolePerms[normalizedPerm]
}

// PermissionsForRole mengembalikan seluruh permission yang dimiliki oleh suatu role (read-only slice).
func (ps *PermissionSet) PermissionsForRole(role string) []string {
	if ps == nil {
		return nil
	}

	normalizedRole := strings.ToLower(strings.TrimSpace(role))
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	rolePerms, exists := ps.permissions[normalizedRole]
	if !exists {
		return nil
	}

	perms := make([]string, 0, len(rolePerms))
	for p := range rolePerms {
		perms = append(perms, p)
	}
	return perms
}
