package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateProviderConfig = errors.New("failed to create provider config")
	ErrFindProviderConfig   = errors.New("failed to find provider config")
	ErrListProviderConfigs  = errors.New("failed to list provider configs")
	ErrUpdateProviderConfig = errors.New("failed to update provider config")
	ErrDeleteProviderConfig = errors.New("failed to delete provider config")
	ErrSaveProviderModels   = errors.New("failed to save provider models")
)

type ConfigRepository struct {
	dbPool *pgxpool.Pool
}

func NewConfigRepository(dbPool *pgxpool.Pool) ConfigRepository {
	return ConfigRepository{dbPool: dbPool}
}

const configColumns = "id, organization_id, provider_type, display_name, enabled, status, last_validated_at, last_error, created_at, updated_at"

func (configRepository *ConfigRepository) CreateFull(ctx context.Context, organizationID uuid.UUID, providerType string, displayName string, models []string) (*ProviderConfig, error) {
	tx, err := configRepository.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateProviderConfig, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var config ProviderConfig
	err = tx.QueryRow(
		ctx,
		"INSERT INTO provider_configs (id, organization_id, provider_type, display_name) VALUES ($1, $2, $3, $4) RETURNING "+configColumns,
		uuid.New(),
		organizationID,
		providerType,
		displayName,
	).Scan(&config.ID,
		&config.OrganizationID,
		&config.ProviderType,
		&config.DisplayName,
		&config.Enabled,
		&config.Status,
		&config.LastValidatedAt,
		&config.LastError,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateProviderConfig, err)
	}
	for _, model := range models {
		if _, err := tx.Exec(ctx, "INSERT INTO provider_models (id, provider_config_id, model) VALUES ($1, $2, $3)", uuid.New(), config.ID, model); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateProviderConfig, err)
	}
	return &config, nil
}

func (configRepository *ConfigRepository) ReplaceModels(ctx context.Context, configID uuid.UUID, models []string) error {
	tx, err := configRepository.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "DELETE FROM provider_models WHERE provider_config_id=$1", configID); err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	for _, model := range models {
		if _, err := tx.Exec(ctx, "INSERT INTO provider_models (id, provider_config_id, model) VALUES ($1, $2, $3)", uuid.New(), configID, model); err != nil {
			return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	return nil
}

func (configRepository *ConfigRepository) SyncModels(ctx context.Context, configID uuid.UUID, models []string) error {
	tx, err := configRepository.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, model := range models {
		if _, err := tx.Exec(ctx, "INSERT INTO provider_models (id, provider_config_id, model) VALUES ($1, $2, $3) ON CONFLICT (provider_config_id, model) DO NOTHING", uuid.New(), configID, model); err != nil {
			return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM provider_models WHERE provider_config_id=$1 AND NOT (model = ANY($2))", configID, models); err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrSaveProviderModels, err)
	}
	return nil
}

func (configRepository *ConfigRepository) ListForOrg(ctx context.Context, organizationID uuid.UUID) ([]ProviderConfig, error) {
	rows, err := configRepository.dbPool.Query(ctx, "SELECT "+configColumns+" FROM provider_configs WHERE organization_id=$1 ORDER BY created_at", organizationID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
	}
	defer rows.Close()
	configs := []ProviderConfig{}
	for rows.Next() {
		var config ProviderConfig
		if err := rows.Scan(&config.ID,
			&config.OrganizationID,
			&config.ProviderType,
			&config.DisplayName,
			&config.Enabled,
			&config.Status,
			&config.LastValidatedAt,
			&config.LastError,
			&config.CreatedAt,
			&config.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
		}
		configs = append(configs, config)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
	}
	return configs, nil
}

func (configRepository *ConfigRepository) FindForOrg(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*ProviderConfig, error) {
	var config ProviderConfig
	err := configRepository.dbPool.QueryRow(
		ctx,
		"SELECT "+configColumns+" FROM provider_configs WHERE id=$1 AND organization_id=$2",
		id,
		organizationID,
	).Scan(&config.ID,
		&config.OrganizationID,
		&config.ProviderType,
		&config.DisplayName,
		&config.Enabled,
		&config.Status,
		&config.LastValidatedAt,
		&config.LastError,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindProviderConfig, err)
	}
	return &config, nil
}

func (configRepository *ConfigRepository) ListModels(ctx context.Context, organizationID uuid.UUID, configID uuid.UUID) ([]ProviderModel, error) {
	rows, err := configRepository.dbPool.Query(
		ctx,
		"SELECT m.id, m.provider_config_id, m.model, m.enabled, m.created_at FROM provider_models m JOIN provider_configs c ON c.id = m.provider_config_id WHERE m.provider_config_id=$1 AND c.organization_id=$2 ORDER BY m.model",
		configID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
	}
	defer rows.Close()
	models := []ProviderModel{}
	for rows.Next() {
		var model ProviderModel
		if err := rows.Scan(&model.ID, &model.ProviderConfigID, &model.Model, &model.Enabled, &model.CreatedAt); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListProviderConfigs, err)
	}
	return models, nil
}

func (configRepository *ConfigRepository) IsModelEnabled(ctx context.Context, organizationID uuid.UUID, configID uuid.UUID, model string) (bool, error) {
	var enabled bool
	err := configRepository.dbPool.QueryRow(
		ctx,
		"SELECT m.enabled FROM provider_models m JOIN provider_configs c ON c.id = m.provider_config_id WHERE m.provider_config_id=$1 AND c.organization_id=$2 AND m.model=$3",
		configID,
		organizationID,
		model,
	).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%w: %w", ErrFindProviderConfig, err)
	}
	return enabled, nil
}

func (configRepository *ConfigRepository) UpdateDetails(ctx context.Context, organizationID uuid.UUID, id uuid.UUID, displayName string, enabled bool) error {
	_, err := configRepository.dbPool.Exec(
		ctx,
		"UPDATE provider_configs SET display_name=$1, enabled=$2, updated_at=NOW() WHERE id=$3 AND organization_id=$4",
		displayName,
		enabled,
		id,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpdateProviderConfig, err)
	}
	return nil
}

func (configRepository *ConfigRepository) SetStatus(ctx context.Context, id uuid.UUID, status string, lastError *string, validated bool) error {
	if validated {
		_, err := configRepository.dbPool.Exec(ctx, "UPDATE provider_configs SET status=$1, last_error=$2, last_validated_at=NOW(), updated_at=NOW() WHERE id=$3", status, lastError, id)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrUpdateProviderConfig, err)
		}
		return nil
	}
	_, err := configRepository.dbPool.Exec(ctx, "UPDATE provider_configs SET status=$1, last_error=$2, updated_at=NOW() WHERE id=$3", status, lastError, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUpdateProviderConfig, err)
	}
	return nil
}

func (configRepository *ConfigRepository) Delete(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) error {
	_, err := configRepository.dbPool.Exec(ctx, "DELETE FROM provider_configs WHERE id=$1 AND organization_id=$2", id, organizationID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeleteProviderConfig, err)
	}
	return nil
}
