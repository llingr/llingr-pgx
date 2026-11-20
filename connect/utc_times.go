// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package connect

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// scanTimesInUTC makes every connection of the pool return timestamptz values in UTC rather
// than the process's local time zone, then runs the config's own AfterConnect, which so keeps
// the last word
func scanTimesInUTC(config *pgxpool.Config) {
	callersAfterConnect := config.AfterConnect
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		registerUTCTimes(conn.TypeMap())
		if callersAfterConnect == nil {
			return nil
		}
		return callersAfterConnect(ctx, conn)
	}
}

// registerUTCTimes registers timestamptz scanning in UTC, and again every array, range and
// multirange of it: pgx builds each on the timestamptz type it registered first, so each would
// otherwise keep returning local time
func registerUTCTimes(typeMap *pgtype.Map) {
	timestamptz := &pgtype.Type{
		Name:  "timestamptz",
		OID:   pgtype.TimestamptzOID,
		Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
	}
	tstzrange := &pgtype.Type{
		Name:  "tstzrange",
		OID:   pgtype.TstzrangeOID,
		Codec: &pgtype.RangeCodec{ElementType: timestamptz},
	}
	tstzmultirange := &pgtype.Type{
		Name:  "tstzmultirange",
		OID:   pgtype.TstzmultirangeOID,
		Codec: &pgtype.MultirangeCodec{ElementType: tstzrange},
	}
	typeMap.RegisterType(timestamptz)
	typeMap.RegisterType(&pgtype.Type{
		Name:  "_timestamptz",
		OID:   pgtype.TimestamptzArrayOID,
		Codec: &pgtype.ArrayCodec{ElementType: timestamptz},
	})
	typeMap.RegisterType(tstzrange)
	typeMap.RegisterType(&pgtype.Type{
		Name:  "_tstzrange",
		OID:   pgtype.TstzrangeArrayOID,
		Codec: &pgtype.ArrayCodec{ElementType: tstzrange},
	})
	typeMap.RegisterType(tstzmultirange)
	typeMap.RegisterType(&pgtype.Type{
		Name:  "_tstzmultirange",
		OID:   pgtype.TstzmultirangeArrayOID,
		Codec: &pgtype.ArrayCodec{ElementType: tstzmultirange},
	})
}
