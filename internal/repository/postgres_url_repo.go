package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raizel/gateway/internal/model"
)

const (
	// PostgreSQL unique violation error code.
	pgUniqueViolation = "23505"
)

// PostgresURLRepo implements URLRepository backed by PostgreSQL via pgx.
type PostgresURLRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresURLRepo creates a new PostgreSQL repository with a connection pool.
func NewPostgresURLRepo(pool *pgxpool.Pool) *PostgresURLRepo {
	return &PostgresURLRepo{pool: pool}
}

// Store inserts a new URL record. Returns ErrDuplicateCode if the short_code violates uniqueness.
func (r *PostgresURLRepo) Store(ctx context.Context, u *model.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, expires_at, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		u.ShortCode,
		u.OriginalURL,
		u.ExpiresAt,
		u.IsActive,
		u.CreatedBy,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return model.ErrDuplicateCode
		}
		return fmt.Errorf("storing url: %w", err)
	}

	return nil
}

// FindByShortCode retrieves an active, non-expired URL by its short code.
func (r *PostgresURLRepo) FindByShortCode(ctx context.Context, code string) (*model.URL, error) {
	query := `
		SELECT id, short_code, original_url, clicks, created_at, updated_at, expires_at, is_active, created_by
		FROM urls
		WHERE short_code = $1`

	var u model.URL
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&u.ID,
		&u.ShortCode,
		&u.OriginalURL,
		&u.Clicks,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ExpiresAt,
		&u.IsActive,
		&u.CreatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrURLNotFound
		}
		return nil, fmt.Errorf("finding url by code: %w", err)
	}

	if !u.IsActive {
		return nil, model.ErrURLInactive
	}

	if u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrURLExpired
	}

	return &u, nil
}

// IncrementClicks atomically increments the click counter for a short code.
func (r *PostgresURLRepo) IncrementClicks(ctx context.Context, code string) error {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code = $1`
	tag, err := r.pool.Exec(ctx, query, code)
	if err != nil {
		return fmt.Errorf("incrementing clicks: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrURLNotFound
	}
	return nil
}

// Ping verifies database connectivity.
func (r *PostgresURLRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
