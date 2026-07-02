package connect

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This file holds the connection options beyond the core identity (host, port,
// credentials, database) and pool sizing in builder.go: observability, HA topology,
// TLS material, connection-pooler compatibility, and a programmatic escape hatch for
// bespoke setups (AWS RDS IAM auth, in-memory TLS, tracing, pgx type registration).
//
// Everything except WithConfigHook is sugar over WithParam: it renders into the
// connection string and is recovered by pgx's ParseConfig. WithConfigHook is the one
// imperative option; it mutates the parsed *pgxpool.Config at Connect time.

// WithApplicationName sets application_name, the label this connection reports to the
// server (visible in pg_stat_activity and server logs). Strongly recommended: it makes
// a connection traceable back to the service that opened it.
func (b *ConnectionBuilder) WithApplicationName(name string) *ConnectionBuilder {
	return b.WithParam(ParamApplicationName, name)
}

// WithConnectTimeout sets the libpq connect_timeout: the maximum time to wait while
// establishing a single connection. libpq's granularity is whole seconds, so the
// duration is rounded up (e.g. 1500ms becomes 2s); for sub-second control, bound the
// context passed to Connect instead. A zero or negative duration means no timeout.
func (b *ConnectionBuilder) WithConnectTimeout(timeout time.Duration) *ConnectionBuilder {
	seconds := int64(0)
	if timeout > 0 {
		seconds = int64((timeout + time.Second - 1) / time.Second) // ceil to whole seconds
	}
	return b.WithParam(ParamConnectTimeout, strconv.FormatInt(seconds, 10))
}

// WithTargetSessionAttrs picks which host to use from a multi-host connection by the
// session's role: "read-write", "read-only", "primary", "standby", "prefer-standby",
// or "any" (the default). The standard way to target the writer in a Postgres HA
// cluster (Patroni and similar). pgx tries the hosts in order and keeps the first that
// matches; an unknown value is rejected at Connect time.
func (b *ConnectionBuilder) WithTargetSessionAttrs(attrs string) *ConnectionBuilder {
	return b.WithParam(ParamTargetSessionAttrs, attrs)
}

// WithSSLRootCert sets sslrootcert, the path to the CA certificate(s) used to verify
// the server under sslmode verify-ca / verify-full. The special value "system" uses
// the host's system certificate pool. For a CA held in memory rather than on disk
// (e.g. pulled from a secret manager), use WithConfigHook to set ConnConfig.TLSConfig.
func (b *ConnectionBuilder) WithSSLRootCert(path string) *ConnectionBuilder {
	return b.WithParam(ParamSSLRootCert, path)
}

// WithSSLCert sets sslcert, the path to the client certificate for mutual TLS.
func (b *ConnectionBuilder) WithSSLCert(path string) *ConnectionBuilder {
	return b.WithParam(ParamSSLCert, path)
}

// WithSSLKey sets sslkey, the path to the client private key for mutual TLS. If the
// key is encrypted, supply its passphrase via WithParam("sslpassword", …).
func (b *ConnectionBuilder) WithSSLKey(path string) *ConnectionBuilder {
	return b.WithParam(ParamSSLKey, path)
}

// WithPgBouncerCompatibility configures pgx to run safely behind a transaction-pooling
// connection pooler that does NOT support prepared statements: stock PgBouncer in
// transaction/statement mode, and AWS RDS Proxy (where prepared statements force
// session pinning, defeating pooling).
//
// Why: by default pgx prepares and caches named server-side statements. Such a pooler
// hands each transaction whatever backend is free, so a statement prepared on one
// backend is missing on the next, producing intermittent "prepared statement \"…\"
// already exists" / "does not exist" errors under load. This sets
// default_query_exec_mode=exec and disables the statement/description caches, so pgx
// stops relying on persistent prepared statements. It is invisible in development (one
// backend) and only bites in production once the pooler multiplexes.
//
// You usually do NOT need this on serverless providers whose poolers support prepared
// statements: Neon's pooler (a PgBouncer fork with max_prepared_statements enabled)
// and Supabase's Supavisor both replay prepared statements per client, so pgx's
// defaults work over their pooled endpoints. Reach for this only for a pooler that
// lacks that support.
//
// This option is orthogonal to scale-to-zero. Whether a serverless compute is allowed
// to suspend is governed by the pool, not the query protocol: keep MinConns at 0 (the
// default) so the pool does not keep reopening connections and waking a suspended
// compute, and optionally set WithMaxConnIdleTime below the provider's suspend window
// so idle connections are released before the compute sleeps. A separate concern is
// migrations: golang-migrate takes a session-scoped advisory lock, which is unreliable
// through any transaction pooler, so run schema.Migrate against the direct (non-pooled)
// endpoint even when application traffic uses the pooled one.
//
// The trade-off is losing pgx's prepared-statement caching (a small latency cost on
// repeated queries). If your pooler runs in session pooling mode, you do not need this.
func (b *ConnectionBuilder) WithPgBouncerCompatibility() *ConnectionBuilder {
	return b.
		WithParam(ParamDefaultQueryExecMode, "exec").
		WithParam(ParamStatementCacheCapacity, "0").
		WithParam(ParamDescriptionCacheCapacity, "0")
}

// WithConfigHook registers a function that receives the fully parsed *pgxpool.Config
// just before the pool is opened, the escape hatch for everything that is not
// expressible as a connection-string parameter. The builder renders and parses the
// string first, then calls the hook, so the hook has the last word and can override
// anything. It runs once per Connect call and does not affect DSN()/PSQL() output. A
// non-nil error from the hook aborts the connection.
//
// Use it for the function-valued pgx settings the string form cannot reach:
//
//   - Dynamic credentials (AWS RDS / Aurora IAM, GCP Cloud SQL IAM, Azure AD): set
//     ConnConfig.BeforeConnect to mint a fresh auth token per connection.
//   - Per-connection setup and custom types (pgvector, PostGIS, enums): set
//     ConnConfig.AfterConnect to run SET statements or register types.
//   - In-memory TLS (a CA loaded from a secret manager, not a file): set
//     ConnConfig.TLSConfig.
//   - Observability: set ConnConfig.Tracer to a pgx QueryTracer (e.g. OpenTelemetry).
//
// Example, AWS RDS IAM authentication:
//
//	b.WithConfigHook(func(c *pgxpool.Config) error {
//	    c.BeforeConnect = func(ctx context.Context, cc *pgx.ConnConfig) error {
//	        token, err := auth.BuildAuthToken(ctx, endpoint, region, dbUser, creds)
//	        if err != nil {
//	            return err
//	        }
//	        cc.Password = token
//	        return nil
//	    }
//	    return nil
//	})
func (b *ConnectionBuilder) WithConfigHook(hook func(*pgxpool.Config) error) *ConnectionBuilder {
	b.configHook = hook
	return b
}
