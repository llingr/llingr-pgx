// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/llingr/llingr-pgx/connect"
)

// the moment UK clocks went forward in 2026, and an hour after it
var (
	firstInstant  = time.Date(2026, time.March, 29, 1, 0, 0, 0, time.UTC)
	secondInstant = time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)
)

// timeTypeCases are timestamptz and every type pgx builds on it, each selected holding the
// first instant, or the first and the second, and the Go type each is scanned into
var timeTypeCases = []struct {
	name     string
	sql      string
	target   func() any
	instants []time.Time
}{
	{
		name:     "timestamptz into time.Time",
		sql:      "SELECT '2026-03-29 01:00:00+00'::timestamptz",
		target:   func() any { return new(time.Time) },
		instants: []time.Time{firstInstant},
	},
	{
		name:     "timestamptz into pgtype.Timestamptz",
		sql:      "SELECT '2026-03-29 01:00:00+00'::timestamptz",
		target:   func() any { return new(pgtype.Timestamptz) },
		instants: []time.Time{firstInstant},
	},
	{
		name: "timestamptz[]",
		sql: "SELECT ARRAY['2026-03-29 01:00:00+00'::timestamptz, " +
			"'2026-03-29 02:00:00+00'::timestamptz]",
		target:   func() any { return new([]time.Time) },
		instants: []time.Time{firstInstant, secondInstant},
	},
	{
		name:     "tstzrange",
		sql:      "SELECT tstzrange('2026-03-29 01:00:00+00', '2026-03-29 02:00:00+00')",
		target:   func() any { return new(pgtype.Range[pgtype.Timestamptz]) },
		instants: []time.Time{firstInstant, secondInstant},
	},
	{
		name:     "tstzrange[]",
		sql:      "SELECT ARRAY[tstzrange('2026-03-29 01:00:00+00', '2026-03-29 02:00:00+00')]",
		target:   func() any { return new([]pgtype.Range[pgtype.Timestamptz]) },
		instants: []time.Time{firstInstant, secondInstant},
	},
	{
		name: "tstzmultirange",
		sql: "SELECT tstzmultirange(tstzrange('2026-03-29 01:00:00+00', " +
			"'2026-03-29 02:00:00+00'))",
		target:   func() any { return new(pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]) },
		instants: []time.Time{firstInstant, secondInstant},
	},
	{
		name: "tstzmultirange[]",
		sql: "SELECT ARRAY[tstzmultirange(tstzrange('2026-03-29 01:00:00+00', " +
			"'2026-03-29 02:00:00+00'))]",
		target:   func() any { return new([]pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]) },
		instants: []time.Time{firstInstant, secondInstant},
	},
}

// TestEveryPoolReturnsTimesInUTC: a pool opened by any of connect's calls returns timestamptz,
// and every array, range and multirange of it, in UTC whatever the process's time zone, over
// the binary and the text protocol and scanned typed or as values, as the instants written;
// a caller's own AfterConnect runs after that, so it has the last word
func TestEveryPoolReturnsTimesInUTC(t *testing.T) {
	host, port := startProvisionedPostgres(t)
	url := fmt.Sprintf("postgres://postgres:postgres@%s:%s/%s?sslmode=disable", host, port,
		testDatabase)
	builder := func() *connect.ConnectionBuilder {
		portNumber, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			t.Fatalf("container port %q: %v", port, err)
		}
		return connect.NewConnectionBuilder().WithHost(host).WithPort(uint16(portNumber)).
			WithUser("postgres").WithPassword("postgres").WithDatabase(testDatabase).
			WithSSLMode("disable")
	}
	opens := map[string]func(ctx context.Context) (*pgxpool.Pool, error){
		"Connect": func(ctx context.Context) (*pgxpool.Pool, error) {
			return connect.Connect(ctx, url)
		},
		"ConnectEnv": func(ctx context.Context) (*pgxpool.Pool, error) {
			t.Setenv("PGHOST", host)
			t.Setenv("PGPORT", port)
			t.Setenv("PGUSER", "postgres")
			t.Setenv("PGPASSWORD", "postgres")
			t.Setenv("PGDATABASE", testDatabase)
			t.Setenv("PGSSLMODE", "disable")
			return connect.ConnectEnv(ctx)
		},
		"ConnectConfig": func(ctx context.Context) (*pgxpool.Pool, error) {
			config, err := pgxpool.ParseConfig(url)
			if err != nil {
				return nil, err
			}
			return connect.ConnectConfig(ctx, config)
		},
		"ConnectionBuilder.ConnectDSN": func(ctx context.Context) (*pgxpool.Pool, error) {
			return builder().ConnectDSN(ctx)
		},
		"ConnectionBuilder.ConnectPSQL": func(ctx context.Context) (*pgxpool.Pool, error) {
			return builder().ConnectPSQL(ctx)
		},
	}
	protocols := map[string][]any{
		"binary results":                {},
		"text results, simple protocol": {pgx.QueryExecModeSimpleProtocol},
	}

	for openName, open := range opens {
		t.Run(openName, func(t *testing.T) {
			ctx := context.Background()
			pool, err := open(ctx)
			if err != nil {
				t.Fatalf("open the pool: %v", err)
			}
			t.Cleanup(pool.Close)
			for _, testCase := range timeTypeCases {
				for protocolName, mode := range protocols {
					t.Run(testCase.name+"/"+protocolName, func(t *testing.T) {
						target := testCase.target()
						err = pool.QueryRow(ctx, testCase.sql, mode...).Scan(target)
						if err != nil {
							t.Fatalf("scan: %v", err)
						}
						requireUTCInstants(t, "scanned", target, testCase.instants)

						rows, queryErr := pool.Query(ctx, testCase.sql, mode...)
						if queryErr != nil {
							t.Fatalf("query: %v", queryErr)
						}
						defer rows.Close()
						if !rows.Next() {
							t.Fatalf("no row: %v", rows.Err())
						}
						values, valuesErr := rows.Values()
						if valuesErr != nil {
							t.Fatalf("values: %v", valuesErr)
						}
						requireUTCInstants(t, "as values", values, testCase.instants)
					})
				}
			}
		})
	}

	t.Run("a caller's AfterConnect has the last word", func(t *testing.T) {
		ctx := context.Background()
		newYork := time.FixedZone("New York", -5*60*60)
		config, err := pgxpool.ParseConfig(url)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		config.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
			conn.TypeMap().RegisterType(&pgtype.Type{
				Name:  "timestamptz",
				OID:   pgtype.TimestamptzOID,
				Codec: &pgtype.TimestamptzCodec{ScanLocation: newYork},
			})
			return nil
		}
		pool, err := connect.ConnectConfig(ctx, config)
		if err != nil {
			t.Fatalf("open the pool: %v", err)
		}
		defer pool.Close()

		var scanned time.Time
		err = pool.QueryRow(ctx, "SELECT '2026-03-29 01:00:00+00'::timestamptz").Scan(&scanned)
		if err != nil || scanned.Location() != newYork || !scanned.Equal(firstInstant) {
			t.Fatalf("scanned %s in %s, %v; want %s in the caller's New York", scanned,
				scanned.Location(), err, firstInstant)
		}
	})

	t.Run("a caller's AfterConnect error fails the connect", func(t *testing.T) {
		config, err := pgxpool.ParseConfig(url)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		errRefused := errors.New("refused by the caller")
		config.AfterConnect = func(context.Context, *pgx.Conn) error {
			return errRefused
		}
		pool, err := connect.ConnectConfig(context.Background(), config)
		if err == nil {
			pool.Close()
			t.Fatal("the connect succeeded, want the caller's AfterConnect error")
		}
		if !errors.Is(err, errRefused) {
			t.Fatalf("connect failed with %v, want the caller's AfterConnect error", err)
		}
	})
}

// requireUTCInstants fails unless the times within value are the instants wanted, in order,
// each in time.UTC itself rather than a zone at the same offset
func requireUTCInstants(t *testing.T, how string, value any, instants []time.Time) {
	t.Helper()
	found := timesWithin(reflect.ValueOf(value))
	if len(found) != len(instants) {
		t.Fatalf("%s: found %d times in %#v, want %d", how, len(found), value, len(instants))
	}
	for index, scanned := range found {
		if scanned.Location() != time.UTC || !scanned.Equal(instants[index]) {
			t.Errorf("%s: time %d is %s in %q, want %s in UTC", how, index, scanned,
				scanned.Location(), instants[index])
		}
	}
}

// timesWithin is every time.Time inside value, in order, through pointers, interfaces,
// slices and exported struct fields, so a range gives its lower bound then its upper
func timesWithin(value reflect.Value) []time.Time {
	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			return nil
		}
		return timesWithin(value.Elem())
	case reflect.Slice, reflect.Array:
		var times []time.Time
		for index := range value.Len() {
			times = append(times, timesWithin(value.Index(index))...)
		}
		return times
	case reflect.Struct:
		if found, isTime := value.Interface().(time.Time); isTime {
			return []time.Time{found}
		}
		var times []time.Time
		for index := range value.NumField() {
			if value.Type().Field(index).IsExported() {
				times = append(times, timesWithin(value.Field(index))...)
			}
		}
		return times
	}
	return nil
}
