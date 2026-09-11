// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package queries

import "testing"

// A comment marker inside a literal is not a comment. Mis-scanning this
// produces SQL that still parses and returns a different result.
func TestStripComments(t *testing.T) {
	for _, testCase := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "a line comment and the newline after it become one space",
			in:   "SELECT 1 -- the answer\nFROM t",
			want: "SELECT 1 FROM t",
		},
		{
			name: "a whole-line comment leaves nothing behind",
			in:   "-- why this exists\nSELECT 1",
			want: "SELECT 1",
		},
		{
			name: "a block comment is removed",
			in:   "SELECT /* inline */ 1",
			want: "SELECT 1",
		},
		{
			name: "block comments nest, so the inner close does not end the outer",
			in:   "SELECT /* outer /* inner */ still comment */ 1",
			want: "SELECT 1",
		},
		{
			name: "a line comment inside a block comment is just text",
			in:   "SELECT /* a -- b */ 1",
			want: "SELECT 1",
		},
		{
			name: "two dashes inside a string are data",
			in:   "SELECT '-- not a comment' FROM t",
			want: "SELECT '-- not a comment' FROM t",
		},
		{
			name: "a block marker inside a string is data",
			in:   "SELECT '/* still data */' FROM t",
			want: "SELECT '/* still data */' FROM t",
		},
		{
			name: "a doubled quote does not close the string",
			in:   "SELECT 'it''s -- fine' -- comment\nFROM t",
			want: "SELECT 'it''s -- fine' FROM t",
		},
		{
			name: "dashes inside a quoted identifier are data",
			in:   `SELECT "weird--column" FROM t -- trailing`,
			want: `SELECT "weird--column" FROM t`,
		},
		{
			name: "dollar-quoted bodies are untouched",
			in:   "DO $$ BEGIN -- inside the body\n END $$; -- outside",
			want: "DO $$ BEGIN -- inside the body\n END $$;",
		},
		{
			name: "a tagged dollar quote is untouched",
			in:   "SELECT $tag$ -- data $tag$ -- comment",
			want: "SELECT $tag$ -- data $tag$",
		},
		{
			name: "placeholders are not dollar quotes",
			in:   "SELECT * FROM t WHERE id = $1 AND kind = $2 -- gone",
			want: "SELECT * FROM t WHERE id = $1 AND kind = $2",
		},
		{
			name: "a backslash in an ordinary string does not escape the closing quote",
			in:   `SELECT 'ends with a backslash \' AS a -- gone`,
			want: `SELECT 'ends with a backslash \' AS a`,
		},
		{
			name: "a backslash in an escape string does escape the closing quote",
			in:   `SELECT E'quote \' still inside -- data' AS a -- gone`,
			want: `SELECT E'quote \' still inside -- data' AS a`,
		},
		{
			name: "an identifier ending in e does not make the next string an escape string",
			in:   `SELECT value'x' -- gone`,
			want: `SELECT value'x'`,
		},
		{
			// Collapsing inside a literal changes the data, not the syntax.
			name: "a newline inside a string literal is content, not formatting",
			in:   "SELECT 'line one\nline two' -- gone",
			want: "SELECT 'line one\nline two'",
		},
		{
			name: "runs of spaces inside a string are content",
			in:   "SELECT 'spaced    out' -- gone",
			want: "SELECT 'spaced    out'",
		},
		{
			name: "a newline inside a quoted identifier is content",
			in:   "SELECT \"my\ncolumn\" FROM t -- gone",
			want: "SELECT \"my\ncolumn\" FROM t",
		},
		{
			name: "newlines inside a dollar-quoted body are content",
			in:   "SELECT $$body\nwith\nnewlines$$ -- gone",
			want: "SELECT $$body\nwith\nnewlines$$",
		},
		{
			// An operator name cannot contain --, so Postgres reads this as a
			// comment too, not a subtraction of -3.
			name: "two dashes against a number are still a comment",
			in:   "SELECT 5 --3",
			want: "SELECT 5",
		},
		{
			name: "runs of whitespace collapse to one space",
			in:   "SELECT\n\t1,\n\t2\nFROM   t",
			want: "SELECT 1, 2 FROM t",
		},
		{
			name: "leading and trailing whitespace is dropped",
			in:   "\n\n  SELECT 1  \n\n",
			want: "SELECT 1",
		},
		{
			name: "an unterminated block comment consumes the rest",
			in:   "SELECT 1 /* never closed",
			want: "SELECT 1",
		},
		{
			name: "a statement with no comments only loses its formatting",
			in:   "SELECT id,\n       name\nFROM users\nWHERE id = @user_id",
			want: "SELECT id, name FROM users WHERE id = @user_id",
		},
		{
			name: "an apostrophe in a block comment opens no string",
			in:   "SELECT /* don't be fooled */ 1 FROM t WHERE name = 'x'",
			want: "SELECT 1 FROM t WHERE name = 'x'",
		},
		{
			name: "a doubled double quote does not close the identifier",
			in:   `SELECT "say ""hi"" -- x" FROM t -- gone`,
			want: `SELECT "say ""hi"" -- x" FROM t`,
		},
		{
			name: "an unterminated string consumes the rest",
			in:   "SELECT 'abc",
			want: "SELECT 'abc",
		},
		{
			name: "a dollar tag may contain digits after the first character",
			in:   "SELECT $a1$ -- data $a1$ -- gone",
			want: "SELECT $a1$ -- data $a1$",
		},
		{
			name: "an inner dollar tag is body text, not a close",
			in:   "SELECT $out$ a $in$ b $out$ -- gone",
			want: "SELECT $out$ a $in$ b $out$",
		},
		{
			name: "a block marker inside a line comment opens no block comment",
			in:   "SELECT 1 -- see /* this\nFROM t",
			want: "SELECT 1 FROM t",
		},
		{
			name: "multi-byte characters in a string survive byte-level scanning",
			in:   "SELECT 'caf\u00e9' -- gone",
			want: "SELECT 'caf\u00e9'",
		},
		{
			name: "a line comment at the end of input needs no newline to close",
			in:   "SELECT 1 --",
			want: "SELECT 1",
		},
		{
			// a$b$ is one identifier. Reading $b$ as an opening delimiter
			// swallows the rest of the statement, and the comment survives.
			name: "a dollar sign inside an identifier opens no dollar quote",
			in:   "SELECT a$b$ FROM t -- gone",
			want: "SELECT a$b$ FROM t",
		},
		{
			name: "an identifier may contain a dollar sign",
			in:   "SELECT my$col FROM t WHERE x = $1 -- gone",
			want: "SELECT my$col FROM t WHERE x = $1",
		},
		{
			// Adjacent string constants concatenate only across a newline, so
			// collapsing this onto one line is a syntax error.
			name: "the newline between adjacent string literals survives",
			in:   "SELECT 'first'\n' second' -- gone",
			want: "SELECT 'first'\n' second'",
		},
		{
			name: "a comment between adjacent string literals leaves the newline",
			in:   "SELECT 'first'\n-- explanation\n' second'",
			want: "SELECT 'first'\n' second'",
		},
		{
			// Already an error in the source, and stripping does not repair it.
			name: "adjacent strings on one line stay on one line",
			in:   "SELECT 'first' ' second'",
			want: "SELECT 'first' ' second'",
		},
		{
			// Postgres refuses this, so removing the comment must not fix it.
			name: "a block comment does not carry the newline before it across",
			in:   "SELECT 'first'\n/* explanation */ ' second'",
			want: "SELECT 'first' ' second'",
		},
		{
			// Refused too: no arrangement of newlines around a block comment
			// joins the parts, since the continuation rule excludes them.
			name: "newlines either side of a block comment do not join the parts",
			in:   "SELECT 'first'\n/* explanation */\n' second'",
			want: "SELECT 'first' ' second'",
		},
		{
			name: "a newline elsewhere still collapses to a space",
			in:   "SELECT 'a',\n'b' FROM t",
			want: "SELECT 'a', 'b' FROM t",
		},
		{
			name: "a unicode escape string is scanned as an ordinary string",
			in:   `SELECT U&'d\0061t' FROM t -- gone`,
			want: `SELECT U&'d\0061t' FROM t`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := stripComments(testCase.in); got != testCase.want {
				t.Fatalf("stripComments:\n in   %q\n got  %q\n want %q", testCase.in, got, testCase.want)
			}
		})
	}
}

// Verbatim from Postgres's src/test/regress/sql/comments.sql: three levels
// of nesting, an apostrophe inside a comment, a line comment inside a block
// comment, and a line comment reaching the string on the far side.
func TestStripCommentsAgainstThePostgresRegressionCase(t *testing.T) {
	const statement = `SELECT -- continued after the following block comments...
/* Deeply nested comment.
   This includes a single apostrophe to make sure we aren't decoding this part as a string.
SELECT 'deep nest' AS n1;
/* Second level of nesting...
SELECT 'deeper nest' as n2;
/* Third level of nesting...
SELECT 'deepest nest' as n3;
*/
Hoo boy. Still two deep...
*/
Now just one deep...
*/
'deeply nested example' AS sixth;`

	const want = "SELECT 'deeply nested example' AS sixth;"
	if got := stripComments(statement); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// Naive quote tracking reads the apostrophe as a string opening and swallows
// everything to the next one.
func TestStripCommentsHandlesAnApostropheInAComment(t *testing.T) {
	const statement = "SELECT 1 -- don't be fooled\nFROM t WHERE name = 'x'"
	const want = "SELECT 1 FROM t WHERE name = 'x'"
	if got := stripComments(statement); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// A lone $ that is not a dollar quote leaves the scanner where it started, so the
// text after it is still scanned for comments. Two ways in: an identifier run that
// is not closed by a second $, and one that reaches the end of the statement.
func TestStripCommentsLeavesAnUnclosedDollarTagAlone(t *testing.T) {
	for _, testCase := range []struct {
		name string
		in   string
		want string
	}{
		{
			name: "tag not closed by a second dollar",
			in:   "SELECT $tag -- trailing\nFROM t",
			want: "SELECT $tag FROM t",
		},
		{
			name: "tag runs to the end of the statement",
			in:   "SELECT a -- trailing\nFROM t WHERE x = $tag",
			want: "SELECT a FROM t WHERE x = $tag",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := stripComments(testCase.in); got != testCase.want {
				t.Fatalf("stripComments:\n in   %q\n got  %q\n want %q", testCase.in, got, testCase.want)
			}
		})
	}
}

// An opening dollar quote with no closing delimiter swallows the rest of the
// statement, so a comment marker inside it is body text and survives. The query is
// already broken; stripping the tail would change how it is broken.
func TestStripCommentsKeepsAnUnterminatedDollarQuoteWhole(t *testing.T) {
	const statement = "SELECT $$abc -- not a comment"
	if got := stripComments(statement); got != statement {
		t.Fatalf("stripComments:\n in   %q\n got  %q\n want %q", statement, got, statement)
	}
}
