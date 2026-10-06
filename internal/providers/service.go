package providers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidConfig   = errors.New("invalid provider config")
	ErrConfigNotFound  = errors.New("provider config not found")
	ErrConfigDisabled  = errors.New("provider config disabled")
	ErrModelNotEnabled = errors.New("model not enabled")
	ErrConfigInUse     = errors.New("provider config is in use")
)

type ProviderService struct {
	Configs     *ConfigRepository
	Credentials *CredentialStore
	Registry    *Registry
}

func NewProviderService(configs *ConfigRepository, credentials *CredentialStore, registry *Registry) ProviderService {
	return ProviderService{Configs: configs, Credentials: credentials, Registry: registry}
}

func (providerService *ProviderService) CreateConfig(ctx context.Context, organizationID uuid.UUID, providerType string, displayName string, apiKey string, models []string) (*ProviderConfig, error) {
	if err := ValidateProviderType(providerType); err != nil {
		return nil, err
	}
	if displayName == "" {
		return nil, fmt.Errorf("%w: display name is required", ErrInvalidConfig)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("%w: at least one model is required", ErrInvalidConfig)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%w: api key is required", ErrInvalidConfig)
	}
	adapter, err := providerService.Registry.Get(providerType)
	if err != nil {
		return nil, err
	}
	if err := providerService.checkModels(ctx, adapter, apiKey, models); err != nil {
		return nil, err
	}
	if err := adapter.ValidateCredentials(ctx, apiKey, models[0]); err != nil {
		return nil, err
	}
	config, err := providerService.Configs.CreateFull(ctx, organizationID, providerType, displayName, models)
	if err != nil {
		return nil, err
	}
	if err := providerService.Credentials.Save(ctx, config.ID, apiKey); err != nil {
		return nil, err
	}
	if err := providerService.Configs.SetStatus(ctx, config.ID, StatusHealthy, nil, true); err != nil {
		return nil, err
	}
	return providerService.getConfig(ctx, organizationID, config.ID)
}

func (providerService *ProviderService) ListConfigs(ctx context.Context, organizationID uuid.UUID) ([]ProviderConfig, error) {
	return providerService.Configs.ListForOrg(ctx, organizationID)
}

func (providerService *ProviderService) checkModels(ctx context.Context, adapter Provider, apiKey string, models []string) error {
	discoverer, ok := adapter.(ModelDiscoverer)
	if !ok {
		for _, model := range models {
			if !adapter.SupportsModel(model) {
				return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
			}
		}
		return nil
	}
	discovered, err := discoverer.DiscoverModels(ctx, apiKey)
	if err != nil {
		return err
	}
	for _, model := range models {
		if !containsModel(discovered, model) {
			return fmt.Errorf("%w: %s", ErrModelUnavailable, model)
		}
	}
	return nil
}

func containsModel(models []string, model string) bool {
	for _, item := range models {
		if item == model {
			return true
		}
	}
	return false
}

func (providerService *ProviderService) GetConfig(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*ProviderConfig, []ProviderModel, error) {
	config, err := providerService.getConfig(ctx, organizationID, id)
	if err != nil {
		return nil, nil, err
	}
	models, err := providerService.Configs.ListModels(ctx, organizationID, id)
	if err != nil {
		return nil, nil, err
	}
	return config, models, nil
}

func (providerService *ProviderService) getConfig(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*ProviderConfig, error) {
	config, err := providerService.Configs.FindForOrg(ctx, organizationID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w", ErrConfigNotFound)
		}
		return nil, err
	}
	return config, nil
}

func (providerService *ProviderService) UpdateConfig(ctx context.Context, organizationID uuid.UUID, id uuid.UUID, displayName string, enabled bool, apiKey string, models []string, changeModels bool) (*ProviderConfig, error) {
	config, err := providerService.getConfig(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if displayName == "" {
		return nil, fmt.Errorf("%w: display name is required", ErrInvalidConfig)
	}
	adapter, err := providerService.Registry.Get(config.ProviderType)
	if err != nil {
		return nil, err
	}
	effectiveModels := models
	if !changeModels {
		existing, err := providerService.Configs.ListModels(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
		for _, model := range existing {
			effectiveModels = append(effectiveModels, model.Model)
		}
	}
	if len(effectiveModels) == 0 {
		return nil, fmt.Errorf("%w: at least one model is required", ErrInvalidConfig)
	}
	key := apiKey
	if key == "" {
		var err error
		key, err = providerService.Credentials.Load(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	if apiKey != "" || changeModels {
		if err := providerService.checkModels(ctx, adapter, key, effectiveModels); err != nil {
			providerService.noteStatus(ctx, id, err)
			return nil, err
		}
		if err := adapter.ValidateCredentials(ctx, key, effectiveModels[0]); err != nil {
			providerService.noteStatus(ctx, id, err)
			return nil, err
		}
	}
	if err := providerService.Configs.UpdateDetails(ctx, organizationID, id, displayName, enabled); err != nil {
		return nil, err
	}
	if changeModels {
		if err := providerService.Configs.ReplaceModels(ctx, id, models); err != nil {
			return nil, err
		}
	}
	if apiKey != "" {
		if err := providerService.Credentials.Save(ctx, id, apiKey); err != nil {
			return nil, err
		}
	}
	if apiKey != "" || changeModels {
		if err := providerService.Configs.SetStatus(ctx, id, StatusHealthy, nil, true); err != nil {
			return nil, err
		}
	}
	return providerService.getConfig(ctx, organizationID, id)
}

func (providerService *ProviderService) DeleteConfig(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) error {
	if _, err := providerService.getConfig(ctx, organizationID, id); err != nil {
		return err
	}
	if err := providerService.Configs.Delete(ctx, organizationID, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w", ErrConfigInUse)
		}
		return err
	}
	return nil
}

func (providerService *ProviderService) TestConfig(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*ProviderConfig, error) {
	config, models, err := providerService.GetConfig(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("%w: no models configured", ErrInvalidConfig)
	}
	adapter, err := providerService.Registry.Get(config.ProviderType)
	if err != nil {
		return nil, err
	}
	key, err := providerService.Credentials.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if discoverer, ok := adapter.(ModelDiscoverer); ok {
		discovered, err := discoverer.DiscoverModels(ctx, key)
		if err != nil {
			providerService.noteStatus(ctx, id, err)
			return nil, err
		}
		if len(discovered) == 0 {
			return nil, fmt.Errorf("%w: no usable models discovered", ErrModelUnavailable)
		}
		if err := providerService.Configs.SyncModels(ctx, id, discovered); err != nil {
			return nil, err
		}
		models, err = providerService.Configs.ListModels(ctx, organizationID, id)
		if err != nil {
			return nil, err
		}
	}
	if err := adapter.ValidateCredentials(ctx, key, models[0].Model); err != nil {
		providerService.noteStatus(ctx, id, err)
		return nil, err
	}
	if err := providerService.Configs.SetStatus(ctx, id, StatusHealthy, nil, true); err != nil {
		return nil, err
	}
	return providerService.getConfig(ctx, organizationID, id)
}

func (providerService *ProviderService) ResolveForJob(ctx context.Context, organizationID uuid.UUID, configID uuid.UUID, model string) (*ProviderConfig, error) {
	config, err := providerService.getConfig(ctx, organizationID, configID)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, fmt.Errorf("%w", ErrConfigDisabled)
	}
	enabled, err := providerService.Configs.IsModelEnabled(ctx, organizationID, configID, model)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, fmt.Errorf("%w: %s", ErrModelNotEnabled, model)
	}
	return config, nil
}

func (providerService *ProviderService) GenerateWithConfig(ctx context.Context, configID *uuid.UUID, providerType string, model string, input Input) (Result, error) {
	adapter, err := providerService.Registry.Get(providerType)
	if err != nil {
		return Result{}, err
	}
	var credential string
	if configID != nil {
		credential, err = providerService.Credentials.Load(ctx, *configID)
		if err != nil {
			return Result{}, fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
		}
	}
	result, err := adapter.Generate(ctx, credential, model, input)
	if err != nil {
		if configID != nil {
			providerService.noteStatus(ctx, *configID, err)
		}
		return Result{}, err
	}
	if configID != nil {
		_ = providerService.Configs.SetStatus(ctx, *configID, StatusHealthy, nil, false)
	}
	return result, nil
}

func (providerService *ProviderService) noteStatus(ctx context.Context, configID uuid.UUID, err error) {
	category := "unknown"
	var providerError *ProviderError
	if errors.As(err, &providerError) {
		category = providerError.Category.Error()
	}
	_ = providerService.Configs.SetStatus(ctx, configID, StatusForError(err), &category, false)
}
