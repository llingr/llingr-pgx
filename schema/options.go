// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package schema

// Option for configuring a Migrate call
type Option func(*migrateOptions)

// migrateOptions with resolved
// configuration for a Migrate call
type migrateOptions struct {
	// FilesystemDirectory is the relative
	// subdirectory within migration's fs.FS
	FilesystemDirectory string
}

// WithFilesystemDirectory sets the sub-path
// within the embedded fs.FS SQL migrations live
func WithFilesystemDirectory(directory string) Option {
	return func(options *migrateOptions) {
		options.FilesystemDirectory = directory
	}
}

// processOptions applies each Option in order
func processOptions(options ...Option) migrateOptions {
	resolved := migrateOptions{
		// default for sql migration files in root-directory
		FilesystemDirectory: ".",
	}

	for _, option := range options {
		option(&resolved)
	}
	return resolved
}
