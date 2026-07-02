package schema_test

import (
	"context"
	"log"

	"github.com/llingr/llingr-pgx/connect"
	"github.com/llingr/llingr-pgx/roles"
	"github.com/llingr/llingr-pgx/schema"
	"github.com/llingr/llingr-pgx/schema/testdata/migrations"
)

// Migrate applies the embedded, forward-only migrations as the owner role,
// substituting the mapped usernames into the files' :"name" placeholders.
//
// The migrations package embeds its *.sql from a Go file in the same
// directory (see testdata/migrations/fs_migration_files.go), so the fs.FS is
// rooted where the files are and no directory shift is needed. This example
// compiles but is not executed (no Output): it needs a reachable Postgres.
func ExampleMigrate() {
	ctx := context.Background()

	usernames := roles.NewPlaceholderBuilder().
		WithAdminOwner("app_owner"). // -> :"admin_owner" owns the schema and runs DDL
		WithApp("app_readwrite").    // -> :"app" in the migration files
		MustBuild()

	// The owner-role connection string ops handed you. This includes the
	// password, so it is a credential: never log it. Migrate borrows from the
	// pool but never closes it.
	adminPool, err := connect.Connect(ctx, "postgres://app_owner:secret@db.example.com:5432/appdb")
	if err != nil {
		log.Fatal(err)
	}
	defer adminPool.Close()

	err = schema.Migrate(ctx, adminPool, migrations.FS, usernames)
	if err != nil {
		log.Fatal(err)
	}
}
