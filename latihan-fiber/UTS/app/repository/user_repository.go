// Package repository - user_repository.go
// Menangani operasi persistensi data tabel `users` ke database PostgreSQL via pgxpool.
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
	ErrUserNotFound      = errors.New("user tidak ditemukan")
	ErrEmailAlreadyExist = errors.New("email sudah terdaftar")
)

// UserRepository adalah interface untuk operasi database entitas User.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	CreateWithTx(ctx context.Context, tx pgx.Tx, email, hashedPassword, role string) (*model.User, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository menginisialisasi repository user dengan connection pool.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

// FindByEmail mencari user aktif berdasarkan alamat email.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password, role, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1;
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// FindByID mencari user berdasarkan ID primary key.
func (r *userRepository) FindByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT id, email, password, role, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1;
	`
	var u model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateWithTx menyisipkan record user baru dalam konteks transaksi database (tx).
func (r *userRepository) CreateWithTx(ctx context.Context, tx pgx.Tx, email, hashedPassword, role string) (*model.User, error) {
	query := `
		INSERT INTO users (email, password, role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, email, role, created_at, updated_at;
	`
	var u model.User
	err := tx.QueryRow(ctx, query, email, hashedPassword, role).Scan(
		&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrEmailAlreadyExist
		}
		return nil, err
	}
	return &u, nil
}
