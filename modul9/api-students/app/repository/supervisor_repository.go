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

// Sentinel errors untuk domain supervisor
var (
	ErrSupervisorFull      = errors.New("SUPERVISOR_FULL")
	ErrAlreadySupervised   = errors.New("ALREADY_SUPERVISED")
	ErrNotSupervised       = errors.New("NOT_SUPERVISED")
	ErrSupervisorVersion   = errors.New("VERSION_CONFLICT")
	ErrSupervisorQuotaBelow = errors.New("QUOTA_BELOW_SUPERVISED")
)

// SupervisorRepository mendefinisikan operasi untuk domain supervisor.
type SupervisorRepository struct {
	db DBTX
}

// NewSupervisorRepository membuat instance SupervisorRepository.
func NewSupervisorRepository(db DBTX) *SupervisorRepository {
	return &SupervisorRepository{db: db}
}

// WithTx menghasilkan salinan repository yang terpasang ke transaction.
func (r *SupervisorRepository) WithTx(tx DBTX) *SupervisorRepository {
	return &SupervisorRepository{db: tx}
}

// FindAll mengembalikan seluruh supervisor.
func (r *SupervisorRepository) FindAll(ctx context.Context) ([]model.Supervisor, error) {
	query := `SELECT id, name, quota, supervised_count, version, created_at
	          FROM supervisors ORDER BY id`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar supervisor: %w", err)
	}
	defer rows.Close()

	var supervisors []model.Supervisor
	for rows.Next() {
		var s model.Supervisor
		if err := rows.Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount,
			&s.Version, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca baris supervisor: %w", err)
		}
		supervisors = append(supervisors, s)
	}
	return supervisors, rows.Err()
}

// FindByID mengembalikan satu supervisor.
func (r *SupervisorRepository) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	var s model.Supervisor
	err := r.db.QueryRow(ctx,
		`SELECT id, name, quota, supervised_count, version, created_at
		 FROM supervisors WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("mengambil supervisor: %w", err)
	}
	return s, nil
}

// IncrementSupervised menambah supervised_count jika belum penuh (UPDATE bersyarat atomik).
func (r *SupervisorRepository) IncrementSupervised(ctx context.Context, supervisorID int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count + 1
		 WHERE id = $1 AND supervised_count < quota`, supervisorID)
	if err != nil {
		return fmt.Errorf("menambah supervised_count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSupervisorFull
	}
	return nil
}

// DecrementSupervised mengurangi supervised_count.
func (r *SupervisorRepository) DecrementSupervised(ctx context.Context, supervisorID int) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count - 1
		 WHERE id = $1 AND supervised_count > 0`, supervisorID)
	if err != nil {
		return fmt.Errorf("mengurangi supervised_count: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertSupervision menyisipkan relasi mahasiswa-pembimbing.
func (r *SupervisorRepository) InsertSupervision(ctx context.Context, studentID, supervisorID int) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO supervisions (student_id, supervisor_id) VALUES ($1, $2)`,
		studentID, supervisorID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadySupervised
		}
		return fmt.Errorf("menyisipkan supervision: %w", err)
	}
	return nil
}

// DeleteSupervision menghapus relasi mahasiswa-pembimbing.
func (r *SupervisorRepository) DeleteSupervision(ctx context.Context, studentID, supervisorID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM supervisions WHERE student_id = $1 AND supervisor_id = $2`,
		studentID, supervisorID)
	if err != nil {
		return fmt.Errorf("menghapus supervision: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotSupervised
	}
	return nil
}

// LockStudent mengunci baris student untuk mencegah banyak request serentak.
func (r *SupervisorRepository) LockStudent(ctx context.Context, studentID int) error {
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

// LockSupervisors mengunci beberapa supervisor dengan urutan ID terkecil dulu (mencegah deadlock).
func (r *SupervisorRepository) LockSupervisors(ctx context.Context, ids ...int) ([]model.Supervisor, error) {
	sorted := make([]int, len(ids))
	copy(sorted, ids)
	sort.Ints(sorted)

	supervisors := make([]model.Supervisor, 0, len(sorted))
	for _, id := range sorted {
		var s model.Supervisor
		err := r.db.QueryRow(ctx,
			`SELECT id, name, quota, supervised_count, version, created_at
			 FROM supervisors WHERE id = $1 FOR UPDATE`, id,
		).Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("mengunci supervisor %d: %w", id, err)
		}
		supervisors = append(supervisors, s)
	}
	return supervisors, nil
}

// GetCurrentSupervisor mengembalikan supervisor_id mahasiswa saat ini, atau 0 jika belum punya.
func (r *SupervisorRepository) GetCurrentSupervisor(ctx context.Context, studentID int) (int, error) {
	var supervisorID int
	err := r.db.QueryRow(ctx,
		`SELECT supervisor_id FROM supervisions WHERE student_id = $1`,
		studentID).Scan(&supervisorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("mengambil supervisor mahasiswa: %w", err)
	}
	return supervisorID, nil
}

// UpdateQuota mengubah kuota supervisor dengan optimistic locking.
func (r *SupervisorRepository) UpdateQuota(ctx context.Context, id, newQuota, expectedVersion int) (model.Supervisor, error) {
	var s model.Supervisor

	var supervisedCount int
	err := r.db.QueryRow(ctx,
		`SELECT supervised_count FROM supervisors WHERE id = $1`, id).Scan(&supervisedCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrNotFound
		}
		return model.Supervisor{}, fmt.Errorf("membaca supervised_count: %w", err)
	}

	if newQuota < supervisedCount {
		return model.Supervisor{}, ErrSupervisorQuotaBelow
	}

	err = r.db.QueryRow(ctx,
		`UPDATE supervisors SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3
		 RETURNING id, name, quota, supervised_count, version, created_at`,
		newQuota, id, expectedVersion,
	).Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Supervisor{}, ErrSupervisorVersion
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return model.Supervisor{}, ErrSupervisorQuotaBelow
		}
		return model.Supervisor{}, fmt.Errorf("memperbarui kuota supervisor: %w", err)
	}
	return s, nil
}
