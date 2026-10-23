// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

// DefaultMigrationsTable is golang-migrate's default version table
const DefaultMigrationsTable = "schema_migrations"

// Option for configuring a Migrate call
type Option func(*migrateOptions)

// migrateOptions with resolved
// configuration for a Migrate call
type migrateOptions struct {
	// FilesystemDirectory is the relative
	// subdirectory within migration's fs.FS
	FilesystemDirectory string

	// MigrationsTable records the
	// applied migration version
	MigrationsTable string
}

// WithFilesystemDirectory sets the sub-path
// within the embedded fs.FS SQL migrations live
func WithFilesystemDirectory(directory string) Option {
	return func(options *migrateOptions) {
		options.FilesystemDirectory = directory
	}
}

// WithMigrationsTable records the applied version in its own table, so a library's
// migrations and the application's can share a database. Must be a plain SQL identifier.
//
// golang-migrate quotes the name, so capitals are kept and must be quoted
// in any SQL naming the table; prefer lowercase.
//
// Moving existing migrations off DefaultMigrationsTable re-runs them from version 0.
// Before the first run with the new table, create that table, with columns
// (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL), and insert the
// current version with dirty false.
func WithMigrationsTable(table string) Option {
	return func(options *migrateOptions) {
		options.MigrationsTable = table
	}
}

// processOptions applies each Option in order
func processOptions(options ...Option) migrateOptions {
	resolved := migrateOptions{
		// default for sql migration files in root-directory
		FilesystemDirectory: ".",
		MigrationsTable:     DefaultMigrationsTable,
	}

	for _, option := range options {
		option(&resolved)
	}
	return resolved
}
