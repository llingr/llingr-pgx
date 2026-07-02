package connect

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// cancelledContext returns a context that is already cancelled, so a connection
// attempt fails immediately at the network step (no real server, no hang) while
// still exercising the function body up to that point.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// validate: valid (returns nil), invalid sslmode, and invalid channel binding.
func TestValidateConfig_AllBranches(t *testing.T) {
	if err := NewConnectionBuilder().
		WithSSLMode(SSLModeRequire).WithChannelBinding(ChannelBindingRequire).
		validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
	if err := NewConnectionBuilder().WithSSLMode("bogus").validate(); err == nil {
		t.Error("invalid sslmode should error")
	}
	if err := NewConnectionBuilder().
		WithSSLMode(SSLModeRequire).WithChannelBinding("bogus").
		validate(); err == nil {
		t.Error("invalid channel binding should error")
	}
}

// PSQL: the port-set-but-host-empty branch of the host switch.
func TestPSQL_PortWithoutHost(t *testing.T) {
	uri := NewConnectionBuilder().WithPort(5432).WithUser("u").PSQL()
	if !strings.Contains(uri, ":5432") {
		t.Errorf("expected \":5432\" in %s", uri)
	}
}

// An IPv6 literal host is bracketed in the URL authority whether or not a port is
// set, and parses back to the bare address either way.
func TestPSQL_IPv6HostIsBracketed(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *ConnectionBuilder
	}{
		{"without port", NewConnectionBuilder().WithHost("::1").WithUser("u")},
		{"with port", NewConnectionBuilder().WithHost("::1").WithPort(5432).WithUser("u")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uri := tc.b.PSQL()
			if !strings.Contains(uri, "[::1]") {
				t.Fatalf("IPv6 host not bracketed: %s", uri)
			}
			config, err := pgxpool.ParseConfig(uri)
			if err != nil {
				t.Fatalf("ParseConfig(%s): %v", uri, err)
			}
			if config.ConnConfig.Host != "::1" {
				t.Errorf("host = %q, want ::1", config.ConnConfig.Host)
			}
		})
	}
}

// A WithParam key that duplicates a dedicated setter in use fails validation:
// duplicate keywords are resolved last-wins by the DSN parser but first-wins by
// the URL parser, so the two renderings of one builder would disagree.
func TestValidate_ParamCollisionWithDedicatedSetter(t *testing.T) {
	collision := NewConnectionBuilder().WithSSLMode(SSLModeRequire).WithParam(ParamSSLMode, "disable")
	if err := collision.validate(); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Errorf("sslmode collision should fail validation, got %v", err)
	}
	if _, err := collision.ConnectDSN(context.Background()); err == nil {
		t.Error("ConnectDSN should surface the collision error")
	}

	pool := NewConnectionBuilder().WithMaxConns(10).WithParam(ParamPoolMaxConns, "5")
	if err := pool.validate(); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Errorf("pool_max_conns collision should fail validation, got %v", err)
	}

	// The escape hatch stays open: the same key via WithParam alone (no dedicated
	// setter in use) is fine, as is a param with no dedicated counterpart.
	if err := NewConnectionBuilder().WithParam(ParamSSLMode, "disable").validate(); err != nil {
		t.Errorf("WithParam without the dedicated setter should validate: %v", err)
	}
	if err := NewConnectionBuilder().WithHost("h").WithParam(ParamApplicationName, "svc").validate(); err != nil {
		t.Errorf("non-colliding param should validate: %v", err)
	}
}

// The builder Connect* methods: validate error path, plus the render-and-open
// path driven to its error return by a cancelled context.
func TestBuilderConnect_ErrorPaths(t *testing.T) {
	if _, err := NewConnectionBuilder().WithSSLMode("bogus").ConnectDSN(context.Background()); err == nil {
		t.Error("ConnectDSN should surface validate error")
	}
	if _, err := NewConnectionBuilder().WithSSLMode("bogus").ConnectPSQL(context.Background()); err == nil {
		t.Error("ConnectPSQL should surface validate error")
	}

	// No sslmode set: validate() passes, so these reach the render-and-open path.
	valid := NewConnectionBuilder().
		WithHost("127.0.0.1").WithPort(5432).WithUser("u").WithDatabase("db")
	if _, err := valid.Connect(cancelledContext()); err == nil {
		t.Error("Connect should error on cancelled context")
	}
	if _, err := valid.ConnectDSN(cancelledContext()); err == nil {
		t.Error("ConnectDSN should error on cancelled context")
	}
	if _, err := valid.ConnectPSQL(cancelledContext()); err == nil {
		t.Error("ConnectPSQL should error on cancelled context")
	}
}

// A nil pool config is rejected before any connection is attempted.
func TestConnectConfig_NilIsError(t *testing.T) {
	if _, err := ConnectConfig(context.Background(), nil); err == nil {
		t.Fatal("nil config should error")
	}
}

// Package-level connectors: ParseConfig error path, and the open path to its error
// return via a cancelled context.
func TestPackageConnect_ErrorPaths(t *testing.T) {
	if _, err := Connect(context.Background(), "postgres://%zz"); err == nil {
		t.Error("malformed connection string should fail at parse")
	}
	if _, err := Connect(cancelledContext(), "postgres://u:p@127.0.0.1:5432/db?sslmode=disable"); err == nil {
		t.Error("Connect should error on cancelled context")
	}
	if _, err := ConnectEnv(cancelledContext()); err == nil {
		t.Error("ConnectEnv should error on cancelled context")
	}

	config, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:5432/db?sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if _, err := ConnectConfig(cancelledContext(), config); err == nil {
		t.Error("ConnectConfig should error on cancelled context")
	}
}
