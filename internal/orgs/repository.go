package orgs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCreateOrganization = errors.New("failed to create organization")
	ErrFindOrganization   = errors.New("failed to find organization")
	ErrAddMember          = errors.New("failed to add member")
	ErrFindMember         = errors.New("failed to find member")
	ErrListMemberships    = errors.New("failed to list memberships")
)

type OrganizationRepository struct {
	dbPool *pgxpool.Pool
}

func NewOrganizationRepository(dbPool *pgxpool.Pool) OrganizationRepository {
	return OrganizationRepository{dbPool: dbPool}
}

func (organizationRepository *OrganizationRepository) Create(ctx context.Context, name string) (*Organization, error) {
	var organization Organization
	err := organizationRepository.dbPool.QueryRow(
		ctx,
		"INSERT INTO organizations (id, name) VALUES ($1, $2) RETURNING id, name, created_at, updated_at",
		uuid.New(),
		name,
	).Scan(&organization.ID, &organization.Name, &organization.CreatedAt, &organization.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateOrganization, err)
	}
	return &organization, nil
}

func (organizationRepository *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organization, error) {
	var organization Organization
	err := organizationRepository.dbPool.QueryRow(
		ctx,
		"SELECT id, name, created_at, updated_at FROM organizations WHERE id=$1",
		id,
	).Scan(&organization.ID, &organization.Name, &organization.CreatedAt, &organization.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindOrganization, err)
	}
	return &organization, nil
}

func (organizationRepository *OrganizationRepository) AddMember(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, role string) error {
	_, err := organizationRepository.dbPool.Exec(
		ctx,
		"INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, $3)",
		organizationID,
		userID,
		role,
	)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAddMember, err)
	}
	return nil
}

func (organizationRepository *OrganizationRepository) FindMember(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID) (*Member, error) {
	var member Member
	err := organizationRepository.dbPool.QueryRow(
		ctx,
		"SELECT organization_id, user_id, role, created_at FROM organization_members WHERE organization_id=$1 AND user_id=$2",
		organizationID,
		userID,
	).Scan(&member.OrganizationID, &member.UserID, &member.Role, &member.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFindMember, err)
	}
	return &member, nil
}

func (organizationRepository *OrganizationRepository) ListMemberships(ctx context.Context, userID uuid.UUID) ([]Membership, error) {
	rows, err := organizationRepository.dbPool.Query(
		ctx,
		"SELECT o.id, o.name, o.created_at, o.updated_at, m.role FROM organizations o JOIN organization_members m ON m.organization_id = o.id WHERE m.user_id=$1 ORDER BY o.created_at",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListMemberships, err)
	}
	defer rows.Close()
	var memberships []Membership
	for rows.Next() {
		var membership Membership
		if err := rows.Scan(&membership.Organization.ID, &membership.Organization.Name, &membership.Organization.CreatedAt, &membership.Organization.UpdatedAt, &membership.Role); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrListMemberships, err)
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrListMemberships, err)
	}
	return memberships, nil
}
