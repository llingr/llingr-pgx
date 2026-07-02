package schema

import "testing"

// With no options, processOptions returns the defaults: migrations are read from
// the root of the embedded fs.FS.
func TestProcessOptions_DefaultsToRootDirectory(t *testing.T) {
	resolved := processOptions()
	if resolved.FilesystemDirectory != "." {
		t.Errorf("default FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, ".")
	}
}

// WithFilesystemDirectory overrides the default sub-path.
func TestWithFilesystemDirectory_Overrides(t *testing.T) {
	resolved := processOptions(WithFilesystemDirectory("migrations"))
	if resolved.FilesystemDirectory != "migrations" {
		t.Errorf("FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, "migrations")
	}
}

// Options apply in order, so a later option wins over an earlier one.
func TestProcessOptions_LastOptionWins(t *testing.T) {
	resolved := processOptions(
		WithFilesystemDirectory("first"),
		WithFilesystemDirectory("second"),
	)
	if resolved.FilesystemDirectory != "second" {
		t.Errorf("FilesystemDirectory = %q, want %q", resolved.FilesystemDirectory, "second")
	}
}
