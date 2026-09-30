//go:build linux

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeMuseMountsAreExplicit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.json")
	if err := writeOverride(path, "development", "/run/soda-muse/nested/registration"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Services map[string]struct {
			Volumes []string `json:"volumes"`
		} `json:"services"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/run/soda-muse/credentials", "/usr/local/bin/muse", "/run/soda-muse-interface"} {
		found := false
		for _, mount := range wire.Services["development"].Volumes {
			if containsDestination(mount, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s from %q", want, wire.Services["development"].Volumes)
		}
	}
}

func containsDestination(mount, destination string) bool {
	_, rest, ok := strings.Cut(mount, ":")
	if !ok {
		return false
	}
	target, _, ok := strings.Cut(rest, ":")
	return ok && target == destination
}
