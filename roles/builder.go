package roles

import (
	"errors"
	"fmt"
	"maps"
)

// Builder creates placeholder <-> username mappings and
// checks both are plain SQL identifiers.
type Builder struct {
	placeholderUsernames PlaceholderUsernames
}

// NewPlaceholderBuilder builder for placeholder -> username map
func NewPlaceholderBuilder() *Builder {
	return &Builder{
		placeholderUsernames: map[Placeholder]Username{},
	}
}

// WithAdminOwner for COMPLETE schema access, only to be used
// for DDL database changes (migrations).
func (b *Builder) WithAdminOwner(username Username) *Builder {
	return b.WithCustom(AdminOwnerRole, username)
}

// WithApp for typical read/write application access WITHOUT
// privileges to alter the schema (mitigates injection attacks
// dropping tables).
func (b *Builder) WithApp(username Username) *Builder {
	return b.WithCustom(AppRole, username)
}

// WithCustom for custom schema access, for example a read-only user
func (b *Builder) WithCustom(role Placeholder, username Username) *Builder {
	b.placeholderUsernames[role] = username
	return b
}

// Build validates and returns placeholder/username map, failing
// if any username is not a plaintext SQL identifier
func (b *Builder) Build() (PlaceholderUsernames, error) {
	const noRolesError = "no roles added to %T"

	if len(b.placeholderUsernames) == 0 {
		return nil, fmt.Errorf(noRolesError, b)
	}
	if err := ValidatePlaceholderUsernames(b.placeholderUsernames); err != nil {
		return nil, err
	}
	return maps.Clone(b.placeholderUsernames), nil
}

// ValidatePlaceholderUsernames reports whether every placeholder and its mapped
// username is a plain SQL identifier, joining all failures. Unlike Build it does
// not require the map to be non-empty: a migration set with no :"name" placeholders
// legitimately needs no roles, and the Builder enforces non-emptiness separately.
// schema.Migrate calls this so a map assembled by hand (bypassing the Builder, and
// so its validation) is still checked before any username reaches the SQL.
func ValidatePlaceholderUsernames(placeholderUsernames PlaceholderUsernames) error {
	const (
		invalidUsername = "invalid username for role %q: %w"
		invalidRole     = "invalid role %q: %w"
	)

	var failures []error
	for role, username := range placeholderUsernames {
		if err := role.Validate(); err != nil {
			failures = append(failures, fmt.Errorf(invalidRole, role, err))
		}
		if err := username.Validate(); err != nil {
			failures = append(failures, fmt.Errorf(invalidUsername, role, err))
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	return nil
}

// MustBuild for fail-fast wiring at startup
func (b *Builder) MustBuild() PlaceholderUsernames {
	placeholderUsernames, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("roles: invalid role set: %v", err))
	}
	return placeholderUsernames
}
