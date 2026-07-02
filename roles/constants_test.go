package roles_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/llingr/llingr-pgx/roles"
)

// The values of these constants are part of the package's contract: the role
// placeholders map to provisioned database users and to the :"name" variables
// substituted into migrations, and PlainSQLIdentifierRegex is a security control
// against placeholder injection.
//
// They are pinned by SHA-256 rather than by literal string assertion: the opaque
// digest is decoupled from the literal, so an accidental edits will be caught.
func TestConstantValuesPinnedBySHA256(t *testing.T) {
	cases := []struct {
		value    string
		wantHash string
	}{
		{
			roles.AdminOwnerRole.String(),
			"198486ad584159349b855a001d2436ed9f291badd68ae4b1878db891ed161ce2",
		},
		{
			roles.AppRole.String(),
			"a172cedcae47474b615c54d510a5d84a8dea3032e958587430b413538be3f333",
		},
		{
			roles.PlainSQLIdentifierRegex,
			"3eb027afea02cf1db879b04724c42ad3093cb3969ec188e8a5ad8c7c7453cae3",
		},
	}

	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			sum := sha256.Sum256([]byte(c.value))
			got := hex.EncodeToString(sum[:])
			if got != c.wantHash {
				t.Errorf("%s value changed (now %q).\n  got  sha256 = %s\n  want sha256 = %s\n"+
					"If this change is intentional, update the pinned hash.",
					c.value, c.value, got, c.wantHash)
			}
		})
	}
}
