// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package connect

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestRegisterUTCTimesCoversEveryTimestamptzType: timestamptz and each array, range and
// multirange of it decode in UTC once registered, in the binary and the text format, as the
// instants encoded. pgx's own map, the control, decodes each in local time or, for an array
// of multiranges, which it has no type for, not at all.
func TestRegisterUTCTimesCoversEveryTimestamptzType(t *testing.T) {
	first := time.Date(2026, time.March, 29, 1, 0, 0, 0, time.UTC)
	second := time.Date(2026, time.March, 29, 2, 0, 0, 0, time.UTC)
	timeRange := pgtype.Range[pgtype.Timestamptz]{
		Lower:     pgtype.Timestamptz{Time: first, Valid: true},
		Upper:     pgtype.Timestamptz{Time: second, Valid: true},
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}
	rangeTimes := func(scanned pgtype.Range[pgtype.Timestamptz]) []time.Time {
		return []time.Time{scanned.Lower.Time, scanned.Upper.Time}
	}
	cases := []struct {
		name    string
		oid     uint32
		encoded any
		decode  func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error)
	}{
		{"timestamptz", pgtype.TimestamptzOID, first,
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned time.Time
				err := typeMap.Scan(pgtype.TimestamptzOID, format, src, &scanned)
				return []time.Time{scanned}, err
			}},
		{"_timestamptz", pgtype.TimestamptzArrayOID, []time.Time{first, second},
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned []time.Time
				err := typeMap.Scan(pgtype.TimestamptzArrayOID, format, src, &scanned)
				return scanned, err
			}},
		{"tstzrange", pgtype.TstzrangeOID, timeRange,
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned pgtype.Range[pgtype.Timestamptz]
				err := typeMap.Scan(pgtype.TstzrangeOID, format, src, &scanned)
				return rangeTimes(scanned), err
			}},
		{"_tstzrange", pgtype.TstzrangeArrayOID, []pgtype.Range[pgtype.Timestamptz]{timeRange},
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned []pgtype.Range[pgtype.Timestamptz]
				err := typeMap.Scan(pgtype.TstzrangeArrayOID, format, src, &scanned)
				if err != nil || len(scanned) != 1 {
					return nil, fmt.Errorf("decoded %v, %w; want one range", scanned, err)
				}
				return rangeTimes(scanned[0]), nil
			}},
		{"tstzmultirange", pgtype.TstzmultirangeOID,
			pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]{timeRange},
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]
				err := typeMap.Scan(pgtype.TstzmultirangeOID, format, src, &scanned)
				if err != nil || len(scanned) != 1 {
					return nil, fmt.Errorf("decoded %v, %w; want one range", scanned, err)
				}
				return rangeTimes(scanned[0]), nil
			}},
		{"_tstzmultirange", pgtype.TstzmultirangeArrayOID,
			[]pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]{{timeRange}},
			func(typeMap *pgtype.Map, format int16, src []byte) ([]time.Time, error) {
				var scanned []pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]
				err := typeMap.Scan(pgtype.TstzmultirangeArrayOID, format, src, &scanned)
				if err != nil || len(scanned) != 1 || len(scanned[0]) != 1 {
					return nil, fmt.Errorf("decoded %v, %w; want one multirange of one range",
						scanned, err)
				}
				return rangeTimes(scanned[0][0]), nil
			}},
	}
	formats := map[string]int16{
		"binary": pgtype.BinaryFormatCode,
		"text":   pgtype.TextFormatCode,
	}

	for _, testCase := range cases {
		for formatName, format := range formats {
			t.Run(testCase.name+"/"+formatName, func(t *testing.T) {
				utc := pgtype.NewMap()
				registerUTCTimes(utc)
				src, err := utc.Encode(testCase.oid, format, testCase.encoded, nil)
				if err != nil {
					t.Fatalf("encode: %v", err)
				}

				decoded, err := testCase.decode(utc, format, src)
				if err != nil {
					t.Fatalf("decode: %v", err)
				}
				wanted := []time.Time{first, second}
				if len(decoded) == 0 || len(decoded) > len(wanted) {
					t.Fatalf("decoded %d times, want 1 or 2", len(decoded))
				}
				for index, scanned := range decoded {
					if scanned.Location() != time.UTC || !scanned.Equal(wanted[index]) {
						t.Errorf("time %d decoded as %s in %q, want %s in UTC", index, scanned,
							scanned.Location(), wanted[index])
					}
				}

				controlled, err := testCase.decode(pgtype.NewMap(), format, src)
				if err != nil {
					return
				}
				for index, scanned := range controlled {
					if scanned.Location() == time.UTC {
						t.Errorf("pgx's own map decoded time %d in UTC, so this case would pass "+
							"without the registration", index)
					}
				}
			})
		}
	}
}
