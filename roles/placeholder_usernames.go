package roles

// PlaceholderUsernames maps each role placeholder to the username bound to it for
// the current environment. These are substituted in migrations.
type PlaceholderUsernames map[Placeholder]Username

// AdminOwnerUsername returns the username bound to AdminOwnerRole, or "" if none was set
func (p PlaceholderUsernames) AdminOwnerUsername() string {
	return p.UsernameFor(AdminOwnerRole)
}

// AppUsername returns the username bound to AppRole, or "" if none was set
func (p PlaceholderUsernames) AppUsername() string {
	return p.UsernameFor(AppRole)
}

// UsernameFor returns the username for the supplied role, or "" if none was set
func (p PlaceholderUsernames) UsernameFor(role Placeholder) string {
	return p[role].String()
}
