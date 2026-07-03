package schema

import (
	"strings"
	"testing"

	roles2 "github.com/llingr/llingr-pgx/roles"
)

func TestSubstitutePlaceholders(t *testing.T) {
	usernames := map[string]string{
		"application_user": "ecommerce_app_user",
		"readonly_user":    "ecommerce_readonly_user",
	}
	input := `GRANT SELECT, INSERT ON customers TO :"application_user";
GRANT SELECT ON customers TO :"readonly_user";`

	output, err := substitutePlaceholders(input, usernames)
	if err != nil {
		t.Fatalf("substitute: %v", err)
	}
	if !strings.Contains(output, `TO "ecommerce_app_user"`) || !strings.Contains(output, `TO "ecommerce_readonly_user"`) {
		t.Fatalf("usernames not substituted as quoted identifiers:\n%s", output)
	}
	if strings.Contains(output, `:"`) {
		t.Fatalf("placeholders remain:\n%s", output)
	}
}

func TestSubstitutePlaceholders_UnknownPlaceholderIsHardError(t *testing.T) {
	_, err := substitutePlaceholders(
		`GRANT SELECT ON x TO :"reporting_user";`,
		map[string]string{"application_user": "a"},
	)
	if err == nil || !strings.Contains(err.Error(), "reporting_user") {
		t.Fatalf("expected unknown-placeholder error naming reporting_user, got %v", err)
	}
}

func TestSubstitutePlaceholders_LeavesCastsAlone(t *testing.T) {
	input := `SELECT customer_id::"text" FROM t;`

	output, err := substitutePlaceholders(input, map[string]string{})
	if err != nil {
		t.Fatalf("substitute: %v", err)
	}
	if output != input {
		t.Fatalf("a cast was altered:\n got %q\nwant %q", output, input)
	}
}

func TestRolePlaceholderUsernames(t *testing.T) {
	roles := roles2.NewPlaceholderBuilder().
		WithOwner("ecommerce_schema_owner").
		WithApp("ecommerce_app_user").
		WithCustom("custom_readonly_user", "ecommerce_readonly_user").
		MustBuild()

	if roles["owner"] != "ecommerce_schema_owner" {
		t.Fatalf("unexpected placeholder usernames: %+v", roles)
	}

	if roles["app"] != "ecommerce_app_user" {
		t.Fatalf("unexpected placeholder usernames: %+v", roles)
	}

	if roles["custom_readonly_user"] != "ecommerce_readonly_user" {
		t.Fatalf("unexpected placeholder usernames: %+v", roles)
	}
}
