package model

import "time"

// StudentPortfolio merepresentasikan portofolio mahasiswa (Tugas Mandiri Modul 10).
// Satu dokumen per mahasiswa (dijaga oleh unique index pada student_id).
type StudentPortfolio struct {
	ID           string                 `json:"id"`
	StudentID    int                    `json:"student_id"`
	Headline     string                 `json:"headline"`
	Bio          string                 `json:"bio"`
	Projects     []PortfolioProject     `json:"projects"`
	Certificates []PortfolioCertificate `json:"certificates"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PortfolioProject merepresentasikan karya/proyek yang dikerjakan mahasiswa (embedded, maks 20).
type PortfolioProject struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link,omitempty"`
	Year         int      `json:"year"`
}

// PortfolioCertificate merepresentasikan sertifikasi yang dimiliki mahasiswa (embedded).
type PortfolioCertificate struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

// UpsertPortfolioRequest adalah payload pembuatan/pembaruan profil portofolio dasar.
type UpsertPortfolioRequest struct {
	Headline string `json:"headline" validate:"required,min=3,max=100"`
	Bio      string `json:"bio"      validate:"required,min=10,max=1000"`
}

// AddProjectRequest adalah payload penambahan proyek ke dalam portofolio (maks 20 proyek).
type AddProjectRequest struct {
	Title        string   `json:"title"        validate:"required,min=3,max=100"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10,dive,min=1,max=30"`
	Link         string   `json:"link"         validate:"omitempty,url"`
	Year         int      `json:"year"         validate:"required,min=2000,max=2100"`
}

// TechnologyStat merepresentasikan hasil agregasi top teknologi dari seluruh proyek mahasiswa.
type TechnologyStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
