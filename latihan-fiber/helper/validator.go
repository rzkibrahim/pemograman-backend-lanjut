// Package helper - validator.go
// Mengelola inisialisasi go-playground/validator dengan custom tag
// dan konversi error validasi menjadi format map[string][]string yang seragam.
package helper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

// Regex untuk tahun akademik dengan format YYYY/YYYY-(Ganjil|Genap)
var academicYearRegex = regexp.MustCompile(`^\d{4}/\d{4}-(Ganjil|Genap)$`)

func init() {
	validate = validator.New()

	// Register custom validator untuk NIM 12 digit angka
	_ = validate.RegisterValidation("nim_12digit", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		if len(val) != 12 {
			return false
		}
		for _, ch := range val {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	})

	// Register custom validator untuk Angkatan (4 digit dan <= tahun sekarang)
	_ = validate.RegisterValidation("angkatan_valid", func(fl validator.FieldLevel) bool {
		val := int(fl.Field().Int())
		currentYear := time.Now().Year()
		return val >= 1990 && val <= currentYear
	})

	// Register custom validator untuk format Tahun Akademik: misal 2026/2027-Ganjil
	_ = validate.RegisterValidation("academic_year", func(fl validator.FieldLevel) bool {
		val := fl.Field().String()
		return academicYearRegex.MatchString(val)
	})
}

// ValidateStruct memvalidasi struct menggunakan tag validator
// Mengembalikan map[string][]string yang berisi pesan kesalahan per field,
// atau nil jika validasi lolos sepenuhnya.
func ValidateStruct(s any) map[string][]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errs := make(map[string][]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range validationErrors {
			field := strings.ToLower(fe.Field())
			// Jika tag json ditentukan, gunakan nama tag json
			msg := messageForTag(fe)
			errs[field] = append(errs[field], msg)
		}
	} else {
		errs["request"] = []string{"format data request tidak valid"}
	}

	return errs
}

// messageForTag menerjemahkan pesan error validator ke bahasa Indonesia yang ramah pengguna
func messageForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Field %s wajib diisi", fe.Field())
	case "email":
		return "Format email tidak valid"
	case "min":
		if fe.Type().Kind().String() == "string" {
			return fmt.Sprintf("Minimal %s karakter", fe.Param())
		}
		return fmt.Sprintf("Nilai minimal adalah %s", fe.Param())
	case "max":
		if fe.Type().Kind().String() == "string" {
			return fmt.Sprintf("Maksimal %s karakter", fe.Param())
		}
		return fmt.Sprintf("Nilai maksimal adalah %s", fe.Param())
	case "nim_12digit":
		return "NIM wajib berupa 12 digit angka"
	case "angkatan_valid":
		return fmt.Sprintf("Angkatan harus berupa 4 digit tahun dan tidak melebihi tahun berjalan (%d)", time.Now().Year())
	case "academic_year":
		return "Format tahun akademik harus YYYY/YYYY-(Ganjil|Genap), contoh: 2026/2027-Ganjil"
	case "oneof":
		return fmt.Sprintf("Nilai harus salah satu dari: %s", fe.Param())
	default:
		return fmt.Sprintf("Field %s tidak valid", fe.Field())
	}
}

// Helper untuk parsing query integer dengan default dan batasan
func ParseQueryInt(val string, defaultVal, minVal, maxVal int) int {
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	if parsed < minVal {
		return minVal
	}
	if maxVal > 0 && parsed > maxVal {
		return maxVal
	}
	return parsed
}
