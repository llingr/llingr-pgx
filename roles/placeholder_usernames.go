package roles

// PlaceholderUsernames maps each role placeholder to the username bound to it for
// the current environment. These are substituted in migrations.
type PlaceholderUsernames map[Placeholder]Username

// OwnerUsername returns the username bound to OwnerRole, or "" if none was set
func (p PlaceholderUsernames) OwnerUsername() string {
	return p.UsernameFor(OwnerRole)
}

// AppUsername returns the username bound to AppRole, or "" if none was set
func (p PlaceholderUsernames) AppUsername() string {
	return p.UsernameFor(AppRole)
}

// UsernameFor returns the username for the supplied role, or "" if none was set
func (p PlaceholderUsernames) UsernameFor(role Placeholder) string {
	return p[role].String()
}
