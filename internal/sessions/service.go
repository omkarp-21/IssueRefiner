package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrGenerateToken   = errors.New("failed to generate session token")
)

type SessionService struct {
	SessionRepository *SessionRepository
}

func NewSessionService(sessionRepository *SessionRepository) SessionService {
	return SessionService{SessionRepository: sessionRepository}
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("%w: %w", ErrGenerateToken, err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func (sessionService *SessionService) Create(ctx context.Context, userID uuid.UUID, organizationID uuid.UUID) (string, *Session, error) {
	token, err := newToken()
	if err != nil {
		return "", nil, err
	}
	session, err := sessionService.SessionRepository.Create(ctx, userID, &organizationID, HashToken(token), time.Now().Add(7*24*time.Hour))
	if err != nil {
		return "", nil, err
	}
	return token, session, nil
}

func (sessionService *SessionService) SetOrganization(ctx context.Context, sessionID uuid.UUID, organizationID uuid.UUID) error {
	return sessionService.SessionRepository.SetOrganization(ctx, sessionID, organizationID)
}

func (sessionService *SessionService) Delete(ctx context.Context, sessionID uuid.UUID) error {
	return sessionService.SessionRepository.Delete(ctx, sessionID)
}

func (sessionService *SessionService) GetSession(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}
	session, err := sessionService.SessionRepository.FindByTokenHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", ErrSessionNotFound)
		}
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("%w", ErrSessionExpired)
	}
	return session, nil
}
