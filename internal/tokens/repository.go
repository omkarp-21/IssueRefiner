package tokens

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"issue-refiner/internal/crypto"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSaveToken    = errors.New("failed to save token")
	ErrFindToken    = errors.New("failed to find token")
	ErrLegacyToken  = errors.New("github token needs reauthorization")
	ErrTokenCorrupt = errors.New("stored github token is corrupt")
)

const tokenVersionPrefix = "v1:"

type TokenRepository struct {
	dbPool *pgxpool.Pool
	key    [32]byte
}

func NewTokenRepository(dbPool *pgxpool.Pool, key [32]byte) TokenRepository {
	return TokenRepository{dbPool: dbPool, key: key}
}

func (tokenRepository *TokenRepository) Save(ctx context.Context, userID uuid.UUID, accessToken string) error {
	encrypted, err := crypto.Encrypt(accessToken, tokenRepository.key)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveToken, err)
	}
	_, err = tokenRepository.dbPool.Exec(
		ctx,
		"INSERT INTO github_tokens (user_id, access_token, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET access_token = EXCLUDED.access_token, updated_at = NOW()",
		userID,
		tokenVersionPrefix+encrypted,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveToken, err)
	}
	return nil
}

func (tokenRepository *TokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*GitHubToken, error) {
	var token GitHubToken
	err := tokenRepository.dbPool.QueryRow(
		ctx,
		"SELECT user_id, access_token, created_at, updated_at FROM github_tokens WHERE user_id=$1",
		userID,
	).Scan(&token.UserID, &token.AccessToken, &token.CreatedAt, &token.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindToken, err)
	}
	if !strings.HasPrefix(token.AccessToken, tokenVersionPrefix) {
		return nil, fmt.Errorf("%w", ErrLegacyToken)
	}
	plaintext, err := crypto.Decrypt(strings.TrimPrefix(token.AccessToken, tokenVersionPrefix), tokenRepository.key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTokenCorrupt, err)
	}
	token.AccessToken = plaintext
	return &token, nil
}
