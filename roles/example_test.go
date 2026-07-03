package roles_test

import (
	"fmt"

	"github.com/llingr/llingr-pgx/roles"
)

// Map each role placeholder to the username provisioned for the target
// environment. Migrations GRANT to :"app" / :"readonly" and the real usernames
// are substituted at run time; the built map also carries the usernames for
// wiring connection pools.
func ExampleNewPlaceholderBuilder() {
	usernames := roles.NewPlaceholderBuilder().
		WithOwner("ecommerce_schema_owner").               // -> :"owner"    built-in
		WithApp("ecommerce_app_user").                     // -> :"app"      built-in
		WithCustom("readonly", "ecommerce_readonly_user"). // -> :"readonly" custom
		MustBuild()

	fmt.Println(usernames.OwnerUsername())
	fmt.Println(usernames.AppUsername())
	fmt.Println(usernames.UsernameFor("readonly"))
	// Output:
	// ecommerce_schema_owner
	// ecommerce_app_user
	// ecommerce_readonly_user
}
