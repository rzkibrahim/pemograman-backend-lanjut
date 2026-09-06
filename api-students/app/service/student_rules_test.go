package service

// File ini berisi unit test untuk business rules murni di student_rules.go.
// Pengujian ini membuktikan bahwa validasi, patch, dan penghitungan halaman
// berjalan benar tanpa memerlukan koneksi database atau HTTP server.

import (
	"testing"

	"api-students/app/model"
)

// =============================================================================
// TestValidateCreate — menguji validasi pembuatan student baru (POST).
//
// Skenario yang diuji:
//   - Permintaan valid → peta error harus kosong.
//   - NIM kosong → harus menghasilkan pesan error pada field "nim".
//   - Name kosong → harus menghasilkan pesan error pada field "name".
//   - Grade di luar rentang (negatif/di atas 100) → error pada field "grade".
//   - Seluruh field kosong → semua field bermasalah sekaligus.
// =============================================================================

func TestValidateCreate(t *testing.T) {
	// Tabel sub-test agar mudah menambah kasus baru di kemudian hari.
	tests := []struct {
		name     string                   // deskripsi singkat kasus uji
		req      model.CreateStudentRequest // data masukan yang diuji
		wantErrs []string                 // field-field yang diharapkan bermasalah
	}{
		{
			name:     "valid — seluruh field terisi dengan benar",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "Budi", Grade: 85},
			wantErrs: nil, // tidak ada error yang diharapkan
		},
		{
			name:     "nim kosong → wajib diisi",
			req:      model.CreateStudentRequest{NIM: "", Name: "Budi", Grade: 80},
			wantErrs: []string{"nim"},
		},
		{
			name:     "nim hanya spasi → tetap dianggap kosong",
			req:      model.CreateStudentRequest{NIM: "   ", Name: "Budi", Grade: 80},
			wantErrs: []string{"nim"},
		},
		{
			name:     "name kosong → wajib diisi",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "", Grade: 80},
			wantErrs: []string{"name"},
		},
		{
			name:     "grade negatif → di luar rentang 0–100",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "Budi", Grade: -1},
			wantErrs: []string{"grade"},
		},
		{
			name:     "grade di atas 100 → di luar rentang 0–100",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "Budi", Grade: 101},
			wantErrs: []string{"grade"},
		},
		{
			name:     "grade tepat 0 → batas bawah valid",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "Budi", Grade: 0},
			wantErrs: nil,
		},
		{
			name:     "grade tepat 100 → batas atas valid",
			req:      model.CreateStudentRequest{NIM: "A12345", Name: "Budi", Grade: 100},
			wantErrs: nil,
		},
		{
			name:     "seluruh field kosong → semua field bermasalah",
			req:      model.CreateStudentRequest{NIM: "", Name: "", Grade: 0},
			wantErrs: []string{"nim", "name"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateCreate(tc.req)

			// Periksa apakah field-field yang diharapkan bermasalah memang ada di hasil.
			for _, field := range tc.wantErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("mengharapkan error pada field %q, tapi tidak ditemukan; errs = %v", field, errs)
				}
			}

			// Periksa apakah jumlah error sesuai harapan (tidak ada error tambahan).
			if len(errs) != len(tc.wantErrs) {
				t.Errorf("jumlah error = %d, ingin %d; errs = %v", len(errs), len(tc.wantErrs), errs)
			}
		})
	}
}

// =============================================================================
// TestValidateReplace — menguji validasi penggantian student secara penuh (PUT).
//
// PUT mengharuskan seluruh field terisi, sehingga validasinya mirip dengan
// ValidateCreate tetapi dengan pesan error yang berbeda ("wajib diisi pada PUT").
// =============================================================================

func TestValidateReplace(t *testing.T) {
	tests := []struct {
		name     string
		req      model.ReplaceStudentRequest
		wantErrs []string
	}{
		{
			name:     "valid — seluruh field terisi dengan benar",
			req:      model.ReplaceStudentRequest{NIM: "A12345", Name: "Budi", Grade: 85},
			wantErrs: nil,
		},
		{
			name:     "nim kosong → wajib diisi pada PUT",
			req:      model.ReplaceStudentRequest{NIM: "", Name: "Budi", Grade: 80},
			wantErrs: []string{"nim"},
		},
		{
			name:     "name kosong → wajib diisi pada PUT",
			req:      model.ReplaceStudentRequest{NIM: "A12345", Name: "", Grade: 80},
			wantErrs: []string{"name"},
		},
		{
			name:     "grade negatif → di luar rentang 0–100",
			req:      model.ReplaceStudentRequest{NIM: "A12345", Name: "Budi", Grade: -5},
			wantErrs: []string{"grade"},
		},
		{
			name:     "grade di atas 100 → di luar rentang 0–100",
			req:      model.ReplaceStudentRequest{NIM: "A12345", Name: "Budi", Grade: 150},
			wantErrs: []string{"grade"},
		},
		{
			name:     "seluruh field kosong → semua field bermasalah",
			req:      model.ReplaceStudentRequest{NIM: "", Name: "", Grade: -1},
			wantErrs: []string{"nim", "name", "grade"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateReplace(tc.req)

			for _, field := range tc.wantErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("mengharapkan error pada field %q, tapi tidak ditemukan; errs = %v", field, errs)
				}
			}

			if len(errs) != len(tc.wantErrs) {
				t.Errorf("jumlah error = %d, ingin %d; errs = %v", len(errs), len(tc.wantErrs), errs)
			}
		})
	}
}

// =============================================================================
// TestApplyPatch — menguji penerapan perubahan parsial (PATCH) pada data student.
//
// ApplyPatch hanya mengubah field yang pointer-nya bukan nil.
// Field nil dibiarkan tetap menggunakan nilai asli (current).
// Skenario yang diuji:
//   - Hanya NIM yang diubah → field lain tetap sama.
//   - Hanya Name yang diubah → field lain tetap sama.
//   - Hanya Grade yang diubah → field lain tetap sama.
//   - Hanya IsActive yang diubah → field lain tetap sama.
//   - Beberapa field sekaligus.
//   - NIM kosong via pointer → menghasilkan error validasi.
//   - Grade di luar rentang via pointer → menghasilkan error validasi.
// =============================================================================

func TestApplyPatch(t *testing.T) {
	// ptrStr dan ptrFloat64 adalah helper agar bisa menulis literal pointer inline.
	ptrStr := func(s string) *string { return &s }
	ptrFloat64 := func(f float64) *float64 { return &f }
	ptrBool := func(b bool) *bool { return &b }

	// Data awal yang akan dimodifikasi oleh patch.
	base := model.Student{
		ID:       1,
		NIM:      "A11111",
		Name:     "Awal",
		Grade:    70,
		IsActive: true,
	}

	tests := []struct {
		name       string
		req        model.PatchStudentRequest
		wantNIM    string   // NIM yang diharapkan setelah patch
		wantName   string   // Name yang diharapkan setelah patch
		wantGrade  float64  // Grade yang diharapkan setelah patch
		wantActive bool     // IsActive yang diharapkan setelah patch
		wantErrs   []string // field-field yang diharapkan bermasalah
	}{
		{
			name:       "ubah NIM saja → field lain tetap",
			req:        model.PatchStudentRequest{NIM: ptrStr("B22222")},
			wantNIM:    "B22222",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   nil,
		},
		{
			name:       "ubah Name saja → field lain tetap",
			req:        model.PatchStudentRequest{Name: ptrStr("Baru")},
			wantNIM:    "A11111",
			wantName:   "Baru",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   nil,
		},
		{
			name:       "ubah Grade saja → field lain tetap",
			req:        model.PatchStudentRequest{Grade: ptrFloat64(95)},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  95,
			wantActive: true,
			wantErrs:   nil,
		},
		{
			name:       "ubah IsActive saja (false) → field lain tetap",
			req:        model.PatchStudentRequest{IsActive: ptrBool(false)},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: false,
			wantErrs:   nil,
		},
		{
			name: "ubah NIM dan Grade sekaligus → keduanya berubah",
			req:  model.PatchStudentRequest{NIM: ptrStr("C33333"), Grade: ptrFloat64(88)},
			wantNIM:    "C33333",
			wantName:   "Awal",
			wantGrade:  88,
			wantActive: true,
			wantErrs:   nil,
		},
		{
			name:       "NIM kosong via pointer → error validasi",
			req:        model.PatchStudentRequest{NIM: ptrStr("")},
			wantNIM:    "A11111", // tidak berubah karena gagal validasi
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   []string{"nim"},
		},
		{
			name:       "NIM hanya spasi → tetap dianggap kosong",
			req:        model.PatchStudentRequest{NIM: ptrStr("   ")},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   []string{"nim"},
		},
		{
			name:       "Name kosong via pointer → error validasi",
			req:        model.PatchStudentRequest{Name: ptrStr("")},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   []string{"name"},
		},
		{
			name:       "Grade negatif → error validasi",
			req:        model.PatchStudentRequest{Grade: ptrFloat64(-10)},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70, // tidak berubah
			wantActive: true,
			wantErrs:   []string{"grade"},
		},
		{
			name:       "Grade di atas 100 → error validasi",
			req:        model.PatchStudentRequest{Grade: ptrFloat64(200)},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   []string{"grade"},
		},
		{
			name:       "tidak ada field yang dikirim → tidak ada perubahan",
			req:        model.PatchStudentRequest{},
			wantNIM:    "A11111",
			wantName:   "Awal",
			wantGrade:  70,
			wantActive: true,
			wantErrs:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Gunakan salinan data awal agar setiap sub-test independen.
			current := base
			result, errs := ApplyPatch(current, tc.req)

			// Verifikasi error validasi yang diharapkan.
			for _, field := range tc.wantErrs {
				if _, ok := errs[field]; !ok {
					t.Errorf("mengharapkan error pada field %q, tapi tidak ditemukan; errs = %v", field, errs)
				}
			}
			if len(errs) != len(tc.wantErrs) {
				t.Errorf("jumlah error = %d, ingin %d; errs = %v", len(errs), len(tc.wantErrs), errs)
			}

			// Verifikasi nilai hasil patch.
			if result.NIM != tc.wantNIM {
				t.Errorf("NIM = %q, ingin %q", result.NIM, tc.wantNIM)
			}
			if result.Name != tc.wantName {
				t.Errorf("Name = %q, ingin %q", result.Name, tc.wantName)
			}
			if result.Grade != tc.wantGrade {
				t.Errorf("Grade = %v, ingin %v", result.Grade, tc.wantGrade)
			}
			if result.IsActive != tc.wantActive {
				t.Errorf("IsActive = %v, ingin %v", result.IsActive, tc.wantActive)
			}
		})
	}
}

// =============================================================================
// TestIsEmptyPatch — menguji deteksi permintaan PATCH yang tidak mengubah apa pun.
// =============================================================================

func TestIsEmptyPatch(t *testing.T) {
	ptrStr := func(s string) *string { return &s }

	tests := []struct {
		name string
		req  model.PatchStudentRequest
		want bool // true jika patch dianggap kosong
	}{
		{
			name: "seluruh field nil → patch kosong",
			req:  model.PatchStudentRequest{},
			want: true,
		},
		{
			name: "NIM terisi → patch tidak kosong",
			req:  model.PatchStudentRequest{NIM: ptrStr("X99999")},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsEmptyPatch(tc.req)
			if got != tc.want {
				t.Errorf("IsEmptyPatch() = %v, ingin %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// TestCountTotalPages — menguji penghitungan jumlah halaman dari total data
// dan batas per halaman. Menggunakan pembulatan ke atas tanpa bilangan pecahan.
//
// Rumus: (total + limit - 1) / limit  (integer division ceiling)
// =============================================================================

func TestCountTotalPages(t *testing.T) {
	tests := []struct {
		name  string
		total int // jumlah total data
		limit int // batas data per halaman
		want  int // jumlah halaman yang diharapkan
	}{
		{name: "habis dibagi → tepat 10 halaman", total: 100, limit: 10, want: 10},
		{name: "sisa 1 → dibulatkan ke atas jadi 11", total: 101, limit: 10, want: 11},
		{name: "data kurang dari limit → tetap 1 halaman", total: 3, limit: 10, want: 1},
		{name: "data kosong → 0 halaman", total: 0, limit: 10, want: 0},
		{name: "limit 0 → aman mengembalikan 0 (guard)", total: 50, limit: 0, want: 0},
		{name: "limit negatif → aman mengembalikan 0 (guard)", total: 50, limit: -1, want: 0},
		{name: "limit 1 → setiap data satu halaman", total: 5, limit: 1, want: 5},
		{name: "total sama dengan limit → tepat 1 halaman", total: 10, limit: 10, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CountTotalPages(tc.total, tc.limit)
			if got != tc.want {
				t.Errorf("CountTotalPages(%d, %d) = %d, ingin %d", tc.total, tc.limit, got, tc.want)
			}
		})
	}
}
