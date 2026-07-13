// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

// Package migrations embeds the SQL migration files from a Go file that sits
// next to the files themselves, so the embed.FS is rooted at this directory
// and the migrations appear at its root: schema.Migrate needs no directory
// shift. This is the recommended layout for a migrations directory.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
