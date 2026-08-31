package main

import "fmt"

// RunVariabelDemo menjalankan demo variabel & map dari BAGIAN 1
func RunVariabelDemo() {

	// ========================================
	// Soal no2 Latihan Mandiri: Variabel & Map (dari folder tugas-mandiri/2-variabel)
	// ========================================

	// --- Deklarasi lima variabel dengan tipe berbeda ---
	var nama string = "Rizki"
	var umur int = 20
	var ipk float64 = 3.55
	var isMahasiswaAktif bool = true
	var matkul []string = []string{"Backend Lanjut,", "Machine Learning,", "Cybersecurity"}

	fmt.Println("=== Variabel Dasar ===")
	fmt.Printf("nama (string)       : %s\n", nama)
	fmt.Printf("umur (int)          : %d\n", umur)
	fmt.Printf("ipk (float64)       : %.2f\n", ipk)
	fmt.Printf("aktif (bool)        : %v\n", isMahasiswaAktif)
	fmt.Printf("matkul (slice)      : %v\n", matkul)

	// --- Map untuk menyimpan data mahasiswa: nama -> nilai ---
	fmt.Println("\n=== Operasi Map Mahasiswa ===")
	nilaiMahasiswa := make(map[string]float64)

	// 1. Menambah
	nilaiMahasiswa["Rizki"] = 88.5
	nilaiMahasiswa["Sari"] = 92.0
	nilaiMahasiswa["Budi"] = 76.25
	fmt.Println("Setelah menambah 3 data:", nilaiMahasiswa)

	// 2. Membaca dengan pengecekan keberadaan (pola dua nilai)
	if nilai, ada := nilaiMahasiswa["Sari"]; ada {
		fmt.Printf("Nilai Sari ditemukan: %.2f\n", nilai)
	} else {
		fmt.Println("Sari belum punya nilai")
	}
	if nilai, ada := nilaiMahasiswa["Dedi"]; ada {
		fmt.Printf("Nilai Dedi ditemukan: %.2f\n", nilai)
	} else {
		fmt.Println("Dedi belum punya nilai (key tidak ada di map)")
	}

	// 3. Menghapus
	delete(nilaiMahasiswa, "Budi")
	fmt.Println("Setelah menghapus 'Budi':", nilaiMahasiswa)

	// 4. Menelusuri seluruh isi map
	fmt.Println("Menelusuri seluruh isi map:")
	for nama, nilai := range nilaiMahasiswa {
		fmt.Printf("  - %s: %.2f\n", nama, nilai)
	}
}