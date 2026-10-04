// Port of test_terminal_assets.py: locked terminal distribution wiring.
package build

import (
	"strings"
	"testing"
)

func packageScripts(t *testing.T) map[string]string {
	t.Helper()
	raw, ok := ReadJSON(t, "package.json").(map[string]any)
	Require(t, ok, "package.json is not an object")
	entries, ok := raw["scripts"].(map[string]any)
	Require(t, ok, "package.json scripts is not an object")
	scripts := make(map[string]string, len(entries))
	for name, command := range entries {
		text, ok := command.(string)
		Require(t, ok, "script %s is not a string", name)
		scripts[name] = text
	}
	return scripts
}

func TestTerminalAssetsBrowserBuildPreparesLockedRenderer(t *testing.T) {
	scripts := packageScripts(t)
	parts := strings.SplitN(scripts["build:forgejo"], " && ", 2)
	Require(t, len(parts) == 2, "build:forgejo has no prepare step")
	Check(t, parts[0] == "cargo run --release --locked -p soda-asset-fetchers --bin soda-fetch-terminal -- --out .artifacts/browser-terminal/vendor",
		"prepare = %q", parts[0])
	Check(t, parts[1] == "bun scripts/build-forgejo.ts", "emit = %q", parts[1])
	groups := []string{"frontend", "forgejo"}
	want := []string{"bun run build:forgejo", "bun run test:frontend:prepared", "bun run test:forgejo:prepared"}
	got := strings.Split(scripts["test"], " && ")
	Require(t, len(got) == len(want), "test script = %q", scripts["test"])
	for i := range want {
		Check(t, got[i] == want[i], "test step %d = %q", i, got[i])
	}
	for _, group := range groups {
		Check(t, scripts["test:"+group] == "bun run build:forgejo && bun run test:"+group+":prepared",
			"test:%s = %q", group, scripts["test:"+group])
		Check(t, !strings.Contains(scripts["test:"+group+":prepared"], "build:forgejo"),
			"test:%s:prepared rebuilds", group)
	}
	Check(t, scripts["test:frontend:prepared"] == "SODA_TAILNET_COMPONENT=1 bun test tests/frontend/*.test.ts",
		"frontend prepared = %q", scripts["test:frontend:prepared"])
	Check(t, scripts["test:forgejo:prepared"] == "SODA_LIT_BROWSER=1 bun test --timeout 120000 tests/forgejo/*.test.ts tests/forgejo/presentation/*.test.ts",
		"forgejo prepared = %q", scripts["test:forgejo:prepared"])
	Check(t, strings.HasPrefix(scripts["test:lit"], "bun run build:forgejo && SODA_LIT_BROWSER=1 bun test "),
		"test:lit = %q", scripts["test:lit"])
	for _, word := range strings.Fields(scripts["test:lit"]) {
		Check(t, word != "tests/forgejo/settings-link.test.ts", "test:lit covers settings-link")
	}
}

func TestTerminalAssetsShippingLockHasOnlyExactLocalRendererFiles(t *testing.T) {
	raw, ok := ReadJSON(t, "appliance/terminal-assets.lock.json").([]any)
	Require(t, ok, "terminal lock is not a list")
	type pair struct{ pkg, ver string }
	var pairs []pair
	files := map[string]bool{}
	for _, entry := range raw {
		item := entry.(map[string]any)
		pairs = append(pairs, pair{item["package"].(string), item["version"].(string)})
		Check(t, strings.HasPrefix(item["url"].(string), "https://registry.npmjs.org/"), "url = %v", item["url"])
		for _, f := range item["files"].([]any) {
			asset := f.(map[string]any)
			files[asset["file"].(string)] = true
			Check(t, len(asset["sha256"].(string)) == 64, "sha256 length for %v", asset["file"])
		}
	}
	wantPairs := []pair{{"@xterm/xterm", "6.0.0"}, {"@xterm/addon-fit", "0.11.0"}}
	Require(t, len(pairs) == len(wantPairs), "lock packages = %v", pairs)
	for i := range wantPairs {
		Check(t, pairs[i] == wantPairs[i], "lock package %d = %v", i, pairs[i])
	}
	wantFiles := map[string]bool{"xterm.mjs": true, "xterm.css": true, "addon-fit.mjs": true, "xterm.LICENSE": true, "fit.LICENSE": true}
	Check(t, len(files) == len(wantFiles), "lock files = %v", files)
	for name := range wantFiles {
		Check(t, files[name], "lock file %s missing", name)
	}
}
