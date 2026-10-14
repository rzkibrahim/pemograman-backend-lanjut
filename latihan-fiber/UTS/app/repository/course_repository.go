// Package repository - course_repository.go
// Menangani operasi persistensi data mata kuliah, perhitungan sisa kuota dinamis,
// dan row-level locking (SELECT ... FOR UPDATE) untuk mencegah race condition.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

var (
	ErrCourseNotFound = errors.New("mata kuliah tidak ditemukan")
)

// CourseRepository mendefinisikan interface operasi data mata kuliah.
type CourseRepository interface {
	List(ctx context.Context, q model.CourseQuery) ([]model.CourseItemResponse, error)
	FindByID(ctx context.Context, id int) (*model.Course, error)
	FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id int) (*model.Course, error)
}

type courseRepository struct {
	pool *pgxpool.Pool
}

// NewCourseRepository membuat instance baru CourseRepository.
func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &courseRepository{pool: pool}
}

// List mengambil daftar mata kuliah beserta perhitungan kuota terisi dan sisa kuota.
// Mendukung filter semester, search kode/nama, dan filter available (hanya kuota tersisa).
func (r *courseRepository) List(ctx context.Context, q model.CourseQuery) ([]model.CourseItemResponse, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if q.Semester > 0 {
		conditions = append(conditions, fmt.Sprintf("c.semester = $%d", argIdx))
		args = append(args, q.Semester)
		argIdx++
	}

	if q.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+q.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	havingClause := ""
	if q.Available {
		havingClause = "HAVING (c.kuota - COUNT(e.id)) > 0"
	}

	query := fmt.Sprintf(`
		SELECT 
			c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
			COUNT(e.id) AS terisi,
			(c.kuota - COUNT(e.id)) AS sisa_kuota
		FROM courses c
		LEFT JOIN enrollments e ON c.id = e.course_id
		%s
		GROUP BY c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota
		%s
		ORDER BY c.semester ASC, c.kode_mk ASC;
	`, whereClause, havingClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.CourseItemResponse, 0)
	for rows.Next() {
		var item model.CourseItemResponse
		if err := rows.Scan(
			&item.ID, &item.KodeMK, &item.NamaMK, &item.SKS, &item.Semester,
			&item.Kuota, &item.Terisi, &item.SisaKuota,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// FindByID mencari mata kuliah berdasarkan ID.
func (r *courseRepository) FindByID(ctx context.Context, id int) (*model.Course, error) {
	query := `
		SELECT id, kode_mk, nama_mk, sks, semester, kuota, created_at, updated_at
		FROM courses
		WHERE id = $1
		LIMIT 1;
	`
	var c model.Course
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}

// FindByIDForUpdate melakukan penguncian baris (Row-Level Locking via SELECT ... FOR UPDATE)
// dalam konteks transaksi aktif. Ini penting untuk mencegah race condition overbooking kuota
// ketika banyak mahasiswa mengambil mata kuliah yang sama secara bersamaan.
func (r *courseRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id int) (*model.Course, error) {
	query := `
		SELECT id, kode_mk, nama_mk, sks, semester, kuota, created_at, updated_at
		FROM courses
		WHERE id = $1
		FOR UPDATE;
	`
	var c model.Course
	err := tx.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &c, nil
}
