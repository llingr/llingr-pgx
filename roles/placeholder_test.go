// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package roles_test

import (
	"testing"

	"github.com/llingr/llingr-pgx/roles"
)

func TestPlaceholder_String(t *testing.T) {
	if roles.OwnerRole.String() != "owner" {
		t.Fatalf("String() = %q, want owner", roles.OwnerRole.String())
	}
}

// Validate enforces the plain-SQL-identifier rule (the injection mitigation).
func TestPlaceholder_Validate(t *testing.T) {
	valid := []roles.Placeholder{
		roles.OwnerRole, roles.AppRole, "readonly", "_x", "a1", "Reporting_2",
	}
	for _, p := range valid {
		if err := p.Validate(); err != nil {
			t.Errorf("%q should be valid: %v", p, err)
		}
	}

	invalid := []roles.Placeholder{
		"",           // empty
		"1abc",       // leading digit
		"a-b",        // hyphen
		"a b",        // space
		"a;b",        // statement separator
		"drop table", // space + keyword
		"naïve",      // non-ASCII
	}
	for _, p := range invalid {
		if err := p.Validate(); err == nil {
			t.Errorf("%q should be invalid", p)
		}
	}
}
