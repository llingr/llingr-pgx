// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"fmt"
)

// Username is a concrete database username; the same role
// often has a different username in each environment.
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
