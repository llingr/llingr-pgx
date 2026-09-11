// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

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

// A zoned IPv6 host (fe80::1%eth0) with no port is rendered as a host query
// parameter (like a socket path), because the bracketed form does not parse
// without a port. With a port it stays in the authority. Both parse back to
// the zoned address.
func TestPSQL_ZonedIPv6Host(t *testing.T) {
	for _, tc := range []struct {
		name string
		b    *ConnectionBuilder
		want string
	}{
		{"without port", NewConnectionBuilder().WithHost("fe80::1%eth0").WithUser("u"), "?host="},
		{"with port", NewConnectionBuilder().WithHost("fe80::1%eth0").WithPort(5432).WithUser("u"), "[fe80::1%25eth0]:5432"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uri := tc.b.PSQL()
			if !strings.Contains(uri, tc.want) {
				t.Fatalf("expected %q in %s", tc.want, uri)
			}
			config, err := pgxpool.ParseConfig(uri)
			if err != nil {
				t.Fatalf("ParseConfig(%s): %v", uri, err)
			}
			if config.ConnConfig.Host != "fe80::1%eth0" {
				t.Errorf("host = %q, want fe80::1%%eth0", config.ConnConfig.Host)
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

// A WithParam key that is not a plain lowercase libpq keyword fails validation:
// whitespace or '=' in a key renders differently in the DSN and URL forms.
func TestValidate_ParamKey(t *testing.T) {
	for _, key := range []string{"", "my key", "a=b", "Upper", "tab\tkey"} {
		b := NewConnectionBuilder().WithParam(key, "v")
		if err := b.validate(); err == nil || !strings.Contains(err.Error(), "invalid parameter name") {
			t.Errorf("key %q should fail validation, got %v", key, err)
		}
	}
	if err := NewConnectionBuilder().WithParam("connect_timeout", "5").validate(); err != nil {
		t.Errorf("valid key should pass: %v", err)
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

	// No sslmode set: validate() passes, so these reach the render-and-open path,
	// where the default ping meets the cancelled context.
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

// WithoutPing decides whether opening reaches the server at all. The default pings,
// so a cancelled context fails at connect time. WithoutPing leaves the pool lazy: at
// minConns 0 pgx opens no connection until the first acquire, so the same cancelled
// context yields a usable pool.
func TestBuilderWithoutPingIsLazy(t *testing.T) {
	reachable := func() *ConnectionBuilder {
		return NewConnectionBuilder().
			WithHost("127.0.0.1").WithPort(5432).WithUser("u").WithDatabase("db")
	}

	if _, err := reachable().Connect(cancelledContext()); err == nil {
		t.Error("the default ping should surface the connect-time failure")
	}

	pool, err := reachable().WithoutPing().Connect(cancelledContext())
	if err != nil {
		t.Fatalf("WithoutPing should not contact the server: %v", err)
	}
	pool.Close()
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

// openPool reports a pool-construction failure distinctly from a connection
// failure. pgxpool rejects a MaxConns below 1 before it dials, so zeroing it on a
// config that ParseConfig produced drives the construction error without needing a
// server. This is the only way into that branch: every other field ParseConfig sets
// is already valid by construction.
func TestOpenPool_RejectsUnusableMaxConns(t *testing.T) {
	config, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:5432/db?sslmode=disable")
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	config.MaxConns = 0

	pool, err := openPool(context.Background(), config)
	if err == nil {
		pool.Close()
		t.Fatal("expected openPool to reject a zero MaxConns")
	}
	if !strings.Contains(err.Error(), "open pool") {
		t.Fatalf("error should name the failed pool open, got: %v", err)
	}
}

// The builder returns openPool's error unwrapped, so the caller sees the pool
// failure rather than a second layer of context. The hook runs after ParseConfig
// and before the pool is built, which is what makes it usable to force this.
func TestBuilderConnect_ReturnsOpenPoolError(t *testing.T) {
	pool, err := NewConnectionBuilder().
		WithHost("127.0.0.1").WithPort(5432).WithUser("u").WithDatabase("db").
		WithSSLMode(SSLModeDisable).
		WithConfigHook(func(config *pgxpool.Config) error {
			config.MaxConns = 0
			return nil
		}).
		Connect(context.Background())
	if err == nil {
		pool.Close()
		t.Fatal("expected the builder to fail when the hook leaves MaxConns unusable")
	}
	if !strings.Contains(err.Error(), "open pool") {
		t.Fatalf("error should name the failed pool open, got: %v", err)
	}
}
