package repos

import (
	"context"
	"errors"
	"fmt"
	"issue-refiner/internal/github"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateRepository   = errors.New("failed to create repository")
	ErrFindRepository     = errors.New("failed to find repository")
	ErrListRepositories   = errors.New("failed to list repositories")
	ErrRepositoryNotFound = errors.New("repository not found")
)

type RepositoryRepository struct {
	dbPool *pgxpool.Pool
}

func NewRepositoryRepository(dbPool *pgxpool.Pool) RepositoryRepository {
	return RepositoryRepository{dbPool: dbPool}
}

const repositoryColumns = "id, organization_id, github_id, owner, name, full_name, description, default_branch, html_url, private, created_at, updated_at, synced_at"

func stringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (repositoryRepository *RepositoryRepository) Upsert(ctx context.Context, organizationID uuid.UUID, remote *github.Repository) (*Repository, error) {
	var repository Repository
	err := repositoryRepository.dbPool.QueryRow(
		ctx,
		`INSERT INTO repositories (
		id,
		organization_id,
		github_id,
		owner,
		name,
		full_name,
		description,
		default_branch,
		html_url,
		private,
		synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (organization_id, github_id) DO UPDATE SET
			owner = EXCLUDED.owner,
			name = EXCLUDED.name,
			full_name = EXCLUDED.full_name,
			description = EXCLUDED.description,
			default_branch = EXCLUDED.default_branch,
			html_url = EXCLUDED.html_url,
			private = EXCLUDED.private,
			updated_at = NOW(),
			synced_at = NOW()
		RETURNING `+repositoryColumns,
		uuid.New(),
		organizationID,
		remote.ID,
		remote.Owner,
		remote.Name,
		remote.FullName,
		stringOrNil(remote.Description),
		remote.DefaultBranch,
		remote.HTMLURL,
		remote.Private,
	).Scan(&repository.ID,
		&repository.OrganizationID,
		&repository.GitHubID,
		&repository.Owner,
		&repository.Name,
		&repository.FullName,
		&repository.Description,
		&repository.DefaultBranch,
		&repository.HTMLURL,
		&repository.Private,
		&repository.CreatedAt,
		&repository.UpdatedAt,
		&repository.SyncedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateRepository, err)
	}
	return &repository, nil
}

func (repositoryRepository *RepositoryRepository) ListForOrg(ctx context.Context, organizationID uuid.UUID) ([]Repository, error) {
	rows, err := repositoryRepository.dbPool.Query(
		ctx,
		"SELECT "+repositoryColumns+" FROM repositories WHERE organization_id=$1 ORDER BY full_name",
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListRepositories, err)
	}
	defer rows.Close()
	repositories := []Repository{}
	for rows.Next() {
		var repository Repository
		if err := rows.Scan(&repository.ID,
			&repository.OrganizationID,
			&repository.GitHubID,
			&repository.Owner,
			&repository.Name,
			&repository.FullName,
			&repository.Description,
			&repository.DefaultBranch,
			&repository.HTMLURL,
			&repository.Private,
			&repository.CreatedAt,
			&repository.UpdatedAt,
			&repository.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListRepositories, err)
		}
		repositories = append(repositories, repository)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListRepositories, err)
	}
	return repositories, nil
}

func (repositoryRepository *RepositoryRepository) FindForOrg(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (*Repository, error) {
	var repository Repository
	err := repositoryRepository.dbPool.QueryRow(
		ctx,
		"SELECT "+repositoryColumns+" FROM repositories WHERE id=$1 AND organization_id=$2",
		id,
		organizationID,
	).Scan(&repository.ID,
		&repository.OrganizationID,
		&repository.GitHubID,
		&repository.Owner,
		&repository.Name,
		&repository.FullName,
		&repository.Description,
		&repository.DefaultBranch,
		&repository.HTMLURL,
		&repository.Private,
		&repository.CreatedAt,
		&repository.UpdatedAt,
		&repository.SyncedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindRepository, err)
	}
	return &repository, nil
}

func (repositoryRepository *RepositoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*Repository, error) {
	var repository Repository
	err := repositoryRepository.dbPool.QueryRow(
		ctx,
		"SELECT "+repositoryColumns+" FROM repositories WHERE id=$1",
		id,
	).Scan(&repository.ID,
		&repository.OrganizationID,
		&repository.GitHubID,
		&repository.Owner,
		&repository.Name,
		&repository.FullName,
		&repository.Description,
		&repository.DefaultBranch,
		&repository.HTMLURL,
		&repository.Private,
		&repository.CreatedAt,
		&repository.UpdatedAt,
		&repository.SyncedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindRepository, err)
	}
	return &repository, nil
}
