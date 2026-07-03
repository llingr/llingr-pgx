package roles

import (
	"fmt"
)

// Placeholder names a role as referenced in migration files. Built-in:
//   - OwnerRole for migrations/DDL
//   - AppRole for application access
//
// Additional custom roles can be added as needed.
type Placeholder string

func (p Placeholder) String() string {
	return string(p)
}

func (p Placeholder) Validate() error {
	if !isPlainSQLIdentifier(p.String()) {
		const invalidRoleError = "role %q is not a valid SQL identifier (letters, digits, underscore; max 63 bytes)"
		return fmt.Errorf(invalidRoleError, p)
	}
	return nil
}
