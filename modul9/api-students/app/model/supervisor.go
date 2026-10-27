package model

import "time"

// Supervisor merepresentasikan pembimbing tugas akhir.
type Supervisor struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Quota           int       `json:"quota"`
	SupervisedCount int       `json:"supervised_count"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

// Supervision merepresentasikan relasi mahasiswa-pembimbing.
type Supervision struct {
	StudentID    int       `json:"student_id"`
	SupervisorID int       `json:"supervisor_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// AssignSupervisorRequest adalah payload untuk menetapkan pembimbing.
type AssignSupervisorRequest struct {
	StudentID int `json:"student_id" validate:"required,min=1"`
}

// SwitchSupervisorRequest adalah payload untuk menukar pembimbing.
type SwitchSupervisorRequest struct {
	StudentID        int `json:"student_id"         validate:"required,min=1"`
	FromSupervisorID int `json:"from_supervisor_id" validate:"required,min=1"`
	ToSupervisorID   int `json:"to_supervisor_id"   validate:"required,min=1"`
}

// UpdateSupervisorQuotaRequest adalah payload untuk mengubah kuota pembimbing.
type UpdateSupervisorQuotaRequest struct {
	Quota   int `json:"quota"   validate:"required,min=1"`
	Version int `json:"version" validate:"required,min=1"`
}
