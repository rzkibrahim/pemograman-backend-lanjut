package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"modul9/app/model"
)

// Sentinel errors untuk domain course
var (
	ErrCourseFull      = errors.New("COURSE_FULL")
	ErrAlreadyEnrolled = errors.New("ALREADY_ENROLLED")
	ErrNotEnrolled     = errors.New("NOT_ENROLLED")
	ErrVersionConflict = errors.New("VERSION_CONFLICT")
	ErrQuotaBelow      = errors.New("QUOTA_BELOW_ENROLLED")
)

// CourseRepository mendefinisikan operasi CRUD dan operasi transaksional untuk courses.
type CourseRepository struct {
	db DBTX
}

// NewCourseRepository membuat instance CourseRepository dengan pool.
func NewCourseRepository(db DBTX) *CourseRepository {
	return &CourseRepository{db: db}
}

// WithTx menghasilkan salinan CourseRepository yang terpasang ke transaction.
func (r *CourseRepository) WithTx(tx DBTX) *CourseRepository {
	return &CourseRepository{db: tx}
}

// FindAll mengembalikan seluruh kelas.
func (r *CourseRepository) FindAll(ctx context.Context) ([]model.Course, error) {
	query := `SELECT id, code, name, credits, quota, enrolled_count, version, created_at
	          FROM courses ORDER BY id`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar course: %w", err)
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota,
			&c.EnrolledCount, &c.Version, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca baris course: %w", err)
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

// FindByID mengembalikan satu kelas berdasarkan ID.
func (r *CourseRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, credits, quota, enrolled_count, version, created_at
		 FROM courses WHERE id = $1`, id,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota,
		&c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("mengambil course: %w", err)
	}
	return c, nil
}

// --- Teknik 1: UPDATE bersyarat atomik ---

// IncrementEnrolled menambah enrolled_count jika belum melebihi quota.
// Mengembalikan ErrCourseFull jika UPDATE kena 0 baris.
func (r *CourseRepository) IncrementEnrolled(ctx context.Context, courseID int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count + 1
		 WHERE id = $1 AND enrolled_count < quota`, courseID)
	if err != nil {
		return fmt.Errorf("menambah enrolled_count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCourseFull
	}
	return nil
}

// DecrementEnrolled mengurangi enrolled_count.
func (r *CourseRepository) DecrementEnrolled(ctx context.Context, courseID int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count - 1
		 WHERE id = $1 AND enrolled_count > 0`, courseID)
	if err != nil {
		return fmt.Errorf("mengurangi enrolled_count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertEnrollment menyisipkan baris enrollment. Mengembalikan ErrAlreadyEnrolled pada duplikasi.
func (r *CourseRepository) InsertEnrollment(ctx context.Context, studentID, courseID int) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO enrollments (student_id, course_id) VALUES ($1, $2)`,
		studentID, courseID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyEnrolled
		}
		return fmt.Errorf("menyisipkan enrollment: %w", err)
	}
	return nil
}

// DeleteEnrollment menghapus baris enrollment. Mengembalikan ErrNotEnrolled jika tidak ada.
func (r *CourseRepository) DeleteEnrollment(ctx context.Context, studentID, courseID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM enrollments WHERE student_id = $1 AND course_id = $2`,
		studentID, courseID)
	if err != nil {
		return fmt.Errorf("menghapus enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotEnrolled
	}
	return nil
}

// --- Teknik 2: Pessimistic locking ---

// LockStudent mengunci baris student dengan SELECT ... FOR UPDATE
// untuk mencegah satu mahasiswa mengirim banyak request serentak.
func (r *CourseRepository) LockStudent(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`, studentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("mengunci student: %w", err)
	}
	return nil
}

// SumCredits menghitung total SKS yang sedang diambil mahasiswa.
func (r *CourseRepository) SumCredits(ctx context.Context, studentID int) (int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.credits), 0)
		 FROM enrollments e
		 JOIN courses c ON c.id = e.course_id
		 WHERE e.student_id = $1`, studentID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("menghitung SKS: %w", err)
	}
	return total, nil
}

// LockCourses mengunci beberapa kelas dengan urutan ID terkecil dulu untuk mencegah deadlock.
func (r *CourseRepository) LockCourses(ctx context.Context, courseIDs ...int) ([]model.Course, error) {
	// Urutkan ID untuk mencegah deadlock
	sorted := make([]int, len(courseIDs))
	copy(sorted, courseIDs)
	sort.Ints(sorted)

	courses := make([]model.Course, 0, len(sorted))
	for _, id := range sorted {
		var c model.Course
		err := r.db.QueryRow(ctx,
			`SELECT id, code, name, credits, quota, enrolled_count, version, created_at
			 FROM courses WHERE id = $1 FOR UPDATE`, id,
		).Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota,
			&c.EnrolledCount, &c.Version, &c.CreatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("mengunci course %d: %w", id, err)
		}
		courses = append(courses, c)
	}
	return courses, nil
}

// --- Teknik 3: Optimistic locking ---

// UpdateQuota mengubah kuota kelas dengan optimistic locking (version).
// Mengembalikan ErrVersionConflict jika version tidak cocok.
// Mengembalikan ErrQuotaBelow jika kuota baru < enrolled_count.
func (r *CourseRepository) UpdateQuota(ctx context.Context, id, newQuota, expectedVersion int) (model.Course, error) {
	var c model.Course

	// Cek apakah kuota baru >= enrolled_count
	var enrolledCount int
	err := r.db.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, id).Scan(&enrolledCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("membaca enrolled_count: %w", err)
	}

	if newQuota < enrolledCount {
		return model.Course{}, ErrQuotaBelow
	}

	err = r.db.QueryRow(ctx,
		`UPDATE courses SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3
		 RETURNING id, code, name, credits, quota, enrolled_count, version, created_at`,
		newQuota, id, expectedVersion,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota,
		&c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrVersionConflict
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			// CHECK constraint violation (quota < enrolled)
			return model.Course{}, ErrQuotaBelow
		}
		return model.Course{}, fmt.Errorf("memperbarui kuota: %w", err)
	}
	return c, nil
}
