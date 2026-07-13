// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

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
