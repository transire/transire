// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package local

import "testing"

func TestResolveAddrDefault(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("TRANSIRE_PORT", "")
	if got := resolveAddr(""); got != ":8080" {
		t.Fatalf("expected :8080, got %s", got)
	}
}

func TestResolveAddrUsesPortEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	if got := resolveAddr(""); got != ":9090" {
		t.Fatalf("expected :9090, got %s", got)
	}
}

func TestResolveAddrUsesTransirePortEnv(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("TRANSIRE_PORT", "7070")
	if got := resolveAddr(""); got != ":7070" {
		t.Fatalf("expected :7070, got %s", got)
	}
}

func TestResolveAddrPrefersExplicit(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("TRANSIRE_PORT", "7070")
	if got := resolveAddr(":1234"); got != ":1234" {
		t.Fatalf("expected explicit :1234, got %s", got)
	}
}
