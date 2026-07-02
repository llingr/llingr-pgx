package connect

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect is an alias for ConnectDSN
func (b *ConnectionBuilder) Connect(ctx context.Context) (*pgxpool.Pool, error) {
	return b.ConnectDSN(ctx)
}

// ConnectDSN renders DSN string and opens connection;
// this is the RDS-safe default
func (b *ConnectionBuilder) ConnectDSN(ctx context.Context) (*pgxpool.Pool, error) {
	return b.connect(ctx, b.DSN())
}

// ConnectPSQL renders "postgres://" URL and opens connection
func (b *ConnectionBuilder) ConnectPSQL(ctx context.Context) (*pgxpool.Pool, error) {
	return b.connect(ctx, b.PSQL())
}

// connect validates the config, parses the rendered connection string, applies the
// WithConfigHook (if any), and opens the pool. It is the single path ConnectDSN and
// ConnectPSQL share, so validation and the hook run identically whichever style opens.
func (b *ConnectionBuilder) connect(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse connection: %w", err)
	}
	if b.configHook != nil {
		if err := b.configHook(config); err != nil {
			return nil, fmt.Errorf("config hook: %w", err)
		}
	}
	return ConnectConfig(ctx, config)
}

// validate SSL mode, channel binding, and WithParam/dedicated-setter collisions
func (b *ConnectionBuilder) validate() error {
	if b.sslmode != "" && !slices.Contains(ValidSSLModes, b.sslmode) {
		return fmt.Errorf("invalid sslMode: %s", b.sslmode)
	}
	if b.channelBinding != "" && !slices.Contains(ValidChannelBindings, b.channelBinding) {
		return fmt.Errorf("invalid channelBinding: %s", b.channelBinding)
	}
	return b.validateNoParamCollisions()
}

// validateNoParamCollisions rejects a WithParam key that duplicates a dedicated
// setter already in use. Rendering the same keyword twice is never safe: the DSN
// and URL parsers resolve duplicates in opposite orders (last wins vs first wins),
// so the two renderings of one builder could produce different pools. WithParam is
// the escape hatch for keywords without a dedicated setter; this enforces that.
func (b *ConnectionBuilder) validateNoParamCollisions() error {
	dedicatedKeyInUse := map[string]bool{
		ParamHost:                      b.host != "",
		ParamPort:                      b.port != 0,
		ParamUser:                      b.user != "",
		ParamPassword:                  b.password != "",
		ParamDatabase:                  b.database != "",
		ParamSSLMode:                   b.sslmode != "",
		ParamChannelBinding:            b.channelBinding != "",
		ParamPoolMaxConns:              b.maxConns != 0,
		ParamPoolMinConns:              b.minConns != 0,
		ParamPoolMinIdleConns:          b.minIdleConns != 0,
		ParamPoolMaxConnLifetime:       b.maxConnLifetime != 0,
		ParamPoolMaxConnLifetimeJitter: b.maxConnLifetimeJitter != 0,
		ParamPoolMaxConnIdleTime:       b.maxConnIdleTime != 0,
		ParamPoolHealthCheckPeriod:     b.healthCheckPeriod != 0,
	}
	for _, key := range b.paramKeys {
		if dedicatedKeyInUse[key] {
			return fmt.Errorf("parameter %q collides with its dedicated setter: set it one way, not both", key)
		}
	}
	return nil
}
