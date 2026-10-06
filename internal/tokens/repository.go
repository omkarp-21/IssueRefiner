package tokens

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSaveToken = errors.New("failed to save token")
	ErrFindToken = errors.New("failed to find token")
)

type TokenRepository struct {
	dbPool *pgxpool.Pool
}

func NewTokenRepository(dbPool *pgxpool.Pool) TokenRepository {
	return TokenRepository{dbPool: dbPool}
}

func (tokenRepository *TokenRepository) Save(ctx context.Context, userID uuid.UUID, accessToken string) error {
	_, err := tokenRepository.dbPool.Exec(
		ctx,
		"INSERT INTO github_tokens (user_id, access_token, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (user_id) DO UPDATE SET access_token = EXCLUDED.access_token, updated_at = NOW()",
		userID,
		accessToken,
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
	return &token, nil
}
