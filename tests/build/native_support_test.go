// Port of test_native_support.py: provisioning renderer and outside contracts.
package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func renderBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-stage-render", "soda-render-provisioning", "--bin", "soda-render-provisioning")
}

type provisionFixture struct {
	root     string
	public   string
	password string
}

func newProvisionFixture(t *testing.T) provisionFixture {
	t.Helper()
	root := TempDir(t)
	Require(t, os.Chmod(root, 0o700) == nil, "chmod root")
	public := filepath.Join(root, "operator.pub")
	WriteFile(t, public, []byte("ssh-ed25519 AAAA synthetic-public-fixture\n"), 0o644)
	password := filepath.Join(root, "password.hash")
	WriteFile(t, password, []byte("$6$synthetic$not-a-real-password-hash\n"), 0o600)
	return provisionFixture{root: root, public: public, password: password}
}

func (f provisionFixture) renderTo(t *testing.T, dest string, env []string, extra ...string) ProcResult {
	t.Helper()
	args := []string{"--operator-key-file", f.public, "--root-password-hash-file", f.password}
	args = append(args, extra...)
	args = append(args, "--out", dest)
	return Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 60 * time.Second}, renderBinary(t), args...)
}

func readButane(t *testing.T, dest string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(dest)
	Require(t, err == nil, "read render output: %v", err)
	var decoded map[string]any
	Require(t, json.Unmarshal(data, &decoded) == nil, "parse render output")
	return decoded
}

func butaneFile(t *testing.T, data map[string]any, path string) map[string]any {
	t.Helper()
	storage, ok := data["storage"].(map[string]any)
	Require(t, ok, "no storage section")
	for _, entry := range storage["files"].([]any) {
		file := entry.(map[string]any)
		if file["path"] == path {
			return file
		}
	}
	t.Fatalf("file %s not rendered", path)
	return nil
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	Require(t, err == nil, "stat %s: %v", path, err)
	return info.Mode().Perm()
}

func TestProvisioningMinimalAndExtensionProfilesAreDistinct(t *testing.T) {
	f := newProvisionFixture(t)
	for _, profile := range []string{"minimal", "extensions"} {
		t.Run(profile, func(t *testing.T) {
			dest := filepath.Join(f.root, profile+".bu")
			first := f.renderTo(t, dest, nil, "--hostname", "soda-native-fixture", "--bootstrap", profile)
			Require(t, first.Code == 0, "render failed: %s", tail(first.Stderr, 2000))
			data := readButane(t, dest)
			users := data["passwd"].(map[string]any)["users"].([]any)
			Require(t, len(users) == 1, "users = %v", users)
			Check(t, users[0].(map[string]any)["name"] == "root", "user = %v", users[0])
			_, hasSystemd := data["systemd"]
			Check(t, hasSystemd == (profile == "extensions"), "systemd present = %v", hasSystemd)
			Check(t, fileMode(t, dest) == 0o600, "mode = %o", fileMode(t, dest))
			hostfile := butaneFile(t, data, "/etc/hostname")
			Check(t, hostfile["contents"].(map[string]any)["inline"] == "soda-native-fixture\n", "hostname wrong")
			before, err := os.ReadFile(dest)
			Require(t, err == nil, "read output: %v", err)
			rerun := f.renderTo(t, dest, nil)
			Require(t, rerun.Code == 1, "rerun exit = %d: %s", rerun.Code, tail(rerun.Stderr, 2000))
			Check(t, rerun.Stderr == "Private provisioning failed (FileExistsError); check paths/modes/key format.\n",
				"stderr=%q", rerun.Stderr)
			after, err := os.ReadFile(dest)
			Require(t, err == nil, "reread output: %v", err)
			Check(t, string(after) == string(before), "rerun rewrote output")
		})
	}
}

func TestProvisioningPrivateModesAndPlaintextAreRejectedBeforeOutput(t *testing.T) {
	f := newProvisionFixture(t)
	dest := filepath.Join(f.root, "refused.bu")
	Require(t, os.Chmod(f.password, 0o644) == nil, "chmod password")
	refused := f.renderTo(t, dest, nil)
	Require(t, refused.Code == 1, "exit = %d: %s", refused.Code, tail(refused.Stderr, 2000))
	Check(t, refused.Stderr == "Private provisioning failed (ValueError); check paths/modes/key format.\n",
		"stderr=%q", refused.Stderr)
	_, err := os.Stat(dest)
	Check(t, os.IsNotExist(err), "output created despite refusal")
	Require(t, os.Chmod(f.password, 0o600) == nil, "chmod password")
	WriteFile(t, f.password, []byte("synthetic plaintext\n"), 0o600)
	refused = f.renderTo(t, dest, nil)
	Require(t, refused.Code == 1, "exit = %d: %s", refused.Code, tail(refused.Stderr, 2000))
	_, err = os.Stat(dest)
	Check(t, os.IsNotExist(err), "output created despite plaintext refusal")
}

func TestProvisioningHostKeyContentsNeverBecomeProcessArguments(t *testing.T) {
	f := newProvisionFixture(t)
	key := filepath.Join(f.root, "host-key")
	WriteFile(t, key, []byte("synthetic-private-host-key\n"), 0o600)
	helper := filepath.Join(f.root, "fakebin")
	Require(t, os.Mkdir(helper, 0o755) == nil, "mkdir fakebin")
	log := filepath.Join(f.root, "argv.log")
	keygen := "#!/bin/sh\nprintf \"%s\\n\" \"$@\" > " + log + "\nprintf \"ssh-ed25519 AAAA fixture\\n\"\n"
	WriteFile(t, filepath.Join(helper, "ssh-keygen"), []byte(keygen), 0o755)
	dest := filepath.Join(f.root, "instance.bu")
	env := SetEnv(os.Environ(), "PATH", helper+string(os.PathListSeparator)+os.Getenv("PATH"))
	done := f.renderTo(t, dest, env, "--hostname", "soda-native-fixture", "--ssh-host-key-file", key)
	Require(t, done.Code == 0, "render failed: %s", tail(done.Stderr, 2000))
	arguments, err := os.ReadFile(log)
	Require(t, err == nil, "read argv log: %v", err)
	keyText, err := os.ReadFile(key)
	Require(t, err == nil, "read key: %v", err)
	Check(t, !strings.Contains(string(arguments), string(keyText)), "key contents in argv")
	Check(t, strings.Contains(string(arguments), key), "key path missing from argv")
	private := butaneFile(t, readButane(t, dest), "/etc/ssh/ssh_host_ed25519_key")
	Check(t, private["mode"] == float64(0o600), "mode = %v", private["mode"])
	Check(t, private["contents"].(map[string]any)["inline"] == string(keyText), "key not embedded")
}

func TestProvisioningPublicBootstrapHasNoIdentityOrAutomaticReboot(t *testing.T) {
	raw := ReadFile(t, "appliance/provisioning/base.json")
	var data map[string]any
	Require(t, json.Unmarshal([]byte(raw), &data) == nil, "parse base.json")
	_, present := data["passwd"]
	Check(t, !present, "passwd present")
	Check(t, !strings.Contains(raw, "password_hash"), "password_hash present")
	Check(t, !strings.Contains(raw, "reboot"), "reboot present")
	Check(t, strings.Contains(raw, "rpm-ostree install"), "missing install")
}

func TestProvisioningInstalledConsoleWelcomeRunsBeforeLogin(t *testing.T) {
	raw := ReadJSON(t, "appliance/provisioning/candidate.json").(map[string]any)
	units := raw["systemd"].(map[string]any)["units"].([]any)
	found := false
	for _, entry := range units {
		unit := entry.(map[string]any)
		if unit["name"] == "soda-console.service" {
			found = unit["enabled"] == true
		}
	}
	Check(t, found, "console service not enabled")
	body := ReadFile(t, "appliance/services/soda-console.service")
	Check(t, strings.Contains(body, "Before=getty@tty1.service"), "missing Before")
	Check(t, strings.Contains(body, "50-soda.issue"), "missing issue")
}

func TestProvisioningInstalledSystemDefaultsToPasswordSSH(t *testing.T) {
	raw := ReadFile(t, "appliance/provisioning/candidate.json")
	var data map[string]any
	Require(t, json.Unmarshal([]byte(raw), &data) == nil, "parse candidate.json")
	files := data["storage"].(map[string]any)["files"].([]any)
	var dropin map[string]any
	for _, entry := range files {
		file := entry.(map[string]any)
		if strings.HasPrefix(file["path"].(string), "/etc/ssh/sshd_config.d/") {
			dropin = file
			break
		}
	}
	Require(t, dropin != nil, "no sshd dropin")
	Check(t, !strings.Contains(dropin["path"].(string), "key-only"), "key-only dropin")
	config := dropin["contents"].(map[string]any)["inline"].(string)
	Check(t, strings.Contains(config, "PermitRootLogin yes"), "missing root login")
	Check(t, strings.Contains(config, "PasswordAuthentication yes"), "missing password auth")
	Check(t, !strings.Contains(config, "prohibit-password"), "prohibit-password present")
	Check(t, !strings.Contains(config, " no\n"), "denial present")
}

func TestOutsideSupportToolsAreNotApplianceCommands(t *testing.T) {
	statIsFile := func(rel string) bool {
		info, err := os.Stat(filepath.Join(RepoRoot, rel))
		return err == nil && !info.IsDir()
	}
	statExists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(RepoRoot, rel))
		return err == nil
	}
	Check(t, statIsFile("rust/soda-release-tools/src/artifacts.rs"), "soda-artifacts owner missing")
	Check(t, !statExists("cmd/soda-artifacts"), "cmd soda-artifacts present")
	Check(t, statIsFile("rust/soda-acceptance/Cargo.toml"), "acceptance crate missing")
	Check(t, !statExists("cmd/soda-acceptance"), "cmd soda-acceptance present")
	Check(t, !statExists("tools/soda-acceptance/main.go"), "go acceptance driver present")
	Check(t, statIsFile("rust/soda-release-tools/src/build_cli.rs"), "soda-build missing")
	Check(t, !statExists("scripts/build-native.sh"), "build-native.sh present")
	Check(t, !statExists("tools/soda-host-image"), "soda-host-image present")
	producer := ReadFile(t, "rust/soda-release-build/src/production.rs")
	Check(t, strings.Contains(producer, `"save".to_string()`), "missing save")
	Check(t, strings.Contains(producer, `"--format=oci-archive".to_string()`), "missing oci-archive")
	Check(t, strings.Contains(producer, "--iidfile"), "missing iidfile")
	Check(t, !strings.Contains(producer, `"push"`), "push present")
}

func TestOutsideBootBindsBrowserServicesWithoutQuadletEnable(t *testing.T) {
	unit := ReadFile(t, "appliance/services/soda-console.service")
	Check(t, strings.Contains(unit, "Wants=forgejo.service soda-dashboard.service soda-proxy.service"), "missing Wants")
	Check(t, !strings.Contains(unit, "enable --now"), "enable present")
	for _, name := range []string{"soda-dashboard.container", "soda-proxy.container"} {
		container := ReadFile(t, "appliance/services/"+name)
		Check(t, strings.Contains(container, "ConditionPathExists=/etc/soda/activated"), "%s missing condition", name)
	}
}
