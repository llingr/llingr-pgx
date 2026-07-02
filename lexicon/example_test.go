package lexicon_test

import (
	"embed"
	"fmt"
	"log"

	"github.com/llingr/llingr-pgx/lexicon"
)

//go:embed testdata/queries
var queryFS embed.FS

// Load parses every embedded .sql file into named fragments, validating names
// and bodies once at startup. Execution stays with your driver: pass the
// returned SQL to pgx, together with pgx.NamedArgs for the @named parameters.
func ExampleLoad() {
	queries, err := lexicon.Load(queryFS)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(queries.Names())
	fmt.Println(queries.SQL("user-by-email"))
	// Output:
	// [all-users user-by-email]
	// SELECT user_id, full_name
	// FROM   users
	// WHERE  email = @email;
}
