// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/llingr/llingr-pgx/roles"
)

// Migrate validates the role/username map before it touches the pool, so a bad
// identifier supplied through a hand-built map (bypassing the Builder) is rejected
// up front rather than quoted into the SQL. Validation precedes any pool use, so a
// nil pool is never dereferenced on this path.
func TestMigrate_InvalidRoleUsernameIsRejected(t *testing.T) {
	bad := map[roles.Placeholder]roles.Username{
		roles.AppRole: "not a valid ident",
	}

	err := Migrate(context.Background(), nil, fstest.MapFS{}, bad)
	if err == nil {
		t.Fatal("expected Migrate to reject an invalid username")
	}
	if !strings.Contains(err.Error(), "invalid role usernames") {
		t.Fatalf("error should name the validation failure, got: %v", err)
	}
}

// A FilesystemDirectory that does not exist in the FS fails when the source driver
// is opened, which happens before any pool use, so a nil pool is never dereferenced.
func TestMigrate_BadFilesystemDirectoryIsError(t *testing.T) {
	fsys := fstest.MapFS{
		"001_x.up.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}

	err := Migrate(context.Background(), nil, fsys, map[roles.Placeholder]roles.Username{},
		WithFilesystemDirectory("does-not-exist"))
	if err == nil {
		t.Fatal("expected error for a missing migrations directory")
	}
	if !strings.Contains(err.Error(), "open embedded migrations") {
		t.Fatalf("error should name the failed source open, got: %v", err)
	}
}

// A cancelled context fails fast, before validation or any pool use. Cancellation
// is only honoured up front: golang-migrate cannot cancel a run in progress.
func TestMigrate_CancelledContextFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Migrate(ctx, nil, fstest.MapFS{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got: %v", err)
	}
}

// golang-migrate pings the database when it builds its driver, so a pool pointed
// at nothing fails there rather than at any later step. pgxpool builds a pool
// without dialling when MinConns is zero, which is what lets this run with no
// server: the pool exists, and its first use is the ping.
func TestMigrate_DriverInitFailureIsReported(t *testing.T) {
	config, err := pgxpool.ParseConfig(
		"postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	fsys := fstest.MapFS{
		"001_x.up.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}

	err = Migrate(context.Background(), pool, fsys, map[roles.Placeholder]roles.Username{})
	if err == nil {
		t.Fatal("expected Migrate to fail against an unreachable database")
	}
	if !strings.Contains(err.Error(), "init migration driver") {
		t.Fatalf("error should name the failed driver init, got: %v", err)
	}
}
