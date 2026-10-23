package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"modul9/app/model"
)

// TokenRepository adalah kontrak penyimpanan data refresh token di basis data.
type TokenRepository interface {
	Create(ctx context.Context, token model.RefreshToken) (model.RefreshToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (model.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeByUserID(ctx context.Context, userID int) error
}

type tokenPostgresRepository struct {
	pool *pgxpool.Pool
}

// NewTokenRepository membuat instance TokenRepository berbasis PostgreSQL.
func NewTokenRepository(pool *pgxpool.Pool) TokenRepository {
	return &tokenPostgresRepository{pool: pool}
}

func (r *tokenPostgresRepository) Create(ctx context.Context, token model.RefreshToken) (model.RefreshToken, error) {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
              VALUES ($1, $2, $3)
              RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return model.RefreshToken{}, fmt.Errorf("menyimpan refresh token: %w", err)
	}

	return token, nil
}

func (r *tokenPostgresRepository) FindByTokenHash(ctx context.Context, tokenHash string) (model.RefreshToken, error) {
	var t model.RefreshToken
	query := `SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
              FROM refresh_tokens WHERE token_hash = $1`

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.RefreshToken{}, ErrNotFound
		}
		return model.RefreshToken{}, fmt.Errorf("mencari refresh token: %w", err)
	}

	return t, nil
}

func (r *tokenPostgresRepository) Revoke(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW()
              WHERE token_hash = $1 AND revoked_at IS NULL`

	cmd, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("mencabut refresh token: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *tokenPostgresRepository) RevokeByUserID(ctx context.Context, userID int) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW()
              WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mencabut seluruh refresh token user: %w", err)
	}

	return nil
}
