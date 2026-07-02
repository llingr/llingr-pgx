package roles_test

import (
	"testing"

	"github.com/llingr/llingr-pgx/roles"
)

// readOnlyRole is an application-defined custom role used across these tests,
// mirroring the :"readonly" placeholder the migrations reference. The library
// ships only AdminOwnerRole and AppRole as built-ins; everything else (such as
// read-only) is supplied by the application via WithCustom.
const readOnlyRole roles.Placeholder = "readonly"

// The builder requires at least one role and mandates no specific one: admin_owner,
// app, or a custom role each suffice on their own. An empty builder is the only
// hard error, since an empty placeholder->username map is pointless.
func TestBuilder_RequiresAtLeastOneRole(t *testing.T) {
	if _, err := roles.NewPlaceholderBuilder().Build(); err == nil {
		t.Fatal("empty builder should error")
	}
	if _, err := roles.NewPlaceholderBuilder().WithAdminOwner("ecommerce_admin_user").Build(); err != nil {
		t.Errorf("admin_owner-only should build: %v", err)
	}
	if _, err := roles.NewPlaceholderBuilder().WithApp("ecommerce_app_user").Build(); err != nil {
		t.Errorf("app-only should build: %v", err)
	}
	if _, err := roles.NewPlaceholderBuilder().WithCustom(readOnlyRole, "ecommerce_readonly_user").Build(); err != nil {
		t.Errorf("custom-only should build: %v", err)
	}
}

// admin_owner (built-in), app (built-in) and a custom read-only role: the
// canonical setup the migrations expect (:"app" and :"readonly"). Confirms each
// wrapper maps to the right placeholder key and value, and nothing extra is added.
func TestBuilder_WrappersMapToPlaceholders(t *testing.T) {
	got := roles.NewPlaceholderBuilder().
		WithAdminOwner("ecommerce_admin_user").
		WithApp("ecommerce_app_user").
		WithCustom(readOnlyRole, "ecommerce_readonly_user").
		MustBuild()

	want := map[roles.Placeholder]roles.Username{
		roles.AdminOwnerRole: "ecommerce_admin_user",
		roles.AppRole:        "ecommerce_app_user",
		readOnlyRole:         "ecommerce_readonly_user",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for role, username := range want {
		if got[role] != username {
			t.Errorf("%s = %q, want %q", role, got[role], username)
		}
	}
}

// Build validates every role and username; invalid identifiers fail fast. The
// full identifier matrix lives in role_username_test.go / role_placeholder_test.go;
// here we only confirm the builder surfaces the failure.
func TestBuilder_InvalidUsernameIsError(t *testing.T) {
	if _, err := roles.NewPlaceholderBuilder().WithApp("").Build(); err == nil {
		t.Error("empty username should error")
	}
	if _, err := roles.NewPlaceholderBuilder().WithApp("not a valid ident").Build(); err == nil {
		t.Error("non-identifier username should error")
	}
}

// MustBuild returns the map on valid input and panics on invalid input.
func TestBuilder_MustBuild(t *testing.T) {
	got := roles.NewPlaceholderBuilder().WithApp("ecommerce_app_user").MustBuild()
	if got[roles.AppRole] != "ecommerce_app_user" {
		t.Fatalf("MustBuild returned %+v", got)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("MustBuild should panic on invalid input")
		}
	}()
	roles.NewPlaceholderBuilder().WithApp("").MustBuild()
}
