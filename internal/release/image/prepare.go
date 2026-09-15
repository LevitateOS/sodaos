// Package image prepares a small, public-only host build context. It does not
// install, migrate, sign or publish an appliance release.
package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

type Base struct {
	Release     string
	MetadataURL string
	Images      map[string]string
}

func LoadBase(source, arch string) (Base, error) {
	var b Base
	if _, err := build.OCIArchitecture(arch); err != nil {
		return b, err
	}
	if err := build.ReadJSON(filepath.Join(source, "appliance/locks/coreos-host.json"), &b); err != nil {
		return b, err
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(b.Release) || b.MetadataURL != "https://builds.coreos.fedoraproject.org/prod/streams/stable/builds/"+b.Release+"/release.json" {
		return b, errors.New("invalid pinned CoreOS release")
	}
	if len(b.Images) != 2 {
		return b, errors.New("both architecture base digests required")
	}
	for _, a := range []string{"x86_64", "aarch64"} {
		const prefix = "quay.io/fedora/fedora-coreos@sha256:"
		if !strings.HasPrefix(b.Images[a], prefix) || !build.Digest(strings.TrimPrefix(b.Images[a], prefix)) {
			return b, errors.New("digest-pinned CoreOS base required")
		}
	}
	return b, nil
}

// PackageInputs uses the current first-install owner rather than maintaining a
// second host package list. Reject changes in its command shape for explicit review.
func admitTailscaleRepo(files []struct {
	Path     string
	Contents struct{ Source string }
},
) (string, error) {
	var repo string
	for _, file := range files {
		if file.Path != "/etc/yum.repos.d/tailscale.repo" {
			continue
		}
		if repo != "" || file.Contents.Source != "https://pkgs.tailscale.com/stable/fedora/tailscale.repo" {
			return "", errors.New("unexpected Tailscale repository")
		}
		repo = file.Contents.Source
	}
	return repo, nil
}

func parseInstallPackages(line string, existing []string, seen map[string]bool) ([]string, error) {
	const prefix = "ExecStart=/usr/bin/rpm-ostree install -y --allow-inactive "
	if existing != nil || !strings.HasPrefix(line, prefix) {
		return nil, errors.New("unexpected package installation command")
	}
	packages := strings.Fields(strings.TrimPrefix(line, prefix))
	for _, name := range packages {
		if !regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`).MatchString(name) || seen[name] {
			return nil, errors.New("invalid or duplicate host package")
		}
		seen[name] = true
	}
	return packages, nil
}

func packagesFromUnits(units []struct{ Name, Contents string }) ([]string, error) {
	var packages []string
	seen := map[string]bool{}
	for _, unit := range units {
		if unit.Name != "soda-extensions.service" {
			continue
		}
		for _, line := range strings.Split(unit.Contents, "\n") {
			if !strings.HasPrefix(line, "ExecStart=") {
				continue
			}
			parsed, err := parseInstallPackages(line, packages, seen)
			if err != nil {
				return nil, err
			}
			packages = parsed
		}
	}
	return packages, nil
}

func PackageInputs(data []byte) ([]string, string, error) {
	var p struct {
		Storage struct {
			Files []struct {
				Path     string
				Contents struct{ Source string }
			}
		}
		Systemd struct {
			Units []struct{ Name, Contents string }
		}
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, "", err
	}
	repo, err := admitTailscaleRepo(p.Storage.Files)
	if err != nil {
		return nil, "", err
	}
	packages, err := packagesFromUnits(p.Systemd.Units)
	if err != nil {
		return nil, "", err
	}
	if len(packages) == 0 || repo == "" {
		return nil, "", errors.New("missing host package inputs")
	}
	return packages, repo, nil
}

// Prepare consumes committed source, never a staged appliance, a /var tree or a
// secret-file directory. Go binaries are compiled separately into the returned
// context's usr/libexec/soda directory by the caller.
type preparedWriter struct {
	source string
	out    string
}

func (w preparedWriter) write(name string, data []byte, mode os.FileMode) error {
	dest := filepath.Join(w.out, name)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := build.WriteNew(dest, data, mode); err != nil {
		return err
	}
	return os.Chmod(dest, mode)
}

func (w preparedWriter) copyFile(from, to string, mode os.FileMode, vendor bool) error {
	path := filepath.Join(w.source, from)
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("regular public source required: %s", from)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if vendor {
		data = []byte(strings.ReplaceAll(string(data), "/usr/local/libexec/soda/", "/usr/libexec/soda/"))
	}
	return w.write(to, data, mode)
}

func (w preparedWriter) writeBaseFiles(packages []string, repo string) error {
	if err := w.copyFile("appliance/host.Containerfile", "Containerfile", 0o644, false); err != nil {
		return err
	}
	for name, text := range map[string]string{
		"packages.list":      strings.Join(packages, "\n") + "\n",
		"packages.expected":  "",
		"tailscale-repo.url": repo + "\n",
	} {
		if err := w.write(name, []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func rootfsFileMap() map[string]string {
	files := map[string]string{
		"appliance/config/soda.sysusers":         "usr/lib/sysusers.d/soda.conf",
		"appliance/host-image/packages.tmpfiles": "usr/lib/tmpfiles.d/soda-host-packages.conf",
		"appliance/config/runners.sysusers":      "usr/lib/sysusers.d/soda-runners.conf",
		"appliance/config/soda.tmpfiles":         "usr/lib/tmpfiles.d/soda.conf",
		"appliance/config/runners.tmpfiles":      "usr/lib/tmpfiles.d/soda-runners.conf",
		"appliance/config/90-soda-routing.conf":  "usr/lib/sysctl.d/90-soda-routing.conf",
		"appliance/config/cockpit.socket.conf":   "usr/lib/systemd/system/cockpit.socket.d/10-soda.conf",
		"appliance/config/cockpit.pam":           "etc/pam.d/cockpit",
		"appliance/config/cockpit.conf":          "etc/cockpit/cockpit.conf",
		"appliance/config/console-welcome.sh":    "etc/profile.d/soda-console-welcome.sh",
		"LICENSE":                                "usr/share/licenses/soda/LICENSE",
		"NOTICE":                                 "usr/share/licenses/soda/NOTICE",
	}
	for _, name := range []string{"soda-host.service", "soda-host.socket", "soda-project@.service", "soda-tailnet@.service", "soda-runner@.service"} {
		files["appliance/services/"+name] = "usr/lib/systemd/system/" + name
	}
	for _, name := range []string{"forgejo.container", "soda-dashboard.container", "soda-proxy.container"} {
		files["appliance/services/"+name] = "usr/share/containers/systemd/" + name
	}
	return files
}

func (w preparedWriter) stageSymlinksAndExtras() error {
	for name, target := range map[string]string{
		"usr/bin/soda-tailnet": "../libexec/soda/soda-tailnet",
		"usr/bin/soda-setup":   "../libexec/soda/soda-setup",
	} {
		dest := filepath.Join(w.out, "rootfs", name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(target, dest); err != nil {
			return err
		}
	}
	return w.write("rootfs/etc/cockpit/disallowed-users", []byte(""), 0o644)
}

func (w preparedWriter) stageRootfsFiles() error {
	for from, to := range rootfsFileMap() {
		if err := w.copyFile(from, "rootfs/"+to, 0o644, true); err != nil {
			return err
		}
	}
	for from, to := range map[string]string{
		"appliance/bin/soda-console-welcome": "usr/libexec/soda/soda-console-welcome",
		"appliance/bin/soda-activate":        "usr/bin/soda-activate",
	} {
		if err := w.copyFile(from, "rootfs/"+to, 0o755, true); err != nil {
			return err
		}
	}
	return w.stageSymlinksAndExtras()
}

func (w preparedWriter) writeBuildRecord(b Base, revision, arch string, packages []string) error {
	record := struct {
		Scope, Revision, Architecture, CoreOS, Base string
		Packages                                    []string
	}{
		"host-content-only; not an installable or signed appliance release",
		revision,
		arch,
		b.Release,
		b.Images[arch],
		packages,
	}
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := w.write("rootfs/usr/share/soda/host-image/build.json", append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	return filepath.WalkDir(filepath.Join(w.out, "rootfs"), func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return nil
	})
}

func loadBaseInputs(source, arch, revision string) (Base, []string, string, error) {
	b, err := LoadBase(source, arch)
	if err != nil {
		return b, nil, "", err
	}
	if !build.Revision(revision) {
		return b, nil, "", errors.New("exact source revision required")
	}
	data, err := os.ReadFile(filepath.Join(source, "appliance/provisioning/base.json"))
	if err != nil {
		return b, nil, "", err
	}
	packages, repo, err := PackageInputs(data)
	if err != nil {
		return b, nil, "", err
	}
	return b, packages, repo, nil
}

// Prepare consumes committed source, never a staged appliance, a /var tree or a
// secret-file directory. Go binaries are compiled separately into the returned
// context's usr/libexec/soda directory by the caller.
func Prepare(source, out, arch, revision string) (Base, error) {
	b, packages, repo, err := loadBaseInputs(source, arch, revision)
	if err != nil {
		return b, err
	}
	if err := build.FreshDirectory(out); err != nil {
		return b, err
	}
	w := preparedWriter{source: source, out: out}
	if err := w.writeBaseFiles(packages, repo); err != nil {
		return b, err
	}
	if err := w.stageRootfsFiles(); err != nil {
		return b, err
	}
	if err := w.writeBuildRecord(b, revision, arch, packages); err != nil {
		return b, err
	}
	return b, nil
}

// Inventory records the context's exact files/modes and links. This is integrity
// evidence, not signature verification or a native product acceptance receipt.
func Inventory(context string) error {
	entries := map[string]build.File{}
	err := filepath.WalkDir(context, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(context, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		f := build.File{Mode: uint32(info.Mode().Perm())}
		if info.Mode()&os.ModeSymlink != 0 {
			f.Link, err = os.Readlink(path)
		} else {
			f.SHA256, err = build.HashFile(path)
		}
		if err != nil {
			return err
		}
		entries[rel] = f
		return nil
	})
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(filepath.Dir(context), "context-inventory.json"), append(data, '\n'), 0o600)
}
