// Package repository - student_repository.go
// Menangani operasi persistensi data mahasiswa ke tabel `students`,
// termasuk pagination, filtering, searching, dynamic sorting, dan soft delete.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

var (
	ErrStudentNotFound    = errors.New("mahasiswa tidak ditemukan")
	ErrNIMAlreadyExist    = errors.New("NIM sudah terdaftar")
)

// StudentRepository mendefinisikan interface operasi database data mahasiswa.
type StudentRepository interface {
	FindByUserID(ctx context.Context, userID int) (*model.Student, error)
	FindByID(ctx context.Context, id int) (*model.Student, error)
	FindByNIM(ctx context.Context, nim string) (*model.Student, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, s *model.Student) (*model.Student, error)
	Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, error)
	SoftDelete(ctx context.Context, id int) error
	List(ctx context.Context, q model.StudentListQuery) ([]model.StudentListItem, int64, error)
}

type studentRepository struct {
	pool *pgxpool.Pool
}

// NewStudentRepository membuat instance baru StudentRepository.
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentRepository{pool: pool}
}

// FindByUserID mencari mahasiswa berdasarkan relasi user_id (akun pengguna).
// Mahasiswa yang telah di-soft delete tidak akan ditemukan.
func (r *studentRepository) FindByUserID(ctx context.Context, userID int) (*model.Student, error) {
	query := `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at, updated_at
		FROM students
		WHERE user_id = $1 AND deleted_at IS NULL
		LIMIT 1;
	`
	var s model.Student
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan,
		&s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return &s, nil
}

// FindByID mencari mahasiswa aktif berdasarkan ID primary key.
func (r *studentRepository) FindByID(ctx context.Context, id int) (*model.Student, error) {
	query := `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at, updated_at
		FROM students
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1;
	`
	var s model.Student
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan,
		&s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return &s, nil
}

// FindByNIM mencari mahasiswa berdasarkan nomor induk mahasiswa (NIM).
func (r *studentRepository) FindByNIM(ctx context.Context, nim string) (*model.Student, error) {
	query := `
		SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, deleted_at, created_at, updated_at
		FROM students
		WHERE nim = $1 AND deleted_at IS NULL
		LIMIT 1;
	`
	var s model.Student
	err := r.pool.QueryRow(ctx, query, nim).Scan(
		&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan,
		&s.IPKTerakhir, &s.DeletedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return &s, nil
}

// CreateWithTx menyisipkan record mahasiswa baru dalam satu transaksi database.
func (r *studentRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, s *model.Student) (*model.Student, error) {
	query := `
		INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at;
	`
	var created model.Student
	err := tx.QueryRow(ctx, query, s.UserID, s.NIM, s.Nama, s.Prodi, s.Angkatan, s.IPKTerakhir).Scan(
		&created.ID, &created.UserID, &created.NIM, &created.Nama, &created.Prodi,
		&created.Angkatan, &created.IPKTerakhir, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrNIMAlreadyExist
		}
		return nil, err
	}
	return &created, nil
}

// Update memperbarui nama, prodi, angkatan, dan ipk_terakhir mahasiswa (NIM tetap tidak berubah).
func (r *studentRepository) Update(ctx context.Context, id int, req model.UpdateStudentRequest) (*model.Student, error) {
	ipk := 0.0
	if req.IPKTerakhir != nil {
		ipk = *req.IPKTerakhir
	}

	query := `
		UPDATE students
		SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4, updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at, updated_at;
	`
	var updated model.Student
	err := r.pool.QueryRow(ctx, query, req.Nama, req.Prodi, req.Angkatan, ipk, id).Scan(
		&updated.ID, &updated.UserID, &updated.NIM, &updated.Nama, &updated.Prodi,
		&updated.Angkatan, &updated.IPKTerakhir, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return &updated, nil
}

// SoftDelete mengisi kolom deleted_at dengan timestamp saat ini sehingga tidak muncul di query aktif.
func (r *studentRepository) SoftDelete(ctx context.Context, id int) error {
	query := `
		UPDATE students
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL;
	`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrStudentNotFound
	}
	return nil
}

// List mengambil daftar mahasiswa dengan pagination, filtering prodi & angkatan, pencarian nim/nama,
// serta sorting dinamis (nama, -nama, ipk_terakhir, -ipk_terakhir).
func (r *studentRepository) List(ctx context.Context, q model.StudentListQuery) ([]model.StudentListItem, int64, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Wajib hanya data yang belum di-soft delete
	conditions = append(conditions, "deleted_at IS NULL")

	if q.Prodi != "" {
		conditions = append(conditions, fmt.Sprintf("prodi ILIKE $%d", argIdx))
		args = append(args, "%"+q.Prodi+"%")
		argIdx++
	}

	if q.Angkatan > 0 {
		conditions = append(conditions, fmt.Sprintf("angkatan = $%d", argIdx))
		args = append(args, q.Angkatan)
		argIdx++
	}

	if q.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(nim ILIKE $%d OR nama ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+q.Search+"%")
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 1. Hitung total data untuk metadata paginasi
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM students %s;", whereClause)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 2. Tentukan pengurutan (Sorting)
	var orderBy string
	switch q.Sort {
	case "nama":
		orderBy = "ORDER BY nama ASC, id ASC"
	case "-nama":
		orderBy = "ORDER BY nama DESC, id DESC"
	case "ipk_terakhir":
		orderBy = "ORDER BY ipk_terakhir ASC, id ASC"
	case "-ipk_terakhir":
		orderBy = "ORDER BY ipk_terakhir DESC, id DESC"
	default:
		orderBy = "ORDER BY id ASC"
	}

	// 3. Klausul LIMIT dan OFFSET
	offset := (q.Page - 1) * q.PerPage
	dataQuery := fmt.Sprintf(`
		SELECT id, nim, nama, prodi, angkatan, ipk_terakhir
		FROM students
		%s
		%s
		LIMIT $%d OFFSET $%d;
	`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, q.PerPage, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]model.StudentListItem, 0)
	for rows.Next() {
		var item model.StudentListItem
		if err := rows.Scan(&item.ID, &item.NIM, &item.Nama, &item.Prodi, &item.Angkatan, &item.IPKTerakhir); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
