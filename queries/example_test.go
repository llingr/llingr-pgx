// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package queries_test

import (
	"embed"
	"fmt"
	"log"

	"github.com/llingr/llingr-pgx/queries"
)

//go:embed testdata/queries
var queryFS embed.FS

// Load parses every embedded .sql file into named statements, validating names
// and bodies once at startup. Execution stays with your driver: pass the
// returned SQL to pgx, together with pgx.NamedArgs for the @named parameters.
func ExampleLoad() {
	q, err := queries.Load(queryFS)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(q.Names())
	fmt.Println(q.SQL("user-by-email"))
	// Output:
	// [all-users user-by-email]
	// SELECT user_id, full_name
	// FROM   users
	// WHERE  email = @email;
}
