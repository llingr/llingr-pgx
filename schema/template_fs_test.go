package schema

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/llingr/llingr-pgx/roles"
)

// newTemplatingFS wraps an in-memory FS (one SQL file referencing :"app", plus a
// subdirectory) so the templatingFS file, directory, and FileInfo paths can all be
// exercised without a database.
func newTemplatingFS() templatingFS {
	return templatingFS{
		inner: fstest.MapFS{
			"001_customers.up.sql": &fstest.MapFile{
				Data: []byte(`GRANT SELECT ON customers TO :"app";`),
				Mode: 0o644,
			},
			"sub/002_more.up.sql": &fstest.MapFile{
				Data: []byte(`-- nothing to substitute`),
				Mode: 0o644,
			},
		},
		usernamesByPlaceholder: map[string]string{"app": "app_readwrite"},
	}
}

// Open on a file substitutes placeholders and returns a memoryFile whose FileInfo
// reports the substituted content. Covers memoryFile.Read/Stat/Close and every
// memoryFileInfo method.
func TestTemplatingFS_OpenSubstitutesAndExposesFileInfo(t *testing.T) {
	tfs := newTemplatingFS()

	file, err := tfs.Open("001_customers.up.sql")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Name() != "001_customers.up.sql" {
		t.Errorf("Name = %q", info.Name())
	}
	if info.IsDir() {
		t.Error("IsDir = true, want false")
	}
	if info.Mode() != 0o444 {
		t.Errorf("Mode = %v, want 0444", info.Mode())
	}
	if info.Sys() != nil {
		t.Errorf("Sys = %v, want nil", info.Sys())
	}
	_ = info.ModTime() // exercise the accessor

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := `GRANT SELECT ON customers TO "app_readwrite";`
	if string(content) != want {
		t.Fatalf("substituted content = %q, want %q", content, want)
	}
	if info.Size() != int64(len(want)) {
		t.Errorf("Size = %d, want %d", info.Size(), len(want))
	}

	// A second read past the end returns EOF (the offset>=len branch).
	if n, err := file.Read(make([]byte, 4)); n != 0 || err != io.EOF {
		t.Fatalf("read past end = (%d, %v), want (0, EOF)", n, err)
	}
}

// Opening a directory returns the inner entry unchanged (the IsDir short-circuit),
// not a templated memoryFile.
func TestTemplatingFS_OpenDirectoryReturnsRawDirEntry(t *testing.T) {
	tfs := newTemplatingFS()
	dir, err := tfs.Open("sub")
	if err != nil {
		t.Fatalf("open dir: %v", err)
	}
	defer func() { _ = dir.Close() }()

	info, err := dir.Stat()
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory")
	}
}

// ReadDir is delegated to the inner FS unchanged.
func TestTemplatingFS_ReadDirDelegates(t *testing.T) {
	tfs := newTemplatingFS()
	entries, err := tfs.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries, got none")
	}
}

// A missing file surfaces the inner FS error.
func TestTemplatingFS_OpenMissingFileErrors(t *testing.T) {
	tfs := newTemplatingFS()
	if _, err := tfs.Open("nope.sql"); err == nil {
		t.Fatal("expected error opening missing file")
	}
}

// A placeholder with no mapped username fails at Open, naming the file.
func TestTemplatingFS_OpenUnknownPlaceholderErrors(t *testing.T) {
	tfs := templatingFS{
		inner: fstest.MapFS{
			"x.up.sql": &fstest.MapFile{Data: []byte(`GRANT SELECT ON x TO :"missing";`)},
		},
		usernamesByPlaceholder: map[string]string{"app": "app_readwrite"},
	}
	_, err := tfs.Open("x.up.sql")
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected unknown-placeholder error naming the placeholder, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "x.up.sql") {
		t.Fatalf("error should name the file, got %v", err)
	}
}

// errFS returns a file that opens cleanly but then fails its Stat or Read, modelling
// a real fs.FS that fails mid-operation (fstest.MapFS never does), to cover
// templatingFS.Open's two defensive error branches.
type errFS struct{ statErr, readErr bool }

func (e errFS) Open(string) (fs.File, error) {
	return &errFile{statErr: e.statErr, readErr: e.readErr}, nil
}

type errFile struct{ statErr, readErr bool }

func (f *errFile) Stat() (fs.FileInfo, error) {
	if f.statErr {
		return nil, errors.New("stat boom")
	}
	return errInfo{}, nil
}

func (f *errFile) Read([]byte) (int, error) {
	if f.readErr {
		return 0, errors.New("read boom")
	}
	return 0, io.EOF
}

func (f *errFile) Close() error { return nil }

type errInfo struct{}

func (errInfo) Name() string       { return "x.up.sql" }
func (errInfo) Size() int64        { return 0 }
func (errInfo) Mode() fs.FileMode  { return 0 }
func (errInfo) ModTime() time.Time { return time.Time{} }
func (errInfo) IsDir() bool        { return false }
func (errInfo) Sys() any           { return nil }

// A Stat failure after a successful Open is propagated (and the file closed).
func TestTemplatingFS_OpenStatErrorPropagates(t *testing.T) {
	tfs := templatingFS{inner: errFS{statErr: true}, usernamesByPlaceholder: map[string]string{}}
	if _, err := tfs.Open("x.up.sql"); err == nil {
		t.Fatal("expected the stat error to propagate")
	}
}

// A read failure while slurping the file content is propagated.
func TestTemplatingFS_OpenReadErrorPropagates(t *testing.T) {
	tfs := templatingFS{inner: errFS{readErr: true}, usernamesByPlaceholder: map[string]string{}}
	if _, err := tfs.Open("x.up.sql"); err == nil {
		t.Fatal("expected the read error to propagate")
	}
}

// rolePlaceholderUsernames keys the map by each placeholder's string value.
func TestRolePlaceholderUsernames_MapsByStringValue(t *testing.T) {
	got := rolePlaceholderUsernames(map[roles.Placeholder]roles.Username{
		roles.AdminOwnerRole: "owner",
		roles.AppRole:        "app_rw",
	})
	if got["admin_owner"] != "owner" || got["app"] != "app_rw" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}
