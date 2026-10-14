// Package repository - enrollment_repository.go
// Menangani operasi persistensi data KRS (enrollments), pengecekan duplikasi,
// penghitungan total SKS yang telah diambil, dan penghapusan KRS.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"siakad-mini/app/model"
)

var (
	ErrEnrollmentNotFound = errors.New("data KRS tidak ditemukan")
	ErrAlreadyEnrolled    = errors.New("mata kuliah sudah pernah diambil pada tahun akademik yang sama")
)

// EnrollmentRepository mendefinisikan interface operasi data KRS mahasiswa.
type EnrollmentRepository interface {
	FindByStudentAndCourse(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error)
	GetTotalSKSByStudent(ctx context.Context, tx pgx.Tx, studentID int, tahunAkademik string) (int, error)
	GetCourseEnrollmentCount(ctx context.Context, tx pgx.Tx, courseID int) (int, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error)
	FindByID(ctx context.Context, id int) (*model.Enrollment, error)
	DeleteByID(ctx context.Context, id int) error
	ListByStudentID(ctx context.Context, studentID int) ([]model.EnrolledCourseItem, error)
}

type enrollmentRepository struct {
	pool *pgxpool.Pool
}

// NewEnrollmentRepository membuat instance baru EnrollmentRepository.
func NewEnrollmentRepository(pool *pgxpool.Pool) EnrollmentRepository {
	return &enrollmentRepository{pool: pool}
}

// FindByStudentAndCourse memeriksa apakah mahasiswa sudah pernah mengambil mata kuliah tertentu pada tahun akademik yang sama.
func (r *enrollmentRepository) FindByStudentAndCourse(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error) {
	query := `
		SELECT id, student_id, course_id, tahun_akademik, created_at
		FROM enrollments
		WHERE student_id = $1 AND course_id = $2 AND tahun_akademik = $3
		LIMIT 1;
	`
	var e model.Enrollment
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, studentID, courseID, tahunAkademik).Scan(
			&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt,
		)
	} else {
		err = r.pool.QueryRow(ctx, query, studentID, courseID, tahunAkademik).Scan(
			&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Belum pernah diambil
		}
		return nil, err
	}
	return &e, nil
}

// GetTotalSKSByStudent menghitung akumulasi SKS yang telah diambil mahasiswa pada tahun akademik tertentu.
func (r *enrollmentRepository) GetTotalSKSByStudent(ctx context.Context, tx pgx.Tx, studentID int, tahunAkademik string) (int, error) {
	query := `
		SELECT COALESCE(SUM(c.sks), 0)
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE e.student_id = $1 AND e.tahun_akademik = $2;
	`
	var totalSKS int
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, studentID, tahunAkademik).Scan(&totalSKS)
	} else {
		err = r.pool.QueryRow(ctx, query, studentID, tahunAkademik).Scan(&totalSKS)
	}
	if err != nil {
		return 0, err
	}
	return totalSKS, nil
}

// GetCourseEnrollmentCount menghitung jumlah mahasiswa yang telah terdaftar pada mata kuliah tertentu.
func (r *enrollmentRepository) GetCourseEnrollmentCount(ctx context.Context, tx pgx.Tx, courseID int) (int, error) {
	query := `SELECT COUNT(*) FROM enrollments WHERE course_id = $1;`
	var count int
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, courseID).Scan(&count)
	} else {
		err = r.pool.QueryRow(ctx, query, courseID).Scan(&count)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CreateWithTx menyimpan record enrollment baru ke dalam database di dalam transaksi aktif.
func (r *enrollmentRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, studentID, courseID int, tahunAkademik string) (*model.Enrollment, error) {
	query := `
		INSERT INTO enrollments (student_id, course_id, tahun_akademik, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, student_id, course_id, tahun_akademik, created_at;
	`
	var e model.Enrollment
	err := tx.QueryRow(ctx, query, studentID, courseID, tahunAkademik).Scan(
		&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrAlreadyEnrolled
		}
		return nil, err
	}
	return &e, nil
}

// FindByID mencari record enrollment berdasarkan ID primary key.
func (r *enrollmentRepository) FindByID(ctx context.Context, id int) (*model.Enrollment, error) {
	query := `
		SELECT id, student_id, course_id, tahun_akademik, created_at
		FROM enrollments
		WHERE id = $1
		LIMIT 1;
	`
	var e model.Enrollment
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEnrollmentNotFound
		}
		return nil, err
	}
	return &e, nil
}

// DeleteByID menghapus record enrollment berdasarkan ID (pembatalan KRS).
func (r *enrollmentRepository) DeleteByID(ctx context.Context, id int) error {
	query := `DELETE FROM enrollments WHERE id = $1;`
	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}

// ListByStudentID mengambil seluruh mata kuliah yang diambil oleh mahasiswa tertentu (untuk detail mahasiswa).
func (r *enrollmentRepository) ListByStudentID(ctx context.Context, studentID int) ([]model.EnrolledCourseItem, error) {
	query := `
		SELECT e.id, c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, e.tahun_akademik
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE e.student_id = $1
		ORDER BY e.tahun_akademik DESC, c.semester ASC;
	`
	rows, err := r.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.EnrolledCourseItem, 0)
	for rows.Next() {
		var item model.EnrolledCourseItem
		if err := rows.Scan(
			&item.EnrollmentID, &item.CourseID, &item.KodeMK, &item.NamaMK,
			&item.SKS, &item.Semester, &item.TahunAkademik,
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
