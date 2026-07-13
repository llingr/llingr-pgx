// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

// Package connect opens pgx connection pools to Postgres, each verified with a ping
// so a bad address or unreachable server fails at connect time rather than on first
// query.
//
// Two layers:
//
//   - Raw inputs you already hold: Connect (a connection string pgx auto-detects as
//     a "postgres://" URL or a libpq keyword/value DSN), ConnectEnv (the libpq
//     environment variables), and ConnectConfig (a pre-built *pgxpool.Config).
//   - Built from fields: ConnectionBuilder collects host, port, credentials, and pool
//     tuning once, then renders that one config as either style of connection string
//     (DSN or PSQL) and connects with ConnectDSN / ConnectPSQL / Connect.
package connect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pool from a connection string, letting pgx auto-detect whether it
// is a "postgres://" URL or a libpq keyword/value DSN. Use this with an existing
// connection string (use ConnectionBuilder to construct one from individual fields).
func Connect(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	return connectString(ctx, connString)
}

// ConnectEnv opens a pool configured entirely from the libpq environment variables
// (PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE, PGSSLMODE, PGSERVICE,
// PGPASSFILE, …), the same set psql honours. It is the empty-connection-string case.
func ConnectEnv(ctx context.Context) (*pgxpool.Pool, error) {
	return connectString(ctx, "")
}

// ConnectConfig opens a pool from pre-built *pgxpool.Config
// This is the single create-and-ping path every connector uses.
func ConnectConfig(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
	if config == nil {
		return nil, fmt.Errorf("nil pool config")
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// connectString parses any connection string pgx understands
func connectString(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse connection: %w", err)
	}
	return ConnectConfig(ctx, config)
}
