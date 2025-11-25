// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest defines the transire.yaml schema.
type Manifest struct {
	App struct {
		Name string `yaml:"name"`
	} `yaml:"app"`
	AWS struct {
		Region string `yaml:"region"`
	} `yaml:"aws"`
	Environments map[string]Environment `yaml:"envs"`
}

type Environment struct {
	Profile string `yaml:"profile"`
	Region  string `yaml:"region"`
}

// LoadManifest reads transire.yaml from the given path. If missing, it returns defaults.
func LoadManifest(path string) (Manifest, error) {
	var m Manifest
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			m.App.Name = "transire-app"
			return m, nil
		}
		return m, err
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if m.App.Name == "" {
		m.App.Name = "transire-app"
	}
	return m, nil
}

// ParseDuration converts a manifest rate string into a duration.
// Supports simple suffixes: s, m, h, d.
func ParseDuration(rate string) (time.Duration, error) {
	d, err := time.ParseDuration(rate)
	if err == nil {
		return d, nil
	}
	return 0, fmt.Errorf("invalid duration %q", rate)
}
