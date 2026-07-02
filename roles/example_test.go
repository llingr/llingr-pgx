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
		WithAdminOwner("ecommerce_admin_user").            // -> :"admin_owner"
		WithApp("ecommerce_app_user").                     // -> :"app"
		WithCustom("readonly", "ecommerce_readonly_user"). // -> :"readonly"
		MustBuild()

	fmt.Println(usernames.AdminOwnerUsername())
	fmt.Println(usernames.AppUsername())
	fmt.Println(usernames.UsernameFor("readonly"))
	// Output:
	// ecommerce_admin_user
	// ecommerce_app_user
	// ecommerce_readonly_user
}
