//go:build linux

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeProviderMountsAreExplicit(t *testing.T) {
	for _, test := range []struct {
		name      string
		muse, git bool
		want      []string
		absent    []string
	}{
		{"git", false, true, []string{"/run/soda-git/credentials", "/usr/local/bin/git-remote-soda", "/run/soda-git-interface/launch.sock"}, []string{"/run/soda-muse/credentials", "/usr/local/bin/muse", "/run/soda-muse-interface", "/run/soda-git-interface"}},
		{"muse", true, false, []string{"/run/soda-muse/credentials", "/usr/local/bin/muse", "/run/soda-muse-interface"}, []string{"/run/soda-git/credentials", "/usr/local/bin/git-remote-soda", "/run/soda-git-interface"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "compose.json")
			if err := writeOverride(path, "development", "/run/soda-muse/nested/registration", test.muse, test.git); err != nil {
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
			destinations := wire.Services["development"].Volumes
			for _, want := range test.want {
				found := false
				for _, mount := range destinations {
					if containsDestination(mount, want) {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing %s from %q", want, destinations)
				}
			}
			for _, absent := range test.absent {
				for _, mount := range destinations {
					if containsDestination(mount, absent) {
						t.Fatalf("unselected %s mounted: %q", absent, mount)
					}
				}
			}
		})
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
