// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
)

// With no options, processOptions returns the defaults: migrations are read from
// the root of the embedded fs.FS.
func TestProcessOptions_DefaultsToRootDirectory(t *testing.T) {
	resolved := processOptions()
	if resolved.FilesystemDirectory != "." {
		t.Errorf("default FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, ".")
	}
}

// With no options, the default table is unchanged for existing deployments.
func TestProcessOptions_DefaultsToSchemaMigrationsTable(t *testing.T) {
	resolved := processOptions()
	if resolved.MigrationsTable != "schema_migrations" {
		t.Errorf("default MigrationsTable = %q, want %q", resolved.MigrationsTable, "schema_migrations")
	}
	if DefaultMigrationsTable != "schema_migrations" {
		t.Errorf("DefaultMigrationsTable = %q, want %q", DefaultMigrationsTable, "schema_migrations")
	}
}

// Upstream exposes its default only as a var, so the const is a copy; this
// catches golang-migrate changing its default under existing deployments.
func TestDefaultMigrationsTable_MatchesGolangMigrate(t *testing.T) {
	if DefaultMigrationsTable != migratepgx.DefaultMigrationsTable {
		t.Errorf("DefaultMigrationsTable = %q, golang-migrate default = %q",
			DefaultMigrationsTable, migratepgx.DefaultMigrationsTable)
	}
}

// WithMigrationsTable overrides the default table and leaves the directory alone.
func TestWithMigrationsTable_Overrides(t *testing.T) {
	resolved := processOptions(WithMigrationsTable("event_store_schema_migrations"))
	if resolved.MigrationsTable != "event_store_schema_migrations" {
		t.Errorf("MigrationsTable = %q, want %q", resolved.MigrationsTable, "event_store_schema_migrations")
	}
	if resolved.FilesystemDirectory != "." {
		t.Errorf("FilesystemDirectory = %q, want the default %q", resolved.FilesystemDirectory, ".")
	}
}

// WithFilesystemDirectory overrides the default sub-path.
func TestWithFilesystemDirectory_Overrides(t *testing.T) {
	resolved := processOptions(WithFilesystemDirectory("migrations"))
	if resolved.FilesystemDirectory != "migrations" {
		t.Errorf("FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, "migrations")
	}
}

// Options apply in order, so a later option wins over an earlier one.
func TestProcessOptions_LastOptionWins(t *testing.T) {
	resolved := processOptions(
		WithFilesystemDirectory("first"),
		WithFilesystemDirectory("second"),
	)
	if resolved.FilesystemDirectory != "second" {
		t.Errorf("FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, "second")
	}
}
