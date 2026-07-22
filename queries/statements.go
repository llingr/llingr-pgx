// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

// Package queries loads named SQL statements from plain .sql files held in a
// typically embedded fs.FS and returns them by name. There is no ORM and
// no query builder: queries are ordinary SQL as text, marked with
// `-- name: <identity>`
//
// A .sql file is a sequence of named blocks:
//
//	-- name: all-users
//	SELECT id, name
//	FROM users;
//
//	-- name: user-by-id
//	SELECT id, name
//	FROM users
//	WHERE id = @user_id;
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

// Statements is a read-only, name-indexed set of
// SQL statements loaded from .sql files
type Statements interface {
	// SQL returns the SQL text for name, or the empty string if name is not
	// defined. Statements are never empty (empty bodies are rejected at load),
	// so an empty result unambiguously means "not defined".
	SQL(name string) string
	// Names returns the defined query names in sorted order.
	Names() []string
}

// statements is the concrete Statements implementation.
type statements struct {
	namedSQLFragments map[string]string
}

var _ Statements = (*statements)(nil)

// Load walks filesystem and parses every file with a ".sql" suffix into
// named SQL statements. It returns these as Statements. Names must be unique
// across all files: a duplicate is reported as an error rather than shadowing.
func Load(fileSystem fs.FS) (Statements, error) {
	byName := make(map[string]string)
	origin := make(map[string]string) // name -> file that defined it

	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}
		data, errR := fs.ReadFile(fileSystem, path)
		if errR != nil {
			return fmt.Errorf("queries: read %s: %w", path, errR)
		}
		named, errP := parse(string(data))
		if errP != nil {
			return fmt.Errorf("queries: parse %s: %w", path, errP)
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
	return &statements{namedSQLFragments: byName}, nil
}

// parse breaks a single .sql file into a name->SQL map
//
// Content before the first marker is ignored; a marker with
// an empty body, or a repeated name, is an error.
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

// SQL statement for the named query
func (f *statements) SQL(name string) string {
	return f.namedSQLFragments[name]
}

// Names all query statements
func (f *statements) Names() []string {
	names := make([]string, 0, len(f.namedSQLFragments))
	for n := range f.namedSQLFragments {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
