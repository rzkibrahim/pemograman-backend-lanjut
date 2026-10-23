package service

import (
	"modul9/app/model"
	"modul9/helper"
)

// CanAccessStudent mengevaluasi hak akses pengguna terhadap suatu baris data mahasiswa.
//
// Aturan Akses (Ownership Check & Pure Rule):
// 1. Pemilik data (current.ID == ownerID) SELALU diizinkan melihat/mengubah datanya sendiri
//    tanpa memerlukan permission :any khusus.
// 2. Bukan pemilik data: WAJIB memiliki permission :any yang sesuai pada role-nya
//    (contoh: "student:read:any" untuk GET, "student:update:any" untuk PUT/PATCH).
//
// PRINSIP FAIL-CLOSED:
// - Jika perms == nil, fungsi menolak akses (false).
// - Jika ID user <= 0 atau ownerID <= 0, fungsi menolak akses (false).
// - Jika role tidak memiliki permission :any, fungsi menolak akses (false).
//
// PENTING: Fungsi ini murni (pure function) dan TIDAK BOLEH mengimpor package Fiber
// ataupun package Repository (memenuhi kaidah arsitektur bersih).
func CanAccessStudent(current model.AuthUser, ownerID int, perms *helper.PermissionSet, anyPermission string) bool {
	// Fail-closed validasi identitas dan ownership
	if current.ID <= 0 || ownerID <= 0 {
		return false
	}

	// 1. Ownership Check: Pemilik data selalu memiliki hak akses penuh terhadap datanya
	if current.ID == ownerID {
		return true
	}

	// 2. Untuk mengakses data orang lain, harus memiliki permission khusus (:any)
	if perms == nil || anyPermission == "" {
		return false
	}

	return perms.Can(current.Role, anyPermission)
}
