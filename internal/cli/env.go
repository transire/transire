// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cli

import "github.com/transire/transire/internal/config"

type envSettings struct {
	profile string
	region  string
}

func resolveEnv(m config.Manifest, envFlag string, profileFlag string, regionFlag string) envSettings {
	profile := profileFlag
	region := regionFlag

	if envFlag != "" {
		if envCfg, ok := m.Environments[envFlag]; ok {
			if profile == "" && envCfg.Profile != "" {
				profile = envCfg.Profile
			}
		}
	}

	// Fallbacks
	if profile == "" {
		profile = "transire-sandbox"
	}
	return envSettings{profile: profile, region: region}
}
