// SPDX-FileCopyrightText: Copyright (c) 2026 The llingr-pgx Authors
// SPDX-License-Identifier: Apache-2.0

package connect

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// Constants pinned by SHA-256 digest than by literal
// assertion to catch inadvertent find/replace changes.
func TestSSLModeAndChannelBindingValuesPinnedBySHA256(t *testing.T) {
	cases := []struct {
		value    string
		wantHash string
	}{
		{SSLModeDisable, "e9d8992f348162fd95acf6d07922aff61ebd06a143eaf134f29d72e76cb420ce"},
		{SSLModeAllow, "410083735735a10e658a19edd1704e606c9dd112e225825b63fafeded766c8b9"},
		{SSLModePrefer, "472dc7749c49f123491f23fbaaaba25d45ff7b5b783648d40c741f66f1ccaa47"},
		{SSLModeRequire, "c4d0cf241a1bfa1c8bf4cf24e8f89d2ab786a284a39adb2fc8df7ea14e73c154"},
		{SSLModeVerifyCA, "1a3ed82a1103bae3e45cd9618363c370d315fb05f1466b58ff42e6719e38ff3e"},
		{SSLModeVerifyFull, "9ab363401920dd4f4677f19ff8e1c830c5bd79b4fd28b486598650f1d5343fea"},
		{ChannelBindingPrefer, "472dc7749c49f123491f23fbaaaba25d45ff7b5b783648d40c741f66f1ccaa47"},
		{ChannelBindingDisable, "e9d8992f348162fd95acf6d07922aff61ebd06a143eaf134f29d72e76cb420ce"},
		{ChannelBindingRequire, "c4d0cf241a1bfa1c8bf4cf24e8f89d2ab786a284a39adb2fc8df7ea14e73c154"},
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

// Constants pinned by SHA-256 digest than by literal
// assertion to catch inadvertent find/replace changes.
func TestConnectionParamKeywordsPinnedBySHA256(t *testing.T) {
	cases := []struct {
		value    string
		wantHash string
	}{
		{ParamHost, "4740ae6347b0172c01254ff55bae5aff5199f4446e7f6d643d40185b3f475145"},
		{ParamPort, "f8d397a33fcb9725db96501e653bf3cfa4455c5639482b9936c22b221634d659"},
		{ParamUser, "04f8996da763b7a969b1028ee3007569eaf3a635486ddab211d512c85b9df8fb"},
		{ParamPassword, "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"},
		{ParamDatabase, "454b6db96bed1d85d4ae0b6e6bee9a0e79c8fde45b1bd6cad6a11349f9c3c8c5"},
		{ParamSSLMode, "58ce0f1d1c7ea08fa77d6bb3f463bd8f29e9eabd674d62080c36f18f9bf8ecf1"},
		{ParamChannelBinding, "5c3361b4f3349dd18883ffb1a46c361debc0a76b2219529323658b98adaf19b6"},
		{ParamPoolMaxConns, "b259ce0682db81bd73b086b11b90519275fa5bdbf7ec26e8089a96968d29d721"},
		{ParamPoolMinConns, "ac728474b552f948eda4e2730fe65ec8ebf620a5230e01258c9f7ad1352aaea4"},
		{ParamPoolMaxConnLifetime, "f0ce6384f5a51ccad97fd63e1e7df7b77e96afc0934a7748d4b82b88f657dc98"},
		{ParamPoolMaxConnIdleTime, "33cdbee953ac7ababf094b9ae90fad869d10951bd1cacfbfefef88e03d1363c7"},
		{ParamPoolHealthCheckPeriod, "6c0275a79afc6b9d71411bcece405a90818c79a6b8f5fed9758b46888a2a9e2f"},
		{SchemePostgres, "a942b37ccfaf5a813b1432caa209a43b9d144e47ad0de1549c289c253e556cd5"},
		{ParamPoolMinIdleConns, "0fcb38abb632c508d148b4a1ee3da6928d747e59a4569693a317f9c48e167576"},
		{ParamPoolMaxConnLifetimeJitter, "0d2da1d5213125ddff350e6054db01abd9b40a80e82b4f9065d8f44bb3a0fd29"},
		{ParamApplicationName, "7e77de90098b13df10c0f294f76102c814eacc7f00f5976912614790cc88f4b3"},
		{ParamConnectTimeout, "a8d0b036f3fdcca54874857617bd771f91ce664af4a2588fd69af141f6718c23"},
		{ParamTargetSessionAttrs, "4a8c11b8e4de3ee49046a773fd5232ee9509959904f33ea0d10138535727f926"},
		{ParamSSLRootCert, "91b9db4b4112dbcde6d84fcef02e3607007d0e93e995dcd39720eceeb687fe4e"},
		{ParamSSLCert, "6b37494ab4591e1d0e99731537488f7a50dbb3c00db805b8db5ad1111e690d51"},
		{ParamSSLKey, "de123c96387b8603f53d2520ffb92541c38744b73d189124bf7227747ea721ce"},
		{ParamDefaultQueryExecMode, "2b3ca77c0eaf8e6f0db52a023a712d1479ddb8714f0a674d4dcc0e0064a65e8c"},
		{ParamStatementCacheCapacity, "3a34c6985b48cb9d9896a49bd8c6add9afe45c0a917e58f155d56bbc3acd07b0"},
		{ParamDescriptionCacheCapacity, "a87a05553a452b9ab51a3a17f20e618ce992f82be277daea237e952805ce08b4"},
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
