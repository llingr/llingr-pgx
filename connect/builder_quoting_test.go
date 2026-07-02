package connect

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// quoteKeywordValue is the load-bearing bit of DSN rendering: AWS RDS and similar
// generate passwords full of characters that break a naive keyword=value string.
// These cases pin its exact output, char by char, so the escaping can never silently
// regress. A value is quoted only when it is empty or contains a space, single quote,
// or backslash; inside the quotes, backslash and single quote are backslash-escaped.
func TestQuoteKeywordValue(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain stays unquoted", "simplepw", "simplepw"},
		{"empty is quoted", "", "''"},
		{"space forces quotes", "a b", "'a b'"},
		{"tab forces quotes", "a\tb", "'a\tb'"},
		{"newline forces quotes", "a\nb", "'a\nb'"},
		{"single quote escaped", "a'b", `'a\'b'`},
		{"backslash escaped", `a\b`, `'a\\b'`},
		{"slash plus equals percent need no quoting", "a/b+c=d%e", "a/b+c=d%e"},
		{"colon and at need no quoting", "p@ss:word", "p@ss:word"},
		{"quote and backslash together", `a'\b`, `'a\'\\b'`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := quoteKeywordValue(c.in); got != c.want {
				t.Errorf("quoteKeywordValue(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// Whatever nasty password goes in, pgx's own parser must recover it byte for byte from
// both the DSN and the URL rendering. This is the guarantee that actually matters in
// production, and it runs offline in milliseconds rather than as a commit/CI/deploy
// cycle against a real RDS or Neon endpoint. The set deliberately includes the escape
// character of each format: backslash (DSN) and percent (URL).
func TestBuilder_SpecialCharPasswordsRoundTrip(t *testing.T) {
	passwords := []string{
		`a/b+c=d e'f`,     // the classic RDS mix: slash, plus, equals, space, single quote
		`50%off+more`,     // percent, the URL escape character itself
		`back\slash`,      // backslash, the DSN escape character
		`quote'and\slash`, // both escape characters together
		`p@ss:w/rd`,       // at-sign and colon, the URL userinfo delimiters
		`spaces and more`, // multiple spaces
		"tab\tnew\nline",  // whitespace beyond space: unquoted, these break DSN parsing outright
	}
	for _, pw := range passwords {
		builder := NewConnectionBuilder().
			WithHost("db.example.com").WithPort(5432).
			WithUser("app").WithPassword(pw).WithDatabase("appdb").
			WithSSLMode("require")

		for _, style := range []struct{ name, str string }{
			{"DSN", builder.DSN()},
			{"PSQL", builder.PSQL()},
		} {
			config, err := pgxpool.ParseConfig(style.str)
			if err != nil {
				t.Errorf("ParseConfig(%s) for password %q: %v\n  rendered: %s", style.name, pw, err, style.str)
				continue
			}
			if config.ConnConfig.Password != pw {
				t.Errorf("%s password round-trip: got %q, want %q\n  rendered: %s",
					style.name, config.ConnConfig.Password, pw, style.str)
			}
		}
	}
}

// A password with no user (the user arriving via PGUSER, say) must survive both
// renderings. The URL form cannot place it in the userinfo, so it travels as a
// password= query parameter rather than being silently dropped.
func TestBuilder_PasswordWithoutUserSurvivesBothRenderings(t *testing.T) {
	builder := NewConnectionBuilder().WithHost("db.example.com").WithPassword("secret")

	for _, style := range []struct{ name, str string }{
		{"DSN", builder.DSN()},
		{"PSQL", builder.PSQL()},
	} {
		config, err := pgxpool.ParseConfig(style.str)
		if err != nil {
			t.Fatalf("ParseConfig(%s): %v\n  rendered: %s", style.name, err, style.str)
		}
		if config.ConnConfig.Password != "secret" {
			t.Errorf("%s dropped the password: got %q\n  rendered: %s",
				style.name, config.ConnConfig.Password, style.str)
		}
	}
}
