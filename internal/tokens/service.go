package tokens

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNoToken = errors.New("github account not connected")
)

type TokenService struct {
	TokenRepository *TokenRepository
}

func NewTokenService(tokenRepository *TokenRepository) TokenService {
	return TokenService{TokenRepository: tokenRepository}
}

func (tokenService *TokenService) Save(ctx context.Context, userID uuid.UUID, accessToken string) error {
	return tokenService.TokenRepository.Save(ctx, userID, accessToken)
}

func (tokenService *TokenService) Get(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := tokenService.TokenRepository.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%w", ErrNoToken)
		}
		return "", err
	}
	return token.AccessToken, nil
}
