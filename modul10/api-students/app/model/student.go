package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM      string  `json:"nim"       validate:"required,nim_format"`
	Name     string  `json:"name"      validate:"required,min=2,max=100"`
	Grade    float64 `json:"grade"     validate:"min=0,max=100"`
	IsActive bool    `json:"is_active"`
}

type ReplaceStudentRequest struct {
	NIM      string  `json:"nim"       validate:"required,nim_format"`
	Name     string  `json:"name"      validate:"required,min=2,max=100"`
	Grade    float64 `json:"grade"     validate:"min=0,max=100"`
	IsActive bool    `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"       validate:"omitnil,nim_format"`
	Name     *string  `json:"name,omitempty"      validate:"omitnil,min=2,max=100"`
	Grade    *float64 `json:"grade,omitempty"     validate:"omitnil,min=0,max=100"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
	GradeMin *float64
	GradeMax *float64
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

type StudentCursorQuery struct {
	Limit    int
	Search   string
	IsActive *bool
	After    *Cursor
}
