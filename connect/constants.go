// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package connect

// SchemePostgres is the URL scheme for a libpq connection URI ("postgres://…").
const SchemePostgres = "postgres"

// Postgres libpq connection-string keyword names. These are the exact
// spellings pgx parses (e.g. "dbname", not "database"), so they serve
// as the keys when rendering a DSN or the URL query.
const (
	ParamHost           = "host"
	ParamPort           = "port"
	ParamUser           = "user"
	ParamPassword       = "password"
	ParamDatabase       = "dbname"
	ParamSSLMode        = "sslmode"
	ParamChannelBinding = "channel_binding"
)

// pgx pool tuning parameter names.
// These are pgxpool additions to standard libpq keywords
const (
	ParamPoolMaxConns              = "pool_max_conns"
	ParamPoolMinConns              = "pool_min_conns"
	ParamPoolMinIdleConns          = "pool_min_idle_conns"
	ParamPoolMaxConnLifetime       = "pool_max_conn_lifetime"
	ParamPoolMaxConnLifetimeJitter = "pool_max_conn_lifetime_jitter"
	ParamPoolMaxConnIdleTime       = "pool_max_conn_idle_time"
	ParamPoolHealthCheckPeriod     = "pool_health_check_period"
)

// Additional libpq keyword names that have dedicated builder setters (see
// builder_advanced.go).
const (
	ParamApplicationName    = "application_name"
	ParamConnectTimeout     = "connect_timeout"
	ParamTargetSessionAttrs = "target_session_attrs"
	ParamSSLRootCert        = "sslrootcert"
	ParamSSLCert            = "sslcert"
	ParamSSLKey             = "sslkey"
)

// pgx query-execution parameter names, used by WithPgBouncerCompatibility to make
// pgx safe behind a transaction-pooling connection pooler (PgBouncer, RDS Proxy,
// the Neon/Supabase poolers).
const (
	ParamDefaultQueryExecMode     = "default_query_exec_mode"
	ParamStatementCacheCapacity   = "statement_cache_capacity"
	ParamDescriptionCacheCapacity = "description_cache_capacity"
)

// libpq ssl modes in increasing order of strictness
const (
	SSLModeDisable    = "disable"     // no SSL - connection is plaintext
	SSLModeAllow      = "allow"       // try plaintext first, fall back to SSL only if the server requires it
	SSLModePrefer     = "prefer"      // try SSL first, fall back to plaintext; no cert check (libpq default)
	SSLModeRequire    = "require"     // SSL required, but the server certificate is not verified
	SSLModeVerifyCA   = "verify-ca"   // SSL required and the certificate must be signed by a trusted CA
	SSLModeVerifyFull = "verify-full" // verify-ca plus the certificate hostname must match the server
)

// ValidSSLModes is the full set of libpq sslmode values WithSSLMode accepts.
var ValidSSLModes = []string{
	SSLModeDisable,
	SSLModeAllow,
	SSLModePrefer,
	SSLModeRequire,
	SSLModeVerifyCA,
	SSLModeVerifyFull,
}

// Channel binding cryptographically ties authentication
// exchange to the specific TLS connection; credentials
// proven on one channel can't be replayed onto another.
const (
	ChannelBindingPrefer  = "prefer"  // use channel binding if the server supports it, or fall back (libpq default)
	ChannelBindingDisable = "disable" // never use channel binding
	ChannelBindingRequire = "require" // demand channel binding; fail the connection if the server cannot do it
)

// ValidChannelBindings is the full set of libpq channel_binding values
// WithChannelBinding accepts.
var ValidChannelBindings = []string{
	ChannelBindingPrefer,
	ChannelBindingDisable,
	ChannelBindingRequire,
}
