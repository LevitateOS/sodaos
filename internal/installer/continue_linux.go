package installer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func addContinuation(destination, binary []byte) ([]byte, error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal(destination, &config); err != nil {
		return nil, err
	}
	var storage map[string]json.RawMessage
	if err := json.Unmarshal(config["storage"], &storage); err != nil || storage == nil {
		return nil, errors.New("invalid continuation storage")
	}
	var files []json.RawMessage
	if err := json.Unmarshal(storage["files"], &files); err != nil {
		return nil, err
	}
	// Use the real writable CoreOS path; /usr/local is its native symlink.
	for path, content := range map[string][]byte{
		"/var/usrlocal/libexec/soda/soda-install": binary,
		"/etc/profile.d/soda-install-next.sh":     []byte("# Guidance only; never install or reboot from a login hook.\nif [ \"$(id -u)\" = 0 ] && [ -t 1 ] && [ ! -e /etc/soda/installed ]; then\n  printf '%s\\n' 'Soda installation is incomplete. Run: sudo /usr/local/libexec/soda/soda-install continue'\nfi\n"),
	} {
		mode := 0644
		if path == "/var/usrlocal/libexec/soda/soda-install" {
			mode = 0755
		}
		for _, existing := range files {
			var f struct {
				Path string `json:"path"`
			}
			if json.Unmarshal(existing, &f) != nil || f.Path == path {
				return nil, errors.New("continuation path collision")
			}
		}
		file, _ := json.Marshal(map[string]interface{}{"path": path, "mode": mode, "contents": map[string]string{"source": "data:;base64," + base64.StdEncoding.EncodeToString(content)}})
		files = append(files, file)
	}
	storage["files"], _ = json.Marshal(files)
	config["storage"], _ = json.Marshal(storage)
	return json.Marshal(config)
}

func extensionsBooted(data []byte) error {
	var state struct {
		Deployments []struct {
			Booted bool `json:"booted"`
			Staged bool `json:"staged"`
		} `json:"deployments"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return errors.New("cannot inspect rpm-ostree deployment state")
	}
	count := 0
	for _, d := range state.Deployments {
		if d.Booted {
			count++
		}
	}
	if len(state.Deployments) == 0 || count != 1 {
		return errors.New("one booted deployment required")
	}
	if !state.Deployments[0].Booted || state.Deployments[0].Staged {
		return errors.New("a deployment is pending: review rpm-ostree status and explicitly reboot to activate extensions, then run continue again")
	}
	return nil
}

func freshAppliance() error {
	for _, path := range []string{"/etc/soda/installed", "/etc/soda/install-started", "/etc/soda/dashboard.json", "/etc/soda/host.json", "/var/lib/soda-installer/continue-started"} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return errors.New("existing or partial Soda installation requires an operator decision; no replay")
		}
	}
	return nil
}

// A root-owned, non-writable-by-others tree prevents an unprivileged writer from
// replacing a script between verification and execution. Native bundle verification
// additionally confines paths and validates the exact allowlist, ELF and OCI bytes.
func protectedBundle(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("absolute canonical bundle directory required")
	}
	for current := path; ; current = filepath.Dir(current) {
		st, err := os.Lstat(current)
		if err != nil {
			return err
		}
		sys, ok := st.Sys().(*syscall.Stat_t)
		if !ok || sys.Uid != 0 || !st.IsDir() || st.Mode().Perm()&0022 != 0 {
			return errors.New("bundle and ancestors must be real root-owned directories not writable by others")
		}
		if current == "/" {
			break
		}
	}
	return filepath.WalkDir(path, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		st, err := d.Info()
		if err != nil {
			return err
		}
		sys, ok := st.Sys().(*syscall.Stat_t)
		if !ok || sys.Uid != 0 {
			return errors.New("bundle entries must be root-owned")
		}
		if st.Mode()&os.ModeSymlink == 0 && st.Mode().Perm()&0022 != 0 {
			return errors.New("bundle entries must not be writable by others")
		}
		return nil
	})
}

func verifyTrustedBundle(bundle, digest, arch string) (nativebuild.Inventory, error) {
	var inventory nativebuild.Inventory
	if !nativebuild.Digest(digest) {
		return inventory, errors.New("independently trusted SHA256SUMS SHA-256 required")
	}
	if err := protectedBundle(bundle); err != nil {
		return inventory, err
	}
	actual, err := nativebuild.HashFile(filepath.Join(bundle, "SHA256SUMS"))
	if err != nil || actual != digest {
		return inventory, errors.New("bundle does not match the independently trusted checksum")
	}
	if err := nativebuild.ReadJSON(filepath.Join(bundle, "build-info.json"), &inventory); err != nil {
		return inventory, errors.New("cannot read bundle inventory")
	}
	if inventory.Architecture != arch || !nativebuild.Revision(inventory.Revision) {
		return inventory, errors.New("bundle architecture/revision mismatch")
	}
	return nativebuild.Verify(bundle, arch, inventory.Revision)
}

func continueInstall(ctx context.Context, c console, run commandRunner) error {
	if err := freshAppliance(); err != nil {
		return err
	}
	c.print("SodaOS installed-host continuation. No disk writes, automatic reboot, provider enrollment or public activation.")
	if _, err := os.Stat("/var/lib/soda/extensions-requested"); err != nil {
		return errors.New("extensions are not confirmed: inspect systemctl status soda-extensions and its journal; do not reinstall the host")
	}
	data, err := run(ctx, "rpm-ostree", []string{"status", "--json"}, nil)
	if err != nil {
		return err
	}
	if err := extensionsBooted(data); err != nil {
		return err
	}
	c.print("Transfer the matching sealed bundle through independently trusted operator SSH/SCP into a new root-owned directory (for example below /root). Keep its architecture directory name. Obtain the SHA-256 of SHA256SUMS from the trusted builder/channel, not from the received bundle itself.")
	bundle, err := c.ask("Absolute bundle directory")
	if err != nil {
		return err
	}
	digest, err := c.ask("Independently trusted SHA-256 of SHA256SUMS")
	if err != nil {
		return err
	}
	if filepath.Base(bundle) != architecture() {
		return errors.New("bundle directory must name the native architecture")
	}
	c.print("Verifying the bundle with this already trusted installer; no bundle program has been executed.")
	inventory, err := verifyTrustedBundle(bundle, digest, architecture())
	if err != nil {
		return errors.New("bundle verification refused; check trusted identity, ownership, architecture and matching source inventory")
	}
	data, err = readRegular("/etc/soda-installer/project-subnet", 128)
	if err != nil {
		return err
	}
	subnet := strings.TrimSpace(string(data))
	observed, err := routes(ctx, run)
	if err != nil {
		return err
	}
	if err := ProjectSubnet(subnet, observed); err != nil {
		return err
	}
	c.print("Install Soda revision %s for %s using project subnet %s. This installs files/images and starts private/loopback services; it does not complete Forgejo/OAuth/TLS setup.", inventory.Revision, architecture(), subnet)
	answer, err := c.ask("Type INSTALL SODA to proceed")
	if err != nil {
		return err
	}
	if answer != "INSTALL SODA" {
		return errors.New("Soda installation cancelled")
	}
	if err := freshAppliance(); err != nil {
		return err
	}
	// The native script also checks actual installed RPMs, routes, containers and
	// identities. This marker additionally prevents replay if it fails early.
	if err := os.Mkdir("/var/lib/soda-installer", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	st, err := os.Lstat("/var/lib/soda-installer")
	if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
		return errors.New("private continuation state directory required")
	}
	if err := nativebuild.WriteNew("/var/lib/soda-installer/continue-started", []byte(inventory.Revision+"\n"), 0600); err != nil {
		return err
	}
	if _, err := run(ctx, "bash", []string{filepath.Join(bundle, "install-native.sh"), bundle, subnet}, nil); err != nil {
		return errors.New("Soda first-install failed or was interrupted; preserve partial state and inspect with the operator. No retry or reboot performed. " + failureSummary(err))
	}
	c.print("Soda components installed. Complete native Forgejo setup over operator SSH forwarding to host loopback port 3000, then soda-setup and soda-activate with the real private origin/TLS inputs. Cockpit remains loopback-first; project client routing is still required.")
	c.print("Guide: https://github.com/LevitateOS/sodaos/blob/main/docs/operator-setup.md")
	return nil
}
