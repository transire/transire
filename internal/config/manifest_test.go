// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadManifestDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transire.yaml")
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.App.Name == "" {
		t.Fatalf("expected default app name")
	}
}

func TestLoadManifestParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transire.yaml")
	if err := os.WriteFile(path, []byte("app:\n  name: demo\naws:\n  region: us-east-1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.App.Name != "demo" || m.AWS.Region != "us-east-1" {
		t.Fatalf("manifest not parsed correctly")
	}
}

func TestParseDuration(t *testing.T) {
	d, err := ParseDuration("2m")
	if err != nil || d != 2*time.Minute {
		t.Fatalf("expected 2m, got %v err %v", d, err)
	}
	if _, err := ParseDuration("bad"); err == nil {
		t.Fatalf("expected error for bad duration")
	}
}
