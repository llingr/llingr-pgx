package roles_test

import (
	"strings"
	"testing"

	"github.com/llingr/llingr-pgx/roles"
)

// ValidatePlaceholderUsernames flags an invalid role key, not only an invalid
// username. The Builder's wrappers only ever produce valid role constants, so this
// branch is reachable only through a hand-built map: exactly the case Migrate guards.
func TestValidatePlaceholderUsernames_InvalidRoleKey(t *testing.T) {
	err := roles.ValidatePlaceholderUsernames(map[roles.Placeholder]roles.Username{
		"not a valid role": "valid_user",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid role") {
		t.Fatalf("expected an invalid-role error, got %v", err)
	}
}

// An empty map validates: a migration set may carry no :"name" placeholders, so the
// non-emptiness rule belongs to the Builder, not to this validator.
func TestValidatePlaceholderUsernames_EmptyMapOK(t *testing.T) {
	if err := roles.ValidatePlaceholderUsernames(nil); err != nil {
		t.Fatalf("empty map should validate, got %v", err)
	}
}
