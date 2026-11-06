package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"

	"latihan-fiber/app/model"
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

// NewCourseRepository membuat instance CourseRepository dengan pool atau koneksi DBTX.
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

func (r *CourseRepository) IsEnrolled(ctx context.Context, studentID, courseID int) (bool, error) {
	var enrolled bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM enrollments WHERE student_id = $1 AND course_id = $2
		)`, studentID, courseID,
	).Scan(&enrolled)
	if err != nil {
		return false, fmt.Errorf("memeriksa enrollment: %w", err)
	}
	return enrolled, nil
}

// --- Teknik 1: UPDATE bersyarat atomik ---

// IncrementEnrolled menambah enrolled_count jika belum melebihi quota dan mengembalikan data course terbaru.
// Mengembalikan ErrCourseFull jika kuota sudah penuh (0 baris ter-update).
func (r *CourseRepository) IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count + 1
		 WHERE id = $1 AND enrolled_count < quota
		 RETURNING id, code, name, credits, quota, enrolled_count, version, created_at`, courseID,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota, &c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrCourseFull
		}
		return model.Course{}, fmt.Errorf("menambah enrolled_count: %w", err)
	}
	return c, nil
}

// DecrementEnrolled mengurangi enrolled_count dan mengembalikan data course terbaru.
func (r *CourseRepository) DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count - 1
		 WHERE id = $1 AND enrolled_count > 0
		 RETURNING id, code, name, credits, quota, enrolled_count, version, created_at`, courseID,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota, &c.EnrolledCount, &c.Version, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("mengurangi enrolled_count: %w", err)
	}
	return c, nil
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

// GetEnrolledStudents mengambil seluruh student_id yang terdaftar pada suatu course (dibutuhkan untuk Tugas Mandiri D.2).
func (r *CourseRepository) GetEnrolledStudents(ctx context.Context, courseID int) ([]int, error) {
	rows, err := r.db.Query(ctx, `SELECT student_id FROM enrollments WHERE course_id = $1 ORDER BY student_id`, courseID)
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta kelas: %w", err)
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("membaca student_id peserta: %w", err)
		}
		studentIDs = append(studentIDs, id)
	}
	return studentIDs, rows.Err()
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

// UpdateQuota memperbarui kuota kelas dengan memeriksa version secara atomik.
func (r *CourseRepository) UpdateQuota(ctx context.Context, id int, newQuota, expectedVersion int) (model.Course, error) {
	var currentEnrolled int
	err := r.db.QueryRow(ctx,
		`SELECT enrolled_count FROM courses WHERE id = $1`, id,
	).Scan(&currentEnrolled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Course{}, ErrNotFound
		}
		return model.Course{}, fmt.Errorf("memeriksa course: %w", err)
	}

	if newQuota < currentEnrolled {
		return model.Course{}, ErrQuotaBelow
	}

	var updated model.Course
	err = r.db.QueryRow(ctx,
		`UPDATE courses
		 SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3
		 RETURNING id, code, name, credits, quota, enrolled_count, version, created_at`,
		newQuota, id, expectedVersion,
	).Scan(
		&updated.ID, &updated.Code, &updated.Name, &updated.Credits,
		&updated.Quota, &updated.EnrolledCount, &updated.Version, &updated.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			_ = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, id).Scan(&exists)
			if !exists {
				return model.Course{}, ErrNotFound
			}
			return model.Course{}, ErrVersionConflict
		}
		return model.Course{}, fmt.Errorf("memperbarui kuota: %w", err)
	}

	return updated, nil
}
