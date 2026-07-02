package roles

import "regexp"

// common roles
const (
	AdminOwnerRole Placeholder = "admin_owner" // for startup/migrations DDL, NOT for runtime Data Manipulation
	AppRole        Placeholder = "app"         // typical runtime DML user, NOT for startup Data Definition
)

// PlainSQLIdentifierRegex mitigates placeholder injection
const PlainSQLIdentifierRegex = `^[A-Za-z_][A-Za-z0-9_]*$`

// MaxIdentifierBytes is Postgres's identifier length limit (NAMEDATALEN-1).
// The server silently truncates longer identifiers, so a GRANT would target a
// truncated name; they are rejected at validation instead.
const MaxIdentifierBytes = 63

// plainSQLIdentifier is PlainSQLIdentifierRegex compiled once, reused by every check.
var plainSQLIdentifier = regexp.MustCompile(PlainSQLIdentifierRegex)

// isPlainSQLIdentifier reports whether s is a plain SQL identifier within
// Postgres's length limit, mitigating placeholder injection and silent
// server-side truncation.
func isPlainSQLIdentifier(s string) bool {
	if s == "" || len(s) > MaxIdentifierBytes {
		return false
	}
	return plainSQLIdentifier.MatchString(s)
}
