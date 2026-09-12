package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul5/app/model"
)

// UserRepository adalah kontrak antarmuka untuk operasi penyimpanan data user.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	Create(ctx context.Context, u model.User) (model.User, error)
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
