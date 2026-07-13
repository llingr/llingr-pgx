// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/llingr/llingr-pgx/roles"
)

// placeholderRegex matches a psql-style quoted-identifier variable like
// :"application_user". A match preceded by a second colon is the tail of a
// ::"type" cast, not a placeholder, and is left alone.
var placeholderRegex = regexp.MustCompile(`:"([A-Za-z_][A-Za-z0-9_]*)"`)

// substitutePlaceholders replaces each :"name" placeholder with the matching
// value, quoted as an identifier. An unmatched placeholder is a hard error, so a
// migration referring to a value that was never supplied fails here, not later.
func substitutePlaceholders(content string, valuesByPlaceholder map[string]string) (string, error) {
	var builder strings.Builder
	var failures []error
	lastEnd := 0

	for _, match := range placeholderRegex.FindAllStringSubmatchIndex(content, -1) {
		start, end := match[0], match[1]
		nameStart, nameEnd := match[2], match[3]

		if start > 0 && content[start-1] == ':' {
			continue
		}

		placeholder := content[nameStart:nameEnd]
		value, found := valuesByPlaceholder[placeholder]
		if !found {
			failures = append(failures, fmt.Errorf("unknown placeholder :%q", placeholder))
			continue
		}

		builder.WriteString(content[lastEnd:start])
		builder.WriteString(quoteIdentifier(value))
		lastEnd = end
	}

	if len(failures) > 0 {
		return "", errors.Join(failures...)
	}

	builder.WriteString(content[lastEnd:])
	return builder.String(), nil
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func rolePlaceholderUsernames(roleUsernames roles.PlaceholderUsernames) map[string]string {
	usernamesByPlaceholder := make(map[string]string, len(roleUsernames))
	for placeholder, username := range roleUsernames {
		usernamesByPlaceholder[placeholder.String()] = username.String()
	}
	return usernamesByPlaceholder
}

// templatingFS wraps a migration fs.FS so reading a file returns its content with
// the :"name" placeholders substituted. Directory listing is delegated unchanged.
type templatingFS struct {
	inner                  fs.FS
	usernamesByPlaceholder map[string]string
}

func (templated templatingFS) Open(name string) (fs.File, error) {
	file, err := templated.inner.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if info.IsDir() {
		return file, nil
	}

	content, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		return nil, err
	}

	substituted, err := substitutePlaceholders(string(content), templated.usernamesByPlaceholder)
	if err != nil {
		return nil, fmt.Errorf("template %s: %w", name, err)
	}

	return &memoryFile{name: info.Name(), content: []byte(substituted), modTime: info.ModTime()}, nil
}

func (templated templatingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(templated.inner, name)
}

type memoryFile struct {
	name    string
	content []byte
	modTime time.Time
	offset  int
}

func (file *memoryFile) Stat() (fs.FileInfo, error) {
	return memoryFileInfo{file: file}, nil
}

func (file *memoryFile) Read(buffer []byte) (int, error) {
	if file.offset >= len(file.content) {
		return 0, io.EOF
	}
	copied := copy(buffer, file.content[file.offset:])
	file.offset += copied
	return copied, nil
}

func (file *memoryFile) Close() error {
	return nil
}

type memoryFileInfo struct {
	file *memoryFile
}

func (info memoryFileInfo) Name() string {
	return info.file.name
}

func (info memoryFileInfo) Size() int64 {
	return int64(len(info.file.content))
}

func (info memoryFileInfo) Mode() fs.FileMode {
	return 0o444
}

func (info memoryFileInfo) ModTime() time.Time {
	return info.file.modTime
}

func (info memoryFileInfo) IsDir() bool {
	return false
}

func (info memoryFileInfo) Sys() any {
	return nil
}
