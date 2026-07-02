package connect_test

import (
	"fmt"

	"github.com/llingr/llingr-pgx/connect"
)

// One builder renders the same configuration as either connection-string style.
// Note the RDS-style password: single-quoted in the DSN, percent-encoded in the
// URL.
//
// Both rendered strings embed the password. They are credentials: never log,
// trace, or print them (this example prints only to demonstrate the format,
// with a throwaway value).
func ExampleConnectionBuilder() {
	b := connect.NewConnectionBuilder().
		WithHost("db.example.com").
		WithPort(5432).
		WithUser("app_readwrite").
		WithPassword("pa$s/w rd"). // connection strings contain this so must not be logged
		WithDatabase("appdb").
		WithSSLMode("require").
		WithMaxConns(10)

	fmt.Println(b.DSN())
	fmt.Println(b.PSQL())
	// Output:
	// host=db.example.com port=5432 user=app_readwrite password='pa$s/w rd' dbname=appdb sslmode=require pool_max_conns=10
	// postgres://app_readwrite:pa$s%2Fw%20rd@db.example.com:5432/appdb?sslmode=require&pool_max_conns=10
}
