// Package migrations embeds the SQL migration files so the directory is a
// self-contained unit: the embed.FS lives alongside the *.sql it carries, rooted
// at this directory. Run them with schema.Migrate(ctx, pool, migrations.FS, roles)
// (FilesystemDirectory defaults to ".").
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
