package orgs

import (
	"errors"
	"testing"
)

func TestValidateRole(t *testing.T) {
	if err := ValidateRole(RoleOwner); err != nil {
		t.Fatalf("expected owner to be valid, got %v", err)
	}
	if err := ValidateRole(RoleMember); err != nil {
		t.Fatalf("expected member to be valid, got %v", err)
	}
	for _, role := range []string{"", "admin", "Owner", "MEMBER", "owner "} {
		if err := ValidateRole(role); !errors.Is(err, ErrInvalidRole) {
			t.Fatalf("expected ErrInvalidRole for %q, got %v", role, err)
		}
	}
}
