// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"embed"
	"fmt"
	"testing"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/llingr/llingr-pgx/connect"
	"github.com/llingr/llingr-pgx/queries"
	"github.com/llingr/llingr-pgx/roles"
	"github.com/llingr/llingr-pgx/schema"
	"github.com/llingr/llingr-pgx/tests/testdata/migrations"
)

//go:embed queries/*.sql
var queryFS embed.FS

// orderJoin is the projection of the customers x customer_orders x currency_codes
// join, bound by scany via the `db` tags. The ids stay as google uuid.UUID end to
// end: bound as parameters going in, scanned back out.
type orderJoin struct {
	OrderID       uuid.UUID `db:"order_id"`
	CustomerID    uuid.UUID `db:"customer_id"`
	CustomerName  string    `db:"customer_name"`
	CustomerEmail string    `db:"customer_email"`
	Currency      string    `db:"currency"`
	CurrencyName  string    `db:"currency_name"`
	Amount        int64     `db:"amount"`
	OrderStatus   string    `db:"order_status"`
}

// TestQueries_InsertAndJoinWithNamedParams drives the queries package end to end
// against real Postgres. It loads named SQL fragments from embedded .sql files,
// inserts a customer and an order, then reads them back through an explicit
// JOIN ... ON. Primary-key UUIDs are generated in Go (google/uuid) and bound as
// named parameters, so the inserts need no gen_random_uuid() or RETURNING. Every
// statement uses pgx named parameters (@name), and the joined row is scanned
// straight into a struct by scany.
func TestQueries_InsertAndJoinWithNamedParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testcontainers integration test in -short mode")
	}
	ctx := context.Background()

	// Load the named SQL fragments once: this is the whole job of the package.
	q, err := queries.Load(queryFS)
	if err != nil {
		t.Fatalf("q.Load: %v", err)
	}

	host, port := startProvisionedPostgres(t)
	urlForRole := func(role roles.Placeholder) string {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			testUsername[role], testPassword, host, port, testDatabase)
	}
	roleUsernames := roles.NewPlaceholderBuilder().
		WithOwner(testUsername[roles.OwnerRole]).
		WithApp(testUsername[roles.AppRole]).
		WithCustom(readOnlyRole, testUsername[readOnlyRole]).
		MustBuild()

	// Migrate as the owner role (the only one allowed DDL), then drop that pool so
	// the rest of the test runs as the lesser app role.
	ownerPool, err := connect.Connect(ctx, urlForRole(roles.OwnerRole))
	if err != nil {
		t.Fatalf("owner pool: %v", err)
	}
	if err := schema.Migrate(ctx, ownerPool, migrations.FS, roleUsernames); err != nil {
		ownerPool.Close()
		t.Fatalf("migrate: %v", err)
	}
	ownerPool.Close()

	app, err := connect.Connect(ctx, urlForRole(roles.AppRole))
	if err != nil {
		t.Fatalf("app pool: %v", err)
	}
	defer app.Close()

	// Keys are minted in Go and bound as parameters. No DB-side generator, and no
	// RETURNING round-trip since we already hold the ids.
	customerID := uuid.New()
	if _, err := app.Exec(ctx, q.SQL("insert-customer"),
		pgx.NamedArgs{
			"customer_id": customerID,
			"email":       "grace@example.com",
			"full_name":   "Grace Hopper",
		}); err != nil {
		t.Fatalf("insert-customer: %v", err)
	}

	orderID := uuid.New()
	if _, err := app.Exec(ctx, q.SQL("insert-order"),
		pgx.NamedArgs{
			"customer_order_id":  orderID,
			"customer_id":        customerID,
			"currency":           "USD",
			"amount":             int64(1399),
			"amount_minor_units": 2,
			"order_status":       "paid",
		}); err != nil {
		t.Fatalf("insert-order: %v", err)
	}

	// Read it back through the explicit join, scanned into a struct by scany.
	var got orderJoin
	if err := pgxscan.Get(ctx, app, &got, q.SQL("order-with-customer"),
		pgx.NamedArgs{"customer_order_id": orderID}); err != nil {
		t.Fatalf("order-with-customer join: %v", err)
	}

	if got.OrderID != orderID || got.CustomerID != customerID ||
		got.CustomerName != "Grace Hopper" || got.CustomerEmail != "grace@example.com" ||
		got.Currency != "USD" || got.CurrencyName != "United States dollar" ||
		got.Amount != 1399 || got.OrderStatus != "paid" {
		t.Fatalf("unexpected joined row: %+v\n  want order %s / customer %s", got, orderID, customerID)
	}
}
