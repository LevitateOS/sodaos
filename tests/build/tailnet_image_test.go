// Port of test_tailnet_image.py: tailnet float checks.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailnetNoStoredPins(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "appliance/locks"))
	Check(t, os.IsNotExist(err), "appliance/locks exists: %v", err)
	build := ReadFile(t, "lib/soda-release-build/src/production.rs")
	Check(t, !strings.Contains(build, "locks/"), "production.rs references locks/")
}

// Tailnet ships as the one pinned official image: no custom Containerfile,
// no floating version/sha build args, no live toolchain resolution. The
// license attribution stays pinned beside the image identity.
func TestTailnetRecipeFloatsOnBuildArgs(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "system/containers/tailnet/Containerfile"))
	Check(t, os.IsNotExist(err), "custom tailnet Containerfile present: %v", err)
	pinned := ReadFile(t, "lib/soda-release-build/src/production_inputs.rs")
	Check(t, strings.Contains(pinned, "TAILSCALE_OFFICIAL_IMAGE"), "missing official image pin")
	Check(t, strings.Contains(pinned, "docker.io/tailscale/tailscale@sha256:"), "official image pin is not digest-bound")
	license := ReadFile(t, "system/licenses/tailscale-LICENSE")
	Check(t, strings.Contains(license, "Copyright (c) 2020 Tailscale Inc & contributors."), "license text changed")
}

func TestTailnetBuildWiresLiveInputsWithoutLock(t *testing.T) {
	build := strings.ReplaceAll(ReadFile(t, "lib/soda-release-build/src/production_images.rs"), " ", "")
	Check(t, !strings.Contains(build, "TAILSCALE_VERSION"), "floating tailscale version arg")
	Check(t, !strings.Contains(build, "ARCHIVE_SHA256"), "floating archive sha arg")
	Check(t, !strings.Contains(build, `"system/containers/tailnet/Containerfile"`), "custom Containerfile ref")
	inputs := ReadFile(t, "lib/soda-release-build/src/production_inputs.rs")
	Check(t, strings.Contains(inputs, `"tailscale" => TAILSCALE_OFFICIAL_IMAGE.to_string()`), "tailscale role is not the official image")
}

func TestTailnetObservedVersionsRecordedWithoutGate(t *testing.T) {
	build := ReadFile(t, "lib/soda-release-build/src/production_inputs.rs")
	Check(t, !strings.Contains(build, "live_tailnet_inputs"), "floating tailnet toolchain resolution")
	Check(t, strings.Contains(build, "TAILSCALE_OFFICIAL_IMAGE"), "missing official image pin")
}
