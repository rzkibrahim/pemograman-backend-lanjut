package service

import (
	"modul10/app/model"
	"modul10/helper"
)

// CanAccessUser mengevaluasi hak akses pemanggil terhadap data profil seorang user.
//
// Aturan Akses (Clean Architecture Pure Rule):
// 1. Pemilik data (current.ID == targetUserID) SELALU diizinkan mengelola profilnya sendiri.
// 2. Jika bukan pemilik data, pemanggil harus memiliki permission yang disyaratkan (misal: "user:read:any" atau "user:update:any").
//
// PRINSIP FAIL-CLOSED:
// Jika perms nil, ID tidak valid, atau permission tidak terpenuhi, fungsi SELALU mengembalikan false.
// Fungsi ini murni (pure function): TIDAK BOLEH mengimpor Fiber maupun lapisan Repository.
func CanAccessUser(current model.AuthUser, targetUserID int, perms *helper.PermissionSet, anyPermission string) bool {
	// Fail-closed validasi identitas
	if current.ID <= 0 || targetUserID <= 0 {
		return false
	}

	// 1. Ownership Check: Pemilik data selalu berhak atas datanya sendiri
	if current.ID == targetUserID {
		return true
	}

	// 2. Pemeriksaan Permission untuk data orang lain (:any)
	if perms == nil || anyPermission == "" {
		return false
	}

	return perms.Can(current.Role, anyPermission)
}
