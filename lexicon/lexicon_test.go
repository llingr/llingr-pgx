package lexicon

import (
	"errors"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// --- helpers ---------------------------------------------------------------

func file(parts ...string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(strings.Join(parts, ""))}
}

func mustLoad(t *testing.T, fsys fs.FS) Fragments {
	t.Helper()
	set, err := Load(fsys)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	return set
}

func wantSQL(t *testing.T, set Fragments, name, want string) {
	t.Helper()
	if got := set.SQL(name); got != want {
		t.Errorf("SQL(%q) = %q, want %q", name, got, want)
	}
}

// --- parsing & accessors ---------------------------------------------------

func TestLoad_ParsesNamedBlocks(t *testing.T) {
	set := mustLoad(t, fstest.MapFS{
		"users.sql": file(
			"-- a header comment before any marker is ignored\n",
			"-- name: all-users\n",
			"SELECT id, name FROM users;\n",
			"\n",
			"-- name: user-by-id\n",
			"SELECT id, name\n",
			"FROM users\n",
			"WHERE id = $1;\n",
		),
	})

	if got := len(set.Names()); got != 2 {
		t.Fatalf("loaded %d fragments, want 2", got)
	}
	// single line
	wantSQL(t, set, "all-users", "SELECT id, name FROM users;")
	// multi-line body keeps internal newlines, trimmed at the edges
	wantSQL(t, set, "user-by-id", "SELECT id, name\nFROM users\nWHERE id = $1;")
}

func TestLoad_InlineCommentsStayInBody(t *testing.T) {
	// A `--` comment that is not a `name:` marker is ordinary SQL and is kept.
	set := mustLoad(t, fstest.MapFS{
		"q.sql": file(
			"-- name: counted\n",
			"-- count active rows\n",
			"SELECT count(*) FROM t WHERE active;\n",
		),
	})
	wantSQL(t, set, "counted", "-- count active rows\nSELECT count(*) FROM t WHERE active;")
}

func TestSQL(t *testing.T) {
	set := mustLoad(t, fstest.MapFS{
		"q.sql": file("-- name: ping\nSELECT 1;\n"),
	})

	wantSQL(t, set, "ping", "SELECT 1;")
	// fragments are never empty, so "" unambiguously means "not defined"
	wantSQL(t, set, "missing", "")
}

func TestNames_Sorted(t *testing.T) {
	set := mustLoad(t, fstest.MapFS{
		"z.sql": file("-- name: zebra\nSELECT 1;\n-- name: apple\nSELECT 2;\n"),
		"m.sql": file("-- name: mango\nSELECT 3;\n"),
	})

	got := set.Names()
	if !sort.StringsAreSorted(got) {
		t.Errorf("Names not sorted: %v", got)
	}
	if want := []string{"apple", "mango", "zebra"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Names = %v, want %v", got, want)
	}
}

// --- directory walking (root + sub-directories) ----------------------------

func TestLoad_WalksRootAndSubdirectories(t *testing.T) {
	// .sql files at the root and nested arbitrarily deep are all discovered;
	// non-.sql files anywhere are ignored. The suffix match is case-insensitive
	// (matters for non-embed sources; //go:embed *.sql would exclude .SQL itself).
	set := mustLoad(t, fstest.MapFS{
		"root.sql":           file("-- name: q_root\nSELECT 0;\n"),
		"UPPER.SQL":          file("-- name: q_upper\nSELECT 9;\n"),
		"a/one.sql":          file("-- name: q_a\nSELECT 1;\n"),
		"a/b/two.sql":        file("-- name: q_ab\nSELECT 2;\n"),
		"a/b/c/three.sql":    file("-- name: q_abc\nSELECT 3;\n"),
		"a/notes.md":         file("# not sql"),
		"a/b/data.json":      file("{}"),
		"a/b/c/d/README.txt": file("ignore me"),
	})

	want := []string{"q_a", "q_ab", "q_abc", "q_root", "q_upper"}
	if got := set.Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names = %v, want %v (sub-directory .sql files must all load)", got, want)
	}
	// prove the deepest one resolves to its body
	wantSQL(t, set, "q_abc", "SELECT 3;")
}

func TestLoad_NoSQLFilesIsEmptyNotError(t *testing.T) {
	set := mustLoad(t, fstest.MapFS{
		"readme.md": file("hi"),
		"dir/x.txt": file("x"),
	})
	if got := len(set.Names()); got != 0 {
		t.Fatalf("loaded %d fragments, want 0", got)
	}

	// a completely empty filesystem is also fine
	if set := mustLoad(t, fstest.MapFS{}); len(set.Names()) != 0 {
		t.Fatalf("empty FS loaded %d fragments, want 0", len(set.Names()))
	}
}

// --- marker syntax tolerance ----------------------------------------------

func TestParse_MarkerSpacingVariants(t *testing.T) {
	cases := map[string]string{ // source -> expected name
		"-- name: a\nSELECT 1;":        "a",
		"--name:b\nSELECT 1;":          "b",
		"--   name:   c   \nSELECT 1;": "c",
		"\t-- name: d\nSELECT 1;":      "d", // leading tab
		"-- name: get-user\nSELECT 1;": "get-user",
	}
	for src, wantName := range cases {
		got, err := parse(src)
		if err != nil {
			t.Errorf("parse(%q): %v", src, err)
			continue
		}
		if _, ok := got[wantName]; !ok {
			t.Errorf("parse(%q): missing %q (got names %v)", src, wantName, names(got))
		}
	}
}

func TestParse_CRLF(t *testing.T) {
	got, err := parse("-- name: a\r\nSELECT 1;\r\nSELECT 2;\r\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got["a"] != "SELECT 1;\nSELECT 2;" {
		t.Errorf("CRLF body = %q", got["a"])
	}
}

func TestParse_LineWithoutNameTokenIsNotAMarker(t *testing.T) {
	// `-- name:` with no identifier is not a marker; it is body text, so with no
	// preceding real marker it is simply ignored (no query is produced).
	got, err := parse("-- name:\nSELECT 1;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no queries, got %v", names(got))
	}
}

// --- error cases -----------------------------------------------------------

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		fsys    fstest.MapFS
		wantSub string
	}{
		{
			name: "duplicate name across files",
			fsys: fstest.MapFS{
				"a.sql": file("-- name: dup\nSELECT 1;\n"),
				"b.sql": file("-- name: dup\nSELECT 2;\n"),
			},
			wantSub: "duplicate query",
		},
		{
			name: "duplicate name across sub-directories",
			fsys: fstest.MapFS{
				"x/a.sql":   file("-- name: dup\nSELECT 1;\n"),
				"y/z/b.sql": file("-- name: dup\nSELECT 2;\n"),
			},
			wantSub: "duplicate query",
		},
		{
			name: "duplicate name within one file",
			fsys: fstest.MapFS{
				"a.sql": file("-- name: dup\nSELECT 1;\n-- name: dup\nSELECT 2;\n"),
			},
			wantSub: "duplicate query",
		},
		{
			name: "empty body between markers",
			fsys: fstest.MapFS{
				"a.sql": file("-- name: empty\n\n-- name: ok\nSELECT 1;\n"),
			},
			wantSub: "no SQL body",
		},
		{
			name: "empty body at end of file",
			fsys: fstest.MapFS{
				"a.sql": file("-- name: trailing\n   \n"),
			},
			wantSub: "no SQL body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			set, err := Load(tc.fsys)
			if err == nil {
				t.Fatalf("want error containing %q, got nil (loaded %d)", tc.wantSub, len(set.Names()))
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoad_DuplicateErrorNamesBothFiles(t *testing.T) {
	// Deterministic (WalkDir is sorted): the later file is the offender and the
	// earlier file is named as the original.
	_, err := Load(fstest.MapFS{
		"a.sql": file("-- name: dup\nSELECT 1;\n"),
		"b.sql": file("-- name: dup\nSELECT 2;\n"),
	})
	if err == nil {
		t.Fatal("want duplicate error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "b.sql") || !strings.Contains(msg, "a.sql") {
		t.Errorf("error %q should mention both a.sql and b.sql", msg)
	}
}

func TestLoad_ReadErrorPropagates(t *testing.T) {
	fsys := readErrFS{
		FS:     fstest.MapFS{"bad.sql": file("-- name: q\nSELECT 1;\n")},
		failOn: "bad.sql",
	}
	_, err := Load(fsys)
	if err == nil || !strings.Contains(err.Error(), "read bad.sql") {
		t.Fatalf("want read error for bad.sql, got %v", err)
	}
}

// --- read-error injection FS ----------------------------------------------

// readErrFS wraps an fs.FS and makes Read fail for one named file, exercising
// Load's file-read error branch.
type readErrFS struct {
	fs.FS
	failOn string
}

func (f readErrFS) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, err
	}
	if name == f.failOn {
		return errFile{file}, nil
	}
	return file, nil
}

type errFile struct{ fs.File }

func (errFile) Read([]byte) (int, error) { return 0, errors.New("injected read failure") }

// names returns the keys of a parsed map, for readable failure messages.
func names(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
