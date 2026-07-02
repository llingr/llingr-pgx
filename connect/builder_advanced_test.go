package connect

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The string-sugar setters render their libpq keyword and parse back through pgx.
func TestAdvancedSetters_RenderAndParse(t *testing.T) {
	b := NewConnectionBuilder().
		WithHost("h").
		WithApplicationName("billing-svc").
		WithConnectTimeout(1500 * time.Millisecond). // ceils to 2s
		WithTargetSessionAttrs("read-write")

	config, err := pgxpool.ParseConfig(b.DSN())
	if err != nil {
		t.Fatalf("ParseConfig: %v\n  %s", err, b.DSN())
	}
	if got := config.ConnConfig.RuntimeParams["application_name"]; got != "billing-svc" {
		t.Errorf("application_name = %q", got)
	}
	if config.ConnConfig.ConnectTimeout != 2*time.Second {
		t.Errorf("ConnectTimeout = %s, want 2s", config.ConnConfig.ConnectTimeout)
	}

	// TLS file setters render their keywords (not parsed here: that would read the files).
	dsn := NewConnectionBuilder().
		WithSSLRootCert("/etc/ssl/ca.pem").WithSSLCert("/etc/ssl/c.pem").WithSSLKey("/etc/ssl/k.pem").
		DSN()
	for _, want := range []string{"sslrootcert=/etc/ssl/ca.pem", "sslcert=/etc/ssl/c.pem", "sslkey=/etc/ssl/k.pem"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN missing %q: %s", want, dsn)
		}
	}
}

// WithMinIdleConns / WithMaxConnLifetimeJitter render their pool_* keys and parse back.
func TestWithMinIdleConns(t *testing.T) {
	config, err := pgxpool.ParseConfig(NewConnectionBuilder().
		WithHost("h").WithMinIdleConns(3).WithMaxConnLifetimeJitter(90 * time.Second).DSN())
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if config.MinIdleConns != 3 {
		t.Errorf("MinIdleConns = %d, want 3", config.MinIdleConns)
	}
	if config.MaxConnLifetimeJitter != 90*time.Second {
		t.Errorf("MaxConnLifetimeJitter = %s, want 1m30s", config.MaxConnLifetimeJitter)
	}
}

// WithPgBouncerCompatibility selects exec mode and disables the statement caches.
func TestPgBouncerCompatibility(t *testing.T) {
	config, err := pgxpool.ParseConfig(NewConnectionBuilder().WithHost("h").WithPgBouncerCompatibility().DSN())
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if config.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Errorf("exec mode = %v, want QueryExecModeExec", config.ConnConfig.DefaultQueryExecMode)
	}
	if config.ConnConfig.StatementCacheCapacity != 0 || config.ConnConfig.DescriptionCacheCapacity != 0 {
		t.Errorf("caches not disabled: stmt=%d desc=%d",
			config.ConnConfig.StatementCacheCapacity, config.ConnConfig.DescriptionCacheCapacity)
	}
}

// The config hook runs after parsing and before opening; an error from it aborts.
func TestConfigHook(t *testing.T) {
	called := false
	_, err := NewConnectionBuilder().
		WithHost("127.0.0.1").WithPort(5432).WithDatabase("db").
		WithConfigHook(func(c *pgxpool.Config) error { called = true; return nil }).
		Connect(cancelledContext())
	if err == nil {
		t.Error("expected error opening on cancelled context")
	}
	if !called {
		t.Error("config hook should run before the pool is opened")
	}

	_, err = NewConnectionBuilder().WithHost("h").
		WithConfigHook(func(c *pgxpool.Config) error { return errors.New("boom") }).
		Connect(context.Background())
	if err == nil || !strings.Contains(err.Error(), "config hook") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want wrapped config-hook error, got %v", err)
	}
}

// A value pgx rejects at parse (here an unknown target_session_attrs) surfaces as a
// wrapped parse error from the builder's connect path, before any network use.
func TestConnect_ParseErrorFromBadParam(t *testing.T) {
	_, err := NewConnectionBuilder().WithHost("h").WithTargetSessionAttrs("bogus").Connect(context.Background())
	if err == nil || !strings.Contains(err.Error(), "parse connection") {
		t.Fatalf("want wrapped parse error, got %v", err)
	}
}

// Socket host renders as a host query parameter in the URL form and round-trips.
func TestPSQL_UnixSocketHost(t *testing.T) {
	uri := NewConnectionBuilder().
		WithHost("/var/run/postgresql").WithPort(5432).WithUser("app").WithDatabase("prod").
		PSQL()
	config, err := pgxpool.ParseConfig(uri)
	if err != nil {
		t.Fatalf("ParseConfig(%s): %v", uri, err)
	}
	if config.ConnConfig.Host != "/var/run/postgresql" {
		t.Errorf("socket host = %q, want /var/run/postgresql", config.ConnConfig.Host)
	}
}
