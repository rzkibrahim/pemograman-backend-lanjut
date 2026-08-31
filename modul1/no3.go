package main

import "fmt"

// ============================
// Soal No3 Latihan Mandiri: Fungsi Pointer (dari folder tugas-mandiri/3-pointer)
// ============================

// swap menukar nilai dua integer melalui pointer
func swap(a, b *int) {
	*a, *b = *b, *a
}

// updateSlice menambahkan item baru ke slice.
// Perlu pointer ke slice karena append bisa mengembalikan header slice
// baru (alamat memori berbeda) saat kapasitas lama tidak cukup — tanpa
// pointer, perubahan itu hanya terjadi pada salinan lokal di dalam function.
func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

// --- Pembanding: pass by value vs pass by pointer ---

func naikkanByValue(x int) {
	x = x + 1 // hanya mengubah salinan lokal
}

func naikkanByPointer(x *int) {
	*x = *x + 1 // mengubah data asli lewat alamatnya
}

// RunPointerDemo menjalankan demo pointer dari BAGIAN 2
func RunPointerDemo() {
	// ========================================
	// BAGIAN 2: Pointer (dari folder tugas-mandiri/3-pointer)
	// ========================================

	// --- Bukti swap ---
	fmt.Println("\n=== swap(a, b *int) ===")
	a, b := 10, 99
	fmt.Printf("Sebelum : a=%d, b=%d\n", a, b)
	swap(&a, &b)
	fmt.Printf("Sesudah : a=%d, b=%d\n", a, b)

	// --- Bukti updateSlice ---
	fmt.Println("\n=== updateSlice(s *[]string, item string) ===")
	daftar := []string{"Go", "Fiber"}
	fmt.Println("Sebelum :", daftar)
	updateSlice(&daftar, "PostgreSQL")
	fmt.Println("Sesudah :", daftar)

	// --- Perbandingan pass by value vs pass by pointer ---
	fmt.Println("\n=== Pass by Value vs Pass by Pointer ===")
	angka := 5
	fmt.Printf("Nilai awal: %d\n", angka)

	naikkanByValue(angka)
	fmt.Printf("Setelah naikkanByValue(angka)   -> tetap: %d\n", angka)

	naikkanByPointer(&angka)
	fmt.Printf("Setelah naikkanByPointer(&angka) -> berubah: %d\n", angka)
}