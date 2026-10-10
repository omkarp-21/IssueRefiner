package providers

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/crypto"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSaveCredential = errors.New("failed to save credential")
	ErrLoadCredential = errors.New("failed to load credential")
	ErrBadCiphertext  = errors.New("invalid encrypted credential")
)

func EncryptAPIKey(plaintext string, key [32]byte) (string, error) {
	encrypted, err := crypto.Encrypt(plaintext, key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSaveCredential, err)
	}
	return encrypted, nil
}

func DecryptAPIKey(cipherHex string, key [32]byte) (string, error) {
	plaintext, err := crypto.Decrypt(cipherHex, key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBadCiphertext, err)
	}
	return plaintext, nil
}

type CredentialStore struct {
	dbPool *pgxpool.Pool
	key    [32]byte
}

func NewCredentialStore(dbPool *pgxpool.Pool, key [32]byte) CredentialStore {
	return CredentialStore{dbPool: dbPool, key: key}
}

func (credentialStore *CredentialStore) Save(ctx context.Context, configID uuid.UUID, apiKey string) error {
	encrypted, err := EncryptAPIKey(apiKey, credentialStore.key)
	if err != nil {
		return err
	}
	_, err = credentialStore.dbPool.Exec(
		ctx,
		"INSERT INTO provider_credentials (provider_config_id, api_key_encrypted, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (provider_config_id) DO UPDATE SET api_key_encrypted = EXCLUDED.api_key_encrypted, updated_at = NOW()",
		configID,
		encrypted,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveCredential, err)
	}
	return nil
}

func (credentialStore *CredentialStore) Load(ctx context.Context, configID uuid.UUID) (string, error) {
	var encrypted string
	err := credentialStore.dbPool.QueryRow(ctx, "SELECT api_key_encrypted FROM provider_credentials WHERE provider_config_id=$1", configID).Scan(&encrypted)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrLoadCredential, err)
	}
	apiKey, err := DecryptAPIKey(encrypted, credentialStore.key)
	if err != nil {
		return "", err
	}
	return apiKey, nil
}
