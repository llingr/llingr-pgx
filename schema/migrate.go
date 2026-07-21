// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

// Package schema applies ordered SQL migrations from an embedded filesystem
// (golang-migrate using pgx/v5 driver), substituting role-username placeholders
// into each file before each migration file is applied.
//
// Migrations use the conventional NNN_name.up.sql naming. The migrations library
// does not wrap a migration in a transaction, so each file must wrap changes
// using BEGIN TRANSACTION; ... COMMIT TRANSACTION; for atomicity.
//
// Placeholder substitution is textual: unlike psql's own interpolation, it does not
// skip string literals or comments, so a literal ':"name"' anywhere in a migration
// will be substituted (or rejected as unknown).
package schema

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/llingr/llingr-pgx/roles"
)

// Migrate applies the pending up-migrations embedded in fsys. On failure the
// error names the version the schema stopped (and is left dirty) at, when
// golang-migrate can report it.
//
// pool must authenticate as a role permitted to run DDL; the caller assumes this
// During migration the roleUsernames are substituted for the :"name" placeholders;
// an unmatched placeholder is a hard error.
//
// By default, migrations are read from the root of fsys; override using WithFilesystemDirectory.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, roleUsernames roles.PlaceholderUsernames,
	options ...Option) (err error) {

	if err = ctx.Err(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	resolvedOptions := processOptions(options...)
	dir := resolvedOptions.FilesystemDirectory

	// Fail fast on a bad identifier rather than quoting it into the SQL. The Builder
	// already validates, but roleUsernames is a plain map a caller can build by hand
	// and so bypass it; this guarantees validation however the map was constructed.
	if err = roles.ValidatePlaceholderUsernames(roleUsernames); err != nil {
		return fmt.Errorf("invalid role usernames: %w", err)
	}

	tfs := templatingFS{
		inner:                  fsys,
		usernamesByPlaceholder: rolePlaceholderUsernames(roleUsernames),
	}

	var (
		migrationSource source.Driver
		driver          database.Driver
	)

	migrationSource, err = iofs.New(tfs, dir)
	if err != nil {
		return fmt.Errorf("open embedded migrations %q: %w", dir, err)
	}
	defer func() {
		_ = migrationSource.Close()
	}()

	// Wrap the caller's pgx pool as a database/sql handle for golang-migrate.
	// OpenDBFromPool does NOT take ownership: closing this sql.DB instance
	// leaves the pool open for the caller to manage.
	sqlDB := stdlib.OpenDBFromPool(pool)
	driver, err = migratepgx.WithInstance(sqlDB, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("init migration driver: %w", err)
	}

	const (
		sourceName         = "iofs"
		databaseDriverName = "pgx5"
	)
	migrator, errM := migrate.NewWithInstance(sourceName, migrationSource, databaseDriverName, driver)
	if errM != nil {
		_ = driver.Close() // not yet owned by a migrator (leaves pool open)
		return fmt.Errorf("init migrator: %w", errM)
	}
	defer func() {
		_, _ = migrator.Close()
	}()

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// golang-migrate marks a version dirty before
		// running it and clears it only on success
		if version, dirty, errV := migrator.Version(); errV == nil && dirty {
			return fmt.Errorf("apply migrations (dirty at version %d): %w", version, err)
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	// Up returned nil (changes applied) or ErrNoChange
	// (already current); both mean success.
	return nil
}
