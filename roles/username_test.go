// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package roles_test

import (
	"strings"
	"testing"

	"github.com/llingr/llingr-pgx/roles"
)

func TestUsername_String(t *testing.T) {
	if roles.Username("ecommerce_app_user").String() != "ecommerce_app_user" {
		t.Fatal("String() mismatch")
	}
}

// Validate enforces the plain-SQL-identifier rule on usernames. This is where the
// empty-username case is covered directly (rather than through the builder).
func TestUsername_Validate(t *testing.T) {
	valid := []roles.Username{
		"app", "ecommerce_app_user", "ecommerce_readonly_user", "_svc", "u1",
	}
	for _, u := range valid {
		if err := u.Validate(); err != nil {
			t.Errorf("%q should be valid: %v", u, err)
		}
	}

	invalid := []roles.Username{
		"",                     // empty
		"1user",                // leading digit
		"user-name",            // hyphen
		"user name",            // space
		"robert'); DROP TABLE", // injection attempt
	}
	for _, u := range invalid {
		if err := u.Validate(); err == nil {
			t.Errorf("%q should be invalid", u)
		}
	}
}

func TestUsername_ErrorMentionsSQLIdentifier(t *testing.T) {
	err := roles.Username("bad name").Validate()
	if err == nil || !strings.Contains(err.Error(), "valid SQL identifier") {
		t.Fatalf("expected SQL-identifier error, got %v", err)
	}
}

// Postgres truncates identifiers longer than 63 bytes (NAMEDATALEN-1) silently,
// so Validate rejects them: 63 bytes is the last valid length, 64 the first invalid.
func TestUsername_LengthLimit(t *testing.T) {
	atLimit := roles.Username(strings.Repeat("a", roles.MaxIdentifierBytes))
	if err := atLimit.Validate(); err != nil {
		t.Errorf("63-byte username should be valid: %v", err)
	}
	overLimit := roles.Username(strings.Repeat("a", roles.MaxIdentifierBytes+1))
	if err := overLimit.Validate(); err == nil {
		t.Error("64-byte username should be invalid (Postgres would truncate it)")
	}
}
