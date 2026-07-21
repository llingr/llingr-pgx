// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

// Package queries loads named SQL fragments from plain .sql files held in an
// embedded (or any other) fs.FS and returns them by name. There is no ORM and
// no query builder: you write ordinary SQL as text, mark each statement with a
// `-- name: <ident>` comment, and look it up at runtime.
//
// A .sql file is a sequence of named blocks:
//
//	-- name: all-users
//	SELECT id, name FROM users;
//
//	-- name: user-by-id
//	SELECT id, name FROM users WHERE id = @user_id;
//
// Any content before the first marker (a file header comment, say) is ignored.
// Typical use embeds the files and loads them once at startup:
//
//	//go:embed queries/*.sql
//	var queryFS embed.FS
//
//	q, err := queries.Load(queryFS)
//	...
//	rows, err := db.Query(ctx, q.SQL("user-by-id"), pgx.NamedArgs{"user_id": 42})
package queries

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// markerRegex matches a block marker line such as `-- name: get-user`. Leading
// whitespace and flexible spacing around the marker are tolerated; the name is
// a single whitespace-delimited token.
var markerRegex = regexp.MustCompile(`^\s*--\s*name:\s*(\S+)\s*$`)

// Fragments is a read-only, name-indexed set of SQL statements loaded from
// .sql files. Obtain one with Load.
type Fragments interface {
	// SQL returns the SQL text for name, or the empty string if name is not
	// defined. Fragments are never empty (empty bodies are rejected at load),
	// so an empty result unambiguously means "not defined".
	SQL(name string) string
	// Names returns the defined query names in sorted order.
	Names() []string
}

// fragments is the concrete, immutable Fragments implementation.
type fragments struct {
	namedSQLFragments map[string]string
}

var _ Fragments = (*fragments)(nil)

// Load walks fsys, parses every file with a ".sql" suffix (case-insensitive)
// into named SQL fragments, and returns them as a Fragments. Names must be
// unique across all files: a duplicate is reported as an error rather than
// silently shadowing an earlier query.
//
// The set of files is whatever fsys exposes, so scope inclusion at the embed
// site (for example //go:embed queries/*.sql) rather than here.
func Load(fsys fs.FS) (Fragments, error) {
	byName := make(map[string]string)
	origin := make(map[string]string) // name -> file that defined it

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Case-insensitive: an embed pattern like *.sql would never surface FILE.SQL
		// anyway, but a broader pattern (or a non-embed fs.FS) can.
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("queries: read %s: %w", path, err)
		}
		named, err := parse(string(data))
		if err != nil {
			return fmt.Errorf("queries: parse %s: %w", path, err)
		}
		for name, sql := range named {
			if prev, dup := origin[name]; dup {
				return fmt.Errorf("queries: duplicate query %q in %s (already defined in %s)", name, path, prev)
			}
			byName[name] = sql
			origin[name] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &fragments{namedSQLFragments: byName}, nil
}

// parse breaks a single .sql file into a name->SQL map. Content before the
// first marker is ignored; a marker with an empty body, or a name repeated
// within the same file, is an error.
func parse(content string) (map[string]string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	out := make(map[string]string)

	var name string
	var body []string

	flush := func() error {
		if name == "" {
			return nil
		}
		sql := strings.TrimSpace(strings.Join(body, "\n"))
		if sql == "" {
			return fmt.Errorf("query %q has no SQL body", name)
		}
		if _, dup := out[name]; dup {
			return fmt.Errorf("duplicate query %q in the same file", name)
		}
		out[name] = sql
		return nil
	}

	for _, line := range strings.Split(content, "\n") {
		if m := markerRegex.FindStringSubmatch(line); m != nil {
			if err := flush(); err != nil {
				return nil, err
			}
			name = m[1]
			body = body[:0]
			continue
		}
		if name != "" {
			body = append(body, line)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *fragments) SQL(name string) string {
	return f.namedSQLFragments[name]
}

func (f *fragments) Names() []string {
	names := make([]string, 0, len(f.namedSQLFragments))
	for n := range f.namedSQLFragments {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
