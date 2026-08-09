package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type SessionRepository struct {
	executor QueryExecutor
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{executor: pool}
}

func (r *SessionRepository) Create(ctx context.Context, params usecase.CreateSessionParams) (model.Session, error) {
	session, err := scanSession(executorFromContext(ctx, r.executor).QueryRow(
		ctx,
		`
INSERT INTO sessions (user_id, refresh_token_hash, expires_at, last_used_at, user_agent)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, last_used_at, user_agent
`,
		params.UserID,
		params.RefreshTokenHash,
		params.ExpiresAt,
		params.LastUsedAt,
		params.UserAgent,
	))
	if err != nil {
		return model.Session{}, mapSessionError("create session", err)
	}

	return session, nil
}

func (r *SessionRepository) GetActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash []byte, now time.Time) (model.Session, error) {
	session, err := scanSession(executorFromContext(ctx, r.executor).QueryRow(
		ctx,
		`
SELECT id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, last_used_at, user_agent
FROM sessions
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > $2
`,
		refreshTokenHash,
		now,
	))
	if err != nil {
		return model.Session{}, mapSessionError("get active session by refresh token hash", err)
	}

	return session, nil
}

func (r *SessionRepository) GetActiveByIDAndUserID(ctx context.Context, sessionID, userID uuid.UUID, now time.Time) (model.Session, error) {
	session, err := scanSession(executorFromContext(ctx, r.executor).QueryRow(
		ctx,
		`
SELECT id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, last_used_at, user_agent
FROM sessions
WHERE id = $1
  AND user_id = $2
  AND revoked_at IS NULL
  AND expires_at > $3
`,
		sessionID,
		userID,
		now,
	))
	if err != nil {
		return model.Session{}, mapSessionError("get active session by id and user id", err)
	}

	return session, nil
}

func (r *SessionRepository) Rotate(ctx context.Context, params usecase.RotateSessionParams) (model.Session, error) {
	session, err := scanSession(executorFromContext(ctx, r.executor).QueryRow(
		ctx,
		`
UPDATE sessions
SET refresh_token_hash = $3,
    expires_at = $4,
    last_used_at = $5
WHERE id = $1
  AND refresh_token_hash = $2
  AND revoked_at IS NULL
  AND expires_at > $5
RETURNING id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, last_used_at, user_agent
`,
		params.SessionID,
		params.OldRefreshTokenHash,
		params.NewRefreshTokenHash,
		params.NewExpiresAt,
		params.LastUsedAt,
	))
	if err != nil {
		return model.Session{}, mapSessionError("rotate session", err)
	}

	return session, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `
UPDATE sessions
SET revoked_at = COALESCE(revoked_at, $2)
WHERE id = $1
	`, sessionID, revokedAt)
	if err != nil {
		return mapSessionError("revoke session", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke session: %w", usecase.ErrSessionNotFound)
	}

	return nil
}

func scanSession(row pgx.Row) (model.Session, error) {
	var (
		session     model.Session
		refreshHash []byte
		revokedAt   sql.NullTime
		userAgent   sql.NullString
	)

	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&refreshHash,
		&session.ExpiresAt,
		&revokedAt,
		&session.CreatedAt,
		&session.LastUsedAt,
		&userAgent,
	); err != nil {
		return model.Session{}, err
	}

	session.RefreshTokenHash = append([]byte(nil), refreshHash...)

	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	if userAgent.Valid {
		session.UserAgent = &userAgent.String
	}

	return session, nil
}

func mapSessionError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, usecase.ErrSessionNotFound)
	}

	return fmt.Errorf("%s: %w", operation, usecase.ErrUnexpectedStorage)
}
