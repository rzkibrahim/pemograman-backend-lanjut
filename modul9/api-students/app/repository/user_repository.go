package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul9/app/model"
)

type UserRepository interface {
	FindAll(ctx context.Context) ([]model.User, error)
	FindAfterCursor(ctx context.Context, q model.CursorQuery) ([]model.User, error)
	FindByUsername(ctx context.Context, username string) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
	Create(ctx context.Context, u model.User) (model.User, error)
	Update(ctx context.Context, u model.User) (model.User, error)
	UpdateRole(ctx context.Context, id int, role string) (model.User, error)
	Delete(ctx context.Context, id int) error
}

const userColumns = "id, username, email, password, role, is_active, created_at"

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func scanUser(rows pgx.Rows) (model.User, error) {
	var u model.User
	err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.IsActive, &u.CreatedAt)
	return u, err
}

func (r *userPostgresRepository) FindAfterCursor(
	ctx context.Context, q model.CursorQuery,
) ([]model.User, error) {
	args := []any{}
	where := " WHERE 1 = 1"

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND username ILIKE $%d", len(args))
	}

	if q.IsActive != nil {
		args = append(args, *q.IsActive)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}

	if q.After != nil {
		args = append(args, q.After.CreatedAt, q.After.ID)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)",
			len(args)-1, len(args))
	}

	args = append(args, q.Limit+1)
	query := fmt.Sprintf(
		"SELECT %s FROM users%s ORDER BY created_at DESC, id DESC LIMIT $%d",
		userColumns, where, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar user: %w", err)
	}
	defer rows.Close()

	result := []model.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca row user: %w", err)
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca hasil query: %w", err)
	}
	return result, nil
}

func (r *userPostgresRepository) FindByUsername(ctx context.Context, username string) (model.User, error) {
	var u model.User
	query := "SELECT " + userColumns + " FROM users WHERE LOWER(username) = LOWER($1)"
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
	query := "SELECT " + userColumns + " FROM users WHERE LOWER(email) = LOWER($1)"
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
	query := "SELECT " + userColumns + " FROM users WHERE id = $1"
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
	query := "SELECT " + userColumns + " FROM users ORDER BY id ASC"
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
	query := `UPDATE users SET email = $1 WHERE id = $2
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
	query := `UPDATE users SET role = $1 WHERE id = $2
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
	tag, err := r.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("menghapus user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
