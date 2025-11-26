// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import (
	"testing"

	"github.com/transire/transire/internal/config"
)

func TestResolveEnvUsesManifestEnv(t *testing.T) {
	m := config.Manifest{
		Environments: map[string]config.Environment{
			"dev": {Profile: "dev-prof"},
		},
	}
	res := resolveEnv(m, "dev", "", "")
	if res.profile != "dev-prof" || res.region != "" {
		t.Fatalf("unexpected env settings: %+v", res)
	}
}

func TestResolveEnvFlagOverrides(t *testing.T) {
	m := config.Manifest{
		Environments: map[string]config.Environment{
			"dev": {Profile: "dev-prof"},
		},
	}
	res := resolveEnv(m, "dev", "override-prof", "ap-southeast-1")
	if res.profile != "override-prof" || res.region != "ap-southeast-1" {
		t.Fatalf("overrides not applied: %+v", res)
	}
}

func TestResolveEnvDefaults(t *testing.T) {
	m := config.Manifest{}
	res := resolveEnv(m, "", "", "")
	if res.profile != "transire-sandbox" || res.region != "" {
		t.Fatalf("defaults not applied: %+v", res)
	}
}
