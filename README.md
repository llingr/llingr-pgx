# llingr-pgx

[![Go Reference](https://pkg.go.dev/badge/github.com/llingr/llingr-pgx.svg)](https://pkg.go.dev/github.com/llingr/llingr-pgx)
[![CI](https://github.com/llingr/llingr-pgx/actions/workflows/ci.yml/badge.svg)](https://github.com/llingr/llingr-pgx/actions/workflows/ci.yml)
[![Lint](https://github.com/llingr/llingr-pgx/actions/workflows/lint.yml/badge.svg)](https://github.com/llingr/llingr-pgx/actions/workflows/lint.yml)
[![Integration](https://github.com/llingr/llingr-pgx/actions/workflows/integration.yml/badge.svg)](https://github.com/llingr/llingr-pgx/actions/workflows/integration.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/llingr/llingr-pgx)](go.mod)
[![Tag](https://img.shields.io/github/v/tag/llingr/llingr-pgx)](https://github.com/llingr/llingr-pgx/tags)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Postgres migrations with injection-safe role templating, a
connection-string builder, and ORM-free SQL helpers:

 * Connection string builder for both key/value DSN and `postgres://` URL formats, reducing
   boilerplate around many of the advanced capabilities provided by the excellent
   [pgx](https://github.com/jackc/pgx) client. Mildly opinionated while leaving access
   for bespoke setups.
 * Embedded schema migrations using [golang-migrate](https://github.com/golang-migrate/migrate)
   which allows full control of transaction boundaries - easily `CREATE INDEX CONCURRENTLY`.
 * Migration file templating using the quoted-identifier form of psql variable interpolation:
   injection-safe runtime (per-environment) `GRANT SELECT ON orders TO :"readonly";`
 * Roles builder to map the above interpolation placeholders to real usernames.
 * SQL fragments package for ORM-free SQL statements; [scany](https://github.com/georgysavva/scany)
   is recommended for binding structs (not bundled with this library).  

```sh
go get github.com/llingr/llingr-pgx
```

For working examples see the `Example*` functions on
[pkg.go.dev](https://pkg.go.dev/github.com/llingr/llingr-pgx) (one per package, in the
`example_test.go` files) and the end-to-end integration tests in `tests/`, which run the
full migrate-grant-query cycle against a real Postgres.

## Quick Start

Embed the migration SQL files, map each role placeholder to a username, open a pool as
the owner role, and migrate:

```go
//go:embed *.sql
var migrationsFS embed.FS

ctx := context.Background()

roleUsernames := roles.NewPlaceholderBuilder().
    WithOwner("example_schema_owner"). // see 'Roles and Placeholders' below
    WithApp("example_app_readwrite").
    MustBuild()

ownerPool, err := connect.NewConnectionBuilder().
    WithHost("db.example.com").WithPort(5432).
    WithDatabase("appdb").WithSSLMode("require").
    WithUser(roleUsernames.OwnerUsername()).
    WithPassword(ownerPassword).
    Connect(ctx)
if err != nil {
    log.Fatal(err)
}
defer ownerPool.Close()

if err := schema.Migrate(ctx, ownerPool, migrationsFS, roleUsernames); err != nil {
    log.Fatal(err)
}
```

## Roles and Placeholders

Migrations may refer to roles by placeholder because the same logical role can have a different
username in each environment. Placeholders are substituted during deployments.

Two built-in placeholders are provided, and custom placeholders can be added:

 * `OwnerRole` (`owner`), owns the schema and runs migrations, but is not meant
   for runtime queries
 * `AppRole` (`app`), the read/write application user. Should not have schema-altering rights
   so that an injection bug cannot drop a table.
 * `WithCustom` supports further roles as needed.

```go
const ReadOnlyRole roles.Placeholder = "readonly"

roleUsernames := roles.NewPlaceholderBuilder().
    WithOwner("example_schema_owner").            // -> :"owner"     built-in
    WithApp("example_app_readwrite").             // -> :"app"       built-in
    WithCustom(ReadOnlyRole, "example_readonly"). // -> :"readonly"  custom
    MustBuild()
```

`Build()` returns a `(roles.PlaceholderUsernames, error)` pair, while `MustBuild()` panics
on error. The returned collection is provided to `schema.Migrate`, and carries accessors such
as `OwnerUsername()` for wiring connection pools (see the Quick Start example).

Placeholder injection is prevented by requiring plain SQL identifiers: `^[A-Za-z_][A-Za-z0-9_]*$`.


## Migrations

The database schema is described using SQL files. These are read from an embedded `fs.FS`. Database
connectivity is via [pgx](https://github.com/jackc/pgx) connection capability.

Migrations typically include statements such as `GRANT SELECT, INSERT, UPDATE ON orders TO :"app"`
with role placeholders substituted at runtime (hard-coded usernames are not recommended). This
gives developers full control over all aspects of a database schema and (through the files themselves)
complete visibility of roles vs access levels.

Complete transaction control is left to the developer, allowing for `CREATE INDEX CONCURRENTLY`
and other non-transactional features. For regular DDL changes use `BEGIN TRANSACTION;` and
`COMMIT TRANSACTION;` to make changes atomic.

Migrations follow [golang-migrate](https://github.com/golang-migrate/migrate)'s `NNN_name.up.sql`
convention and are forward-only: to mitigate risks, `.down.sql` rollbacks are not directly supported.

```go
//go:embed *.sql
var migrationsFS embed.FS

// ...
ctx := context.Background()

usernames := roles.NewPlaceholderBuilder().
    WithOwner("prod_eu_ecomm_owner"). // -> :"owner" for running DDL, rarely used in SQL templates
    WithApp("prod_eu_ecomm_app").     // -> :"app" in the migration files
    MustBuild()

// Use schema owner DDL role for schema migrations. EXAMPLE ONLY: do *NOT* embed secrets in code!
ownerPool, err := connect.Connect(ctx, "postgres://prod_eu_ecomm_owner:secret@prod-eu.example:5432/ecomm")
if err != nil {
    log.Fatal(err)
}
defer ownerPool.Close()

err = schema.Migrate(ctx, ownerPool, migrationsFS, usernames)
if err != nil {
    log.Fatal(err)
}
```


## SQL Fragments

The `queries` package provides a store for explicitly crafted, 'named' SQL statements,
which provide explicit control and clarity over the SQL sent to a database, and avoids
the complexity associated with ORM tools and query builders. The `-- name:` marker is
the yesql convention, as used by [dotsql](https://github.com/qustavo/dotsql) and similar.

The [pgx](https://github.com/jackc/pgx) client provides a really useful `NamedArgs` capability
which makes writing and debugging queries significantly easier - the example below shows this:

```sql
-- name: insert-customer
INSERT INTO customers (customer_id, email, full_name)
VALUES (@customer_id, @email, @full_name);

-- name: order-with-customer
SELECT o.customer_order_id AS order_id,
       c.full_name         AS customer_name,
       o.amount            AS amount
FROM   customer_orders o
JOIN   customers c ON o.customer_id = c.customer_id
WHERE  o.customer_order_id = @customer_order_id;
```

### SQL Fragments using [pgx](https://github.com/jackc/pgx) Named Arguments

```go
//go:embed queries/*.sql
var queryFS embed.FS

q, err := queries.Load(queryFS) // load and validate once at startup
if err != nil {
    log.Fatal(err)
}

_, err = pool.Exec(ctx, q.SQL("insert-customer"), pgx.NamedArgs{
    "customer_id": customerId,
    "email":       "grace@example.com",
    "full_name":   "Grace Hopper",
})
```

For more complete examples (insert, join, struct binding) see:

- `tests/queries/orders.sql`
- `tests/queries_integration_test.go`


## Connection Strings

The `connect` package opens a `*pgxpool.Pool` and tests the connection with a ping.
This needs a connection string which can be supplied in a number of ways:

- `Connect(ctx, connString)` accepts either a `postgres://` URL or a libpq
  keyword/value string
- `ConnectEnv(ctx)` reads from environment variables
  (`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`, `PGSERVICE`,
  and the rest that `psql` honours)
- `ConnectConfig(ctx, config)` uses `*pgxpool.Config`


The `ConnectionBuilder` provides a fluent API to create connection strings in code,
and provides a good selection of discoverable settings. More advanced settings are
exposed through helpers (e.g. PgBouncer, see below), and direct key/value bindings.

```go
b := connect.NewConnectionBuilder().
    WithHost("db.example.com").
    WithPort(5432).
    WithUser("app_readwrite").
    WithPassword(rdsPassword). // special characters are supported
    WithDatabase("appdb").
    WithSSLMode("require").
    WithMaxConns(10).
    WithMaxConnIdleTime(30 * time.Minute)

dsn := b.DSN()    // host=db.example.com port=5432 user=app_readwrite password='...' pool_max_conns=10 ...
uri := b.PSQL()   // postgres://app_readwrite:...@db.example.com:5432/appdb?sslmode=require&pool_max_conns=10&...

pool, err := b.ConnectDSN(ctx)   // or ConnectPSQL(ctx), or Connect(ctx)
```

The builder provides an equivalent connection pool whichever style is used to connect. 
`DSN()` and `PSQL()` return the string without opening anything, while `ConnectDSN`
and `ConnectPSQL` generate a connection string and open the connection; plain `Connect`
is an alias for the DSN form.

The DSN single-quotes values whereas the PSQL URL must percent-encode them, and while
both are handled correctly, the DSN format is generally more intuitive to work with.

Further settings in `builder_advanced.go`:

- `WithApplicationName(name)` labels the connection in `pg_stat_activity` and the server
  logs, making it traceable to the service that opened it.
- `WithConnectTimeout(d)` sets a per-connection connect deadline; libpq's granularity is
  whole seconds and the value is rounded up, so use the context passed to `Connect` for
  sub-second control.
- `WithTargetSessionAttrs(attrs)` chooses the writer or a reader from a multi-host
  connection (`read-write`, `read-only`, `primary`, `standby`, `prefer-standby`, or
  `any`), the usual way to target the primary in an HA cluster.
- `WithSSLRootCert`, `WithSSLCert`, and `WithSSLKey` set the CA and client-certificate
  paths for `verify-ca` / `verify-full` and mutual TLS, and `sslrootcert=system` uses
  the host's certificate pool.
- `WithChannelBinding(mode)` sets `channel_binding` (`prefer`, `disable`, or `require`),
  tying authentication to the TLS channel so credentials cannot be replayed.
- `WithPgBouncerCompatibility()` makes pgx safe behind a transaction pooler that lacks
  prepared-statement support, covered under
  [Serverless and scale-to-zero](#serverless-and-scale-to-zero).


`WithConfigHook` exposes `*pgxpool.Config` which allows for custom behaviour;
the hook runs at connect time and does not affect `DSN()` or `PSQL()`.

```go
// example AWS RDS IAM auth: create a fresh token per connection
b.WithConfigHook(func(c *pgxpool.Config) error {
    c.BeforeConnect = func(ctx context.Context, cc *pgx.ConnConfig) error {
        token, err := auth.BuildAuthToken(ctx, endpoint, region, dbUser, creds)
        if err != nil {
            return err
        }
        cc.Password = token
        return nil
    }
    return nil
})
```

The `ConnConfig.AfterConnect` callback is useful for registering `pgvector` or PostGIS
types, and to run `SET` statements, `ConnConfig.TLSConfig` to supply a CA held in memory rather
than on disk, or `ConnConfig.Tracer` for OpenTelemetry. Unix-socket hosts, given as an absolute path,
work in both approaches and are mapped to `host=` query parameters in the URL form.


## Serverless and Scale-to-Zero

Serverless Postgres such as Neon, Supabase, or Aurora Serverless v2 has two further
wrinkles: compute that suspends when idle, and client traffic arriving through a
transaction pooler. Advice for handling this is set out below.

Keep the pool from holding connections open indefinitely: leave `MinConns` and `MinIdleConns`
at their `0` default so the pool never reopens a connection just to maintain a floor, and
set `WithMaxConnIdleTime` below the provider's suspend window so idle connections are
released before the compute sleeps; Neon's default suspend is five minutes, which makes
three to four minutes a safe idle time.

Run migrations against the direct endpoint even when the application uses the pooled
one. `schema.Migrate` takes a session-scoped advisory lock, golang-migrate's guard
against concurrent migrators, and that lock is unreliable through a transaction pooler,
which may route successive statements to different backends. With Neon this is simply
the difference between the plain host and the `-pooler` host.

Match the query protocol to the pooler; pgx prepares and caches named server-side
statements by default, and a pooler without prepared-statement support (stock PgBouncer
in transaction mode, or AWS RDS Proxy) will then throw intermittent
`prepared statement "..." already exists` errors under load.
`WithPgBouncerCompatibility()` switches pgx to exec mode and disables the caches for
exactly this situation. You do not need it on Neon or Supabase, whose poolers replay
prepared statements per client.

A representative Neon setup migrates on the direct endpoint and serves on the pooled one:

```go
// Migrations: direct endpoint (no -pooler), as the owner role.
ownerPool, err := connect.NewConnectionBuilder().
    WithHost("ep-cool-darkness-123456.us-east-2.aws.neon.tech").
    WithUser("app_owner").WithPassword(neonOwnerPassword).WithDatabase("appdb").
    WithSSLMode("require").WithChannelBinding("require").
    Connect(ctx)
if err != nil {
    log.Fatal(err)
}

// Application traffic: pooled endpoint, tuned to let the compute suspend.
appPool, err := connect.NewConnectionBuilder().
    WithHost("ep-cool-darkness-123456-pooler.us-east-2.aws.neon.tech").
    WithUser("app_readwrite").WithPassword(neonAppPassword).WithDatabase("appdb").
    WithSSLMode("require").WithChannelBinding("require").
    WithMaxConns(10).WithMaxConnIdleTime(3 * time.Minute). // MinConns stays 0
    Connect(ctx)
if err != nil {
    log.Fatal(err)
}
```

Supabase uses the same idea with different endpoints: its Supavisor pooler listens
on port 6543 in transaction mode for application traffic and on 5432 in session mode,
which is where migrations should run so the advisory lock holds. AWS RDS Proxy pins a
session as soon as a prepared statement is used, which defeats pooling, so pair it with
`WithPgBouncerCompatibility()` and run migrations against the cluster writer endpoint
rather than the proxy; for Aurora or RDS IAM authentication, create tokens - per
connection - using `WithConfigHook` as shown above.

---
Development companion for [llingr-demux](https://github.com/llingr/llingr-demux), the
formally verified, high-throughput event-streaming engine. See [llingr.io](https://llingr.io).
