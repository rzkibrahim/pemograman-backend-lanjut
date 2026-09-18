package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul6/app/model"
)

// UserRepository adalah kontrak antarmuka untuk operasi penyimpanan data user.
type UserRepository interface {
	FindAll(ctx context.Context) ([]model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	Create(ctx context.Context, u model.User) (model.User, error)
	Update(ctx context.Context, u model.User) (model.User, error)
	UpdateRole(ctx context.Context, id int, role string) (model.User, error)
	Delete(ctx context.Context, id int) error
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository membuat instansiasi UserRepository berbasis PostgreSQL.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func (r *userPostgresRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var u model.User
	query := `SELECT id, username, email, password, role, is_active, created_at 
              FROM users WHERE LOWER(username) = LOWER($1)`

	err := r.pool.QueryRow(ctx, query, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mencari user berdasarkan username: %w", err)
	}

	return u, nil
}

func (r *userPostgresRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	query := `SELECT id, username, email, password, role, is_active, created_at 
              FROM users WHERE LOWER(email) = LOWER($1)`

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mencari user berdasarkan email: %w", err)
	}

	return u, nil
}

func (r *userPostgresRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	query := `SELECT id, username, email, password, role, is_active, created_at 
              FROM users WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("mencari user berdasarkan id: %w", err)
	}

	return u, nil
}

func (r *userPostgresRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	query := `INSERT INTO users (username, email, password, role, is_active)
              VALUES ($1, $2, $3, $4, $5)
              RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query, u.Username, u.Email, u.Password, u.Role, u.IsActive).
		Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("menyimpan user: %w", err)
	}

	return u, nil
}

func (r *userPostgresRepository) FindAll(ctx context.Context) ([]model.User, error) {
	query := `SELECT id, username, email, password, role, is_active, created_at FROM users ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca data user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *userPostgresRepository) Update(ctx context.Context, u model.User) (model.User, error) {
	query := `UPDATE users 
              SET email = $1 
              WHERE id = $2 
              RETURNING id, username, email, password, role, is_active, created_at`

	err := r.pool.QueryRow(ctx, query, u.Email, u.ID).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.User{}, ErrDuplicate
		}
		return model.User{}, fmt.Errorf("memperbarui user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) UpdateRole(ctx context.Context, id int, role string) (model.User, error) {
	query := `UPDATE users 
              SET role = $1 
              WHERE id = $2 
              RETURNING id, username, email, password, role, is_active, created_at`

	var u model.User
	err := r.pool.QueryRow(ctx, query, role, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("memperbarui role user: %w", err)
	}
	return u, nil
}

func (r *userPostgresRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

