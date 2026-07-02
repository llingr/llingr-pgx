package roles

import (
	"fmt"
)

// Username often different usernames for
// the same roles in different environments
type Username string

func (u Username) String() string {
	return string(u)
}

func (u Username) Validate() error {
	if !isPlainSQLIdentifier(u.String()) {
		const invalidUsernameError = "username %q is not a valid SQL identifier (letters, digits, underscore; max 63 bytes)"
		return fmt.Errorf(invalidUsernameError, u)
	}
	return nil
}
