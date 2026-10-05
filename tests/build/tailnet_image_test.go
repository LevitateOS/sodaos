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
	build := ReadFile(t, "rust/soda-release-build/src/production.rs")
	Check(t, !strings.Contains(build, "locks/"), "production.rs references locks/")
}

func TestTailnetRecipeFloatsOnBuildArgs(t *testing.T) {
	recipe := ReadFile(t, "appliance/tailnet.Containerfile")
	Check(t, strings.Contains(recipe, "ADD --checksum=sha256:${ARCHIVE_SHA256}"), "missing checksum ADD")
	Check(t, strings.Contains(recipe, "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz"), "missing archive URL")
	Check(t, strings.Contains(recipe, `ENTRYPOINT ["/usr/local/bin/tailscaled"]`), "missing entrypoint")
	Check(t, strings.Contains(recipe, "COPY appliance/licenses/tailscale-LICENSE /usr/share/licenses/tailscale/LICENSE"), "missing license COPY")
	license := ReadFile(t, "appliance/licenses/tailscale-LICENSE")
	Check(t, strings.Contains(license, "Copyright (c) 2020 Tailscale Inc & contributors."), "license text changed")
}

func TestTailnetBuildWiresLiveInputsWithoutLock(t *testing.T) {
	build := strings.ReplaceAll(ReadFile(t, "rust/soda-release-build/src/production.rs"), " ", "")
	Check(t, strings.Contains(build, `"--build-arg=TAILSCALE_VERSION={}"`), "missing version arg")
	Check(t, strings.Contains(build, `"--build-arg=ARCHIVE_SHA256={}"`), "missing sha arg")
	Check(t, strings.Contains(build, `"appliance/tailnet.Containerfile"`), "missing Containerfile ref")
}

func TestTailnetObservedVersionsRecordedWithoutGate(t *testing.T) {
	build := ReadFile(t, "rust/soda-release-build/src/production.rs")
	Check(t, !strings.Contains(build, "require_tailnet_release"), "gate present")
	Check(t, strings.Contains(build, "live_tailnet_inputs"), "missing live_tailnet_inputs")
}
