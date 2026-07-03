package connect

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionBuilder collects connection fields and pool tuning, then renders
// this as either style connection string (PSQL, DSN).
//
//	b := connect.NewConnectionBuilder().
//	    WithHost("db.example.com").
//	    WithDatabase("app_db").
//	    WithUser("app").
//	    WithPassword(secret).
//	    WithSSLMode("require").
//	    WithMaxConns(10)
//
//	dsn  := b.DSN()                  // host=db.example.com port=5432 user=app password='...' ...
//	uri  := b.PSQL()                 // postgres://app:%E2%80%A6@db.example.com:5432/appdb?...
//	pool, err := b.ConnectDSN(ctx)   // or ConnectPSQL(ctx), or Connect(ctx)
//
// Both renderings carry the pool tuning as pgx's pool_* parameters, so the same
// config produces an equivalent pool whichever style you connect with. Prefer DSN
// (the default for Connect) when the password holds URL-unsafe characters, as
// AWS RDS passwords routinely do.
type ConnectionBuilder struct {
	host     string
	port     uint16
	user     string
	password string
	database string
	sslmode  string

	channelBinding string

	paramKeys []string // for keeping key order
	params    map[string]string

	maxConns              int32
	minConns              int32
	minIdleConns          int32
	maxConnLifetime       time.Duration
	maxConnLifetimeJitter time.Duration
	maxConnIdleTime       time.Duration
	healthCheckPeriod     time.Duration

	// configHook is the programmatic escape hatch applied to the parsed config at
	// Connect time; see WithConfigHook in builder_advanced.go.
	configHook func(*pgxpool.Config) error
}

// NewConnectionBuilder returns an empty builder.
func NewConnectionBuilder() *ConnectionBuilder {
	return &ConnectionBuilder{params: map[string]string{}}
}

// WithHost sets the host (a hostname, IP, or absolute path to a unix socket dir).
func (b *ConnectionBuilder) WithHost(host string) *ConnectionBuilder {
	b.host = host
	return b
}

// WithPort sets the TCP port.
func (b *ConnectionBuilder) WithPort(port uint16) *ConnectionBuilder {
	b.port = port
	return b
}

// WithUser sets the role to authenticate as.
func (b *ConnectionBuilder) WithUser(user string) *ConnectionBuilder {
	b.user = user
	return b
}

// WithPassword sets the password. URL-unsafe characters are fine:
// DSN quotes and PSQL percent-encodes these.
func (b *ConnectionBuilder) WithPassword(password string) *ConnectionBuilder {
	b.password = password
	return b
}

// WithDatabase sets the database name (libpq "dbname").
func (b *ConnectionBuilder) WithDatabase(database string) *ConnectionBuilder {
	b.database = database
	return b
}

// WithSSLMode sets the libpq sslmode (disable, require, verify-ca, verify-full, …).
// See ValidSSLModes for the full set.
func (b *ConnectionBuilder) WithSSLMode(mode string) *ConnectionBuilder {
	b.sslmode = mode
	return b
}

// WithParam sets any other libpq keyword (e.g. "connect_timeout", "application_name",
// "sslrootcert"), for parameters without a dedicated setter. Using it to duplicate a
// dedicated setter that is also in use fails validation at Connect time, since the
// DSN and URL parsers resolve duplicate keywords in opposite orders.
func (b *ConnectionBuilder) WithParam(key, value string) *ConnectionBuilder {
	if _, exists := b.params[key]; !exists {
		b.paramKeys = append(b.paramKeys, key)
	}
	b.params[key] = value
	return b
}

// WithMaxConns caps the pool size (pgx pool_max_conns).
func (b *ConnectionBuilder) WithMaxConns(maxConns int32) *ConnectionBuilder {
	b.maxConns = maxConns
	return b
}

// WithMinConns sets the minimum total pool size (pgx pool_min_conns).
func (b *ConnectionBuilder) WithMinConns(minConns int32) *ConnectionBuilder {
	b.minConns = minConns
	return b
}

// WithMinIdleConns sets the minimum number of idle connections the pool keeps ready
// (pgx pool_min_idle_conns), so bursts find warm connections without paying connect
// latency. Distinct from WithMinConns, which is a floor on total connections; the pool
// maintains max(minConns, minIdleConns) idle resources. Leave it at 0 for scale-to-zero
// deployments, since a non-zero value keeps reopening connections and waking a
// suspended compute, exactly like a non-zero MinConns.
func (b *ConnectionBuilder) WithMinIdleConns(minIdleConns int32) *ConnectionBuilder {
	b.minIdleConns = minIdleConns
	return b
}

// WithMaxConnLifetime sets how long a connection may live before recycling
// (pgx pool_max_conn_lifetime).
func (b *ConnectionBuilder) WithMaxConnLifetime(lifetime time.Duration) *ConnectionBuilder {
	b.maxConnLifetime = lifetime
	return b
}

// WithMaxConnLifetimeJitter spreads connection recycling over a random window
// (pgx pool_max_conn_lifetime_jitter): each connection's effective lifetime becomes
// MaxConnLifetime plus a random duration in [0, jitter). Without it, connections opened
// together expire together, producing a periodic thundering herd of reconnects; a jitter
// of roughly 10-20% of the lifetime staggers them. Only meaningful alongside
// WithMaxConnLifetime and a pool of more than a few connections.
func (b *ConnectionBuilder) WithMaxConnLifetimeJitter(jitter time.Duration) *ConnectionBuilder {
	b.maxConnLifetimeJitter = jitter
	return b
}

// WithMaxConnIdleTime sets how long an idle connection may sit before being closed
// (pgx pool_max_conn_idle_time).
func (b *ConnectionBuilder) WithMaxConnIdleTime(idle time.Duration) *ConnectionBuilder {
	b.maxConnIdleTime = idle
	return b
}

// WithHealthCheckPeriod sets the interval between idle-connection health checks
// (pgx pool_health_check_period).
func (b *ConnectionBuilder) WithHealthCheckPeriod(period time.Duration) *ConnectionBuilder {
	b.healthCheckPeriod = period
	return b
}
