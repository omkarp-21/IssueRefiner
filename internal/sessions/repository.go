package sessions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateSession = errors.New("failed to create session")
	ErrFindSession   = errors.New("failed to find session")
	ErrUpdateSession = errors.New("failed to update session")
)

type SessionRepository struct {
	dbPool *pgxpool.Pool
}

func NewSessionRepository(dbPool *pgxpool.Pool) SessionRepository {
	return SessionRepository{dbPool: dbPool}
}

func (sessionRepository *SessionRepository) Create(ctx context.Context, userID uuid.UUID, organizationID *uuid.UUID, tokenHash string, expiresAt time.Time) (*Session, error) {
	var session Session
	err := sessionRepository.dbPool.QueryRow(
		ctx,
		"INSERT INTO sessions (id, token_hash, user_id, organization_id, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, token_hash, user_id, organization_id, expires_at, created_at",
		uuid.New(),
		tokenHash,
		userID,
		organizationID,
		expiresAt,
	).Scan(&session.ID, &session.TokenHash, &session.UserID, &session.OrganizationID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateSession, err)
	}
	return &session, nil
}

func (sessionRepository *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	var session Session
	err := sessionRepository.dbPool.QueryRow(
		ctx,
		"SELECT id, token_hash, user_id, organization_id, expires_at, created_at FROM sessions WHERE token_hash=$1",
		tokenHash,
	).Scan(&session.ID, &session.TokenHash, &session.UserID, &session.OrganizationID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindSession, err)
	}
	return &session, nil
}

func (sessionRepository *SessionRepository) SetOrganization(ctx context.Context, sessionID uuid.UUID, organizationID uuid.UUID) error {
	_, err := sessionRepository.dbPool.Exec(
		ctx,
		"UPDATE sessions SET organization_id=$1 WHERE id=$2",
		organizationID,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpdateSession, err)
	}
	return nil
}
