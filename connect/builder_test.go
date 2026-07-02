package connect

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A password with the special characters AWS RDS produces must survive both
// renderings: quoted in the DSN, percent-encoded in the URL. Parsing each string
// back must recover the exact fields and the pool tuning.
func TestBuilder_BothStylesRoundTripFromOneConfig(t *testing.T) {
	const rdsPassword = `a/b+c=d e'f`

	builder := NewConnectionBuilder().
		WithHost("db.example.com").
		WithPort(5432).
		WithUser("app").
		WithPassword(rdsPassword).
		WithDatabase("appdb").
		WithSSLMode("require").
		WithMaxConns(10).
		WithMaxConnIdleTime(30 * time.Minute)

	for _, style := range []struct {
		name       string
		connString string
	}{
		{"DSN", builder.DSN()},
		{"PSQL", builder.PSQL()},
	} {
		t.Run(style.name, func(t *testing.T) {
			config, err := pgxpool.ParseConfig(style.connString)
			if err != nil {
				t.Fatalf("ParseConfig(%s): %v\n  %s", style.name, err, style.connString)
			}
			if config.ConnConfig.Host != "db.example.com" || config.ConnConfig.Port != 5432 {
				t.Errorf("host/port = %s:%d", config.ConnConfig.Host, config.ConnConfig.Port)
			}
			if config.ConnConfig.User != "app" || config.ConnConfig.Database != "appdb" {
				t.Errorf("user/db = %s/%s", config.ConnConfig.User, config.ConnConfig.Database)
			}
			if config.ConnConfig.Password != rdsPassword {
				t.Errorf("password round-trip failed: %q", config.ConnConfig.Password)
			}
			if config.MaxConns != 10 {
				t.Errorf("MaxConns = %d, want 10", config.MaxConns)
			}
			if config.MaxConnIdleTime != 30*time.Minute {
				t.Errorf("MaxConnIdleTime = %s, want 30m", config.MaxConnIdleTime)
			}
		})
	}
}

// The DSN spells fields in libpq keyword form; the URL spells them as a scheme,
// userinfo, host, path and query.
func TestBuilder_RenderingShapes(t *testing.T) {
	builder := NewConnectionBuilder().
		WithHost("h").WithPort(5432).WithUser("u").WithDatabase("d").WithSSLMode("require")

	dsn := builder.DSN()
	for _, want := range []string{"host=h", "port=5432", "user=u", "dbname=d", "sslmode=require"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN missing %q: %s", want, dsn)
		}
	}

	uri := builder.PSQL()
	if !strings.HasPrefix(uri, "postgres://u@h:5432/d") {
		t.Errorf("unexpected URL: %s", uri)
	}
	if !strings.Contains(uri, "sslmode=require") {
		t.Errorf("URL missing sslmode: %s", uri)
	}
}

// An empty builder renders an empty string, which pgx reads as defaults/env.
func TestBuilder_EmptyRendersEmpty(t *testing.T) {
	if dsn := NewConnectionBuilder().DSN(); dsn != "" {
		t.Errorf("empty DSN = %q, want \"\"", dsn)
	}
}

// The remaining pool tunables and an extra param all render and parse back.
func TestBuilder_AllPoolTunablesAndParam(t *testing.T) {
	builder := NewConnectionBuilder().
		WithHost("h").
		WithParam("application_name", "migrator").
		WithMinConns(2).
		WithMaxConnLifetime(time.Hour).
		WithHealthCheckPeriod(time.Minute)

	config, err := pgxpool.ParseConfig(builder.DSN())
	if err != nil {
		t.Fatalf("ParseConfig: %v\n  %s", err, builder.DSN())
	}
	if config.MinConns != 2 {
		t.Errorf("MinConns = %d, want 2", config.MinConns)
	}
	if config.MaxConnLifetime != time.Hour {
		t.Errorf("MaxConnLifetime = %s, want 1h", config.MaxConnLifetime)
	}
	if config.HealthCheckPeriod != time.Minute {
		t.Errorf("HealthCheckPeriod = %s, want 1m", config.HealthCheckPeriod)
	}
	if config.ConnConfig.RuntimeParams["application_name"] != "migrator" {
		t.Errorf("application_name = %q", config.ConnConfig.RuntimeParams["application_name"])
	}
}

// A Neon-style config (sslmode + channel_binding) renders and parses in both styles.
func TestBuilder_ChannelBinding(t *testing.T) {
	builder := NewConnectionBuilder().
		WithHost("ep-pooler.eu-central-1.aws.neon.tech").
		WithUser("app").WithPassword("pw").WithDatabase("prod").
		WithSSLMode("require").WithChannelBinding("require")

	if err := builder.validate(); err != nil {
		t.Fatalf("Neon-style config (sslmode=require, channel_binding=require) should validate: %v", err)
	}
	for _, s := range []struct{ name, str string }{{"DSN", builder.DSN()}, {"PSQL", builder.PSQL()}} {
		config, err := pgxpool.ParseConfig(s.str)
		if err != nil {
			t.Fatalf("ParseConfig(%s): %v\n  %s", s.name, err, s.str)
		}
		if config.ConnConfig.ChannelBinding != "require" {
			t.Errorf("%s ChannelBinding = %q, want require", s.name, config.ConnConfig.ChannelBinding)
		}
	}
}

// Host without a port renders cleanly in both styles (the URL omits the colon).
func TestBuilder_HostWithoutPort(t *testing.T) {
	uri := NewConnectionBuilder().WithHost("only-host").WithUser("u").PSQL()
	if !strings.HasPrefix(uri, "postgres://u@only-host") || strings.Contains(uri, "only-host:") {
		t.Errorf("unexpected URL for host-without-port: %s", uri)
	}
}
