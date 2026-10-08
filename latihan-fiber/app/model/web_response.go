// Package model - web_response.go
// Mendefinisikan struktur seragam untuk seluruh HTTP response (sukses, pagination, error)
// sesuai dengan format yang ditentukan pada spesifikasi UTS SIAKAD Mini.
package model

// WebResponse adalah struktur baku untuk respons JSON sukses.
type WebResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    any             `json:"data,omitempty"`
	Meta    *PaginationMeta `json:"meta,omitempty"`
}

// PaginationMeta memuat metadata paginasi halaman (page-based pagination).
// Sesuai contoh pada Endpoint 3 UTS: current_page, per_page, total, last_page.
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	LastPage    int   `json:"last_page"`
}

// ErrorResponse adalah struktur baku respons error seragam (4xx, 5xx).
// Untuk validasi 422, field errors berisi daftar pesan kegagalan per field.
type ErrorResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	Errors    map[string][]string `json:"errors,omitempty"`
	Code      string              `json:"code,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}
