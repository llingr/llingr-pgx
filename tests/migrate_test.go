// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/llingr/llingr-pgx/roles"

	"github.com/llingr/llingr-pgx/connect"
	"github.com/llingr/llingr-pgx/schema"
	"github.com/llingr/llingr-pgx/tests/testdata/migrations"
)

const (
	testDatabase = "appdb"
	testPassword = "tester" // matches docker/init_db.sql
)

// testUsername maps each library role to the user the dev Dockerfile pre-creates,
// proving the provisioned users line up with the code's Placeholder values.
// readOnlyRole is an application-defined custom role (the library ships only
// OwnerRole and AppRole as built-ins); its placeholder matches the
// migrations' :"readonly".
const readOnlyRole roles.Placeholder = "readonly"

var testUsername = map[roles.Placeholder]roles.Username{
	roles.OwnerRole: "ecommerce_schema_owner",
	roles.AppRole:   "ecommerce_app_user",
	readOnlyRole:    "ecommerce_readonly_user",
}

type customer struct {
	CustomerID uuid.UUID `db:"customer_id"`
	Email      string    `db:"email"`
	FullName   string    `db:"full_name"`
	CreatedTs  time.Time `db:"created_ts"`
}

// TestMigrateAgainstProvisionedPostgres builds the dev Postgres image (which
// pre-creates the role-aligned users), then proves: migrations apply as the owner
// role from the embedded FS, the read-write user can write and be read back via
// scany, and per-table grants hold (app may UPDATE orders but not DELETE them;
// read-only may read but not write). Queries use pgx named parameters in snake_case
// mirroring the columns, and primary-key UUIDs are minted in Go (google/uuid) and
// bound as parameters rather than defaulted by the database.
func TestMigrateAgainstProvisionedPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()
	host, port := startProvisionedPostgres(t)
	urlForRole := func(role roles.Placeholder) string {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			testUsername[role], testPassword, host, port, testDatabase)
	}

	// The placeholder->username map carries the usernames substituted into the
	// migrations' :"name" placeholders. They must match the users the dev image provisions.
	roleUsernames := roles.NewPlaceholderBuilder().
		WithOwner(testUsername[roles.OwnerRole]).
		WithApp(testUsername[roles.AppRole]).
		WithCustom(readOnlyRole, testUsername[readOnlyRole]).
		MustBuild()

	// Migrate as the owner role (the only one allowed to run DDL): inject a pool
	// authenticated as Owner, closed at test end so only the lesser app /
	// read-only roles remain for the assertions below. Three migrations: customers,
	// currency, customer_orders.
	ownerPool, err := connect.Connect(ctx, urlForRole(roles.OwnerRole))
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	defer ownerPool.Close()

	if err := schema.Migrate(ctx, ownerPool, migrations.FS, roleUsernames); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Re-running is a no-op (idempotent).
	if err := schema.Migrate(ctx, ownerPool, migrations.FS, roleUsernames); err != nil {
		t.Fatalf("re-migrate: %v", err)
	}

	// The read-write user can write; read it back with scany. Pools come from the
	// library's connect.Connect helper.
	readWrite, err := connect.Connect(ctx, urlForRole(roles.AppRole))
	if err != nil {
		t.Fatalf("read-write pool: %v", err)
	}
	defer readWrite.Close()

	customerID := uuid.New()
	if _, err := readWrite.Exec(ctx,
		`INSERT INTO customers (customer_id, email, full_name)
		 VALUES (@customer_id, @email, @full_name)`,
		pgx.NamedArgs{
			"customer_id": customerID,
			"email":       "ada@example.com",
			"full_name":   "Ada Lovelace",
		}); err != nil {
		t.Fatalf("read-write insert customer: %v", err)
	}

	var loaded customer
	if err := pgxscan.Get(ctx, readWrite, &loaded,
		`SELECT customer_id, email, full_name, created_ts
		   FROM customers WHERE email = @email`,
		pgx.NamedArgs{"email": "ada@example.com"}); err != nil {
		t.Fatalf("scany get customer: %v", err)
	}
	if loaded.FullName != "Ada Lovelace" || loaded.CustomerID != customerID {
		t.Fatalf("unexpected customer row: %+v", loaded)
	}

	// FK into the orders table proves migration order resolved the reference.
	orderID := uuid.New()
	if _, err := readWrite.Exec(ctx,
		`INSERT INTO customer_orders
		   (customer_order_id, customer_id, currency, amount, amount_minor_units, order_status)
		 VALUES (@customer_order_id, @customer_id, @currency, @amount, @amount_minor_units, @order_status)`,
		pgx.NamedArgs{
			"customer_order_id":  orderID,
			"customer_id":        customerID,
			"currency":           "USD",
			"amount":             int64(1399),
			"amount_minor_units": 2,
			"order_status":       "pending",
		}); err != nil {
		t.Fatalf("read-write insert order: %v", err)
	}

	// Per-table privileges: the app user may UPDATE orders...
	if _, err := readWrite.Exec(ctx,
		`UPDATE customer_orders SET order_status = 'paid' WHERE customer_id = @customer_id`,
		pgx.NamedArgs{"customer_id": customerID}); err != nil {
		t.Fatalf("read-write UPDATE on customer_orders should be allowed: %v", err)
	}
	// ...but must NOT delete them (no DELETE granted in the migration).
	if _, err := readWrite.Exec(ctx,
		`DELETE FROM customer_orders WHERE customer_id = @customer_id`,
		pgx.NamedArgs{"customer_id": customerID}); err == nil {
		t.Fatal("read-write DELETE on customer_orders should be denied (no DELETE granted)")
	}

	// The read-only user can read but not write.
	readOnly, err := connect.Connect(ctx, urlForRole(readOnlyRole))
	if err != nil {
		t.Fatalf("read-only pool: %v", err)
	}
	defer readOnly.Close()

	var count int
	if err := readOnly.QueryRow(ctx, `SELECT count(*) FROM customers`).Scan(&count); err != nil {
		t.Fatalf("read-only select: %v", err)
	}
	if count != 1 {
		t.Fatalf("read-only customer count = %d, want 1", count)
	}
	if _, err := readOnly.Exec(ctx,
		`INSERT INTO customers (customer_id, email, full_name)
		 VALUES (@customer_id, @email, @full_name)`,
		pgx.NamedArgs{
			"customer_id": uuid.New(),
			"email":       "mallory@example.com",
			"full_name":   "Mallory",
		}); err == nil {
		t.Fatal("read-only INSERT should have been denied, but succeeded")
	}
}

// TestMigrate_FailingMigrationReportsDirtyVersion drives the dirty-version branch of
// schema.Migrate: a migration that fails mid-run leaves golang-migrate's version
// marked dirty, and Migrate should surface that, naming the version it stopped at.
// The broken migration is a single in-memory file with invalid SQL (no placeholders),
// so version 1 is attempted, fails, and is left dirty.
func TestMigrate_FailingMigrationReportsDirtyVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()
	host, port := startProvisionedPostgres(t)

	// Owner is the only role allowed to run DDL; the broken migration has no
	// :"name" placeholders, so an owner-only map suffices.
	roleUsernames := roles.NewPlaceholderBuilder().
		WithOwner(testUsername[roles.OwnerRole]).
		MustBuild()

	ownerURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		testUsername[roles.OwnerRole], testPassword, host, port, testDatabase)
	ownerPool, err := connect.Connect(ctx, ownerURL)
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	defer ownerPool.Close()

	brokenMigrations := fstest.MapFS{
		"001_broken.up.sql": &fstest.MapFile{
			Data: []byte("BEGIN TRANSACTION;\nTHIS IS NOT VALID SQL;\nCOMMIT TRANSACTION;\n"),
		},
	}

	err = schema.Migrate(ctx, ownerPool, brokenMigrations, roleUsernames)
	if err == nil {
		t.Fatal("expected migrate to fail on the broken migration")
	}
	const want = "apply migrations (dirty at version 1)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error should report the dirty version.\n  want substring: %q\n  got: %v", want, err)
	}
}
