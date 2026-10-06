package orgs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRole    = errors.New("invalid role")
	ErrNotMember      = errors.New("not a member of organization")
	ErrNoOrganization = errors.New("user has no organization")
)

type OrganizationService struct {
	OrganizationRepository *OrganizationRepository
}

func NewOrganizationService(organizationRepository *OrganizationRepository) OrganizationService {
	return OrganizationService{OrganizationRepository: organizationRepository}
}

func ValidateRole(role string) error {
	if role != RoleOwner && role != RoleMember {
		return fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	return nil
}

func (organizationService *OrganizationService) AddMember(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, role string) error {
	if err := ValidateRole(role); err != nil {
		return err
	}
	return organizationService.OrganizationRepository.AddMember(ctx, organizationID, userID, role)
}

func (organizationService *OrganizationService) EnsurePersonalOrg(ctx context.Context, userID uuid.UUID, name string) (*Organization, error) {
	memberships, err := organizationService.OrganizationRepository.ListMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(memberships) > 0 {
		return &memberships[0].Organization, nil
	}
	organization, err := organizationService.OrganizationRepository.Create(ctx, name)
	if err != nil {
		return nil, err
	}
	if err := organizationService.AddMember(ctx, organization.ID, userID, RoleOwner); err != nil {
		return nil, err
	}
	return organization, nil
}

func (organizationService *OrganizationService) ResolveOrg(ctx context.Context, userID uuid.UUID, hint *uuid.UUID) (*Organization, string, error) {
	if hint != nil {
		member, err := organizationService.OrganizationRepository.FindMember(ctx, *hint, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", fmt.Errorf("%w", ErrNotMember)
			}
			return nil, "", err
		}
		organization, err := organizationService.OrganizationRepository.GetByID(ctx, *hint)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", fmt.Errorf("%w", ErrNotMember)
			}
			return nil, "", err
		}
		return organization, member.Role, nil
	}
	memberships, err := organizationService.OrganizationRepository.ListMemberships(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if len(memberships) == 0 {
		return nil, "", fmt.Errorf("%w", ErrNoOrganization)
	}
	return &memberships[0].Organization, memberships[0].Role, nil
}
