package hostimage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

type PackageLock struct {
	CoreOS       string
	Architecture string
	Requested    []string
	Install      []string
	Inventory    []string
}

// LockHostPackages pins the complete added transaction and checks the entire
// resulting RPM inventory. Mirrors may disappear: fail, never resolve a newer
// substitute. This is NEVRA/provenance locking, not byte-reproducible RPM storage.
func LockHostPackages(source, context, arch string, base Base) (string, error) {
	var lock PackageLock
	if err := nativebuild.ReadJSON(filepath.Join(source, "appliance/locks/host-packages-"+arch+".json"), &lock); err != nil {
		return "", fmt.Errorf("qualified architecture package lock required: %w", err)
	}
	b, err := os.ReadFile(filepath.Join(source, "appliance/provisioning/base.json"))
	if err != nil {
		return "", err
	}
	requested, _, err := PackageInputs(b)
	if err != nil {
		return "", err
	}
	if lock.CoreOS != base.Release || lock.Architecture != arch || !slices.Equal(lock.Requested, requested) || len(lock.Install) == 0 || len(lock.Inventory) == 0 {
		return "", errors.New("host package lock does not match current base/provisioning")
	}
	seen := map[string]bool{}
	for _, n := range lock.Install {
		if !regexp.MustCompile(`^[a-z0-9][a-zA-Z0-9+._:-]*$`).MatchString(n) || seen[n] {
			return "", errors.New("invalid RPM lock entry")
		}
		seen[n] = true
	}
	if !slices.IsSorted(lock.Inventory) {
		return "", errors.New("sorted RPM inventory required")
	}
	seen = map[string]bool{}
	for _, line := range lock.Inventory {
		if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9+._-]* [0-9]+:[a-zA-Z0-9+._~^-]+$`).MatchString(line) || seen[line] {
			return "", errors.New("invalid expected RPM inventory")
		}
		seen[line] = true
	}
	expected := []byte(strings.Join(lock.Inventory, "\n") + "\n")
	if err = ownedWrite(filepath.Join(context, "packages.list"), []byte(strings.Join(lock.Install, "\n")+"\n"), 0644); err != nil {
		return "", err
	}
	if err = ownedWrite(filepath.Join(context, "packages.expected"), expected, 0644); err != nil {
		return "", err
	}
	return hashBytes(expected), nil
}

func ownedWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}
func hashBytes(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

// publicFiles verifies directly emitted assets without another layout copy.
// Public modes must not depend on the builder's umask.
func publicFiles(source string) (map[string]string, error) {
	files := map[string]string{}
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() {
		return nil, errors.New("real generated public directory required")
	}
	err = filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			if info.Mode() != os.ModeDir|0755 {
				return errors.New("public presentation directory must be 0755")
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return errors.New("public stage symlink/special file refused")
		}
		if info.Mode() != 0644 {
			return errors.New("public presentation file must be 0644")
		}
		files[filepath.ToSlash(rel)], err = nativebuild.HashFile(path)
		return err
	})
	return files, err
}

func StagePresentation(forgejoContext, context string) (string, error) {
	files, err := publicFiles(filepath.Join(forgejoContext, "forgejo"))
	if err != nil {
		return "", err
	}
	if len(files) == 0 || files["templates/custom/header.tmpl"] == "" {
		return "", errors.New("complete Forgejo presentation required")
	}
	data, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err = ownedWrite(filepath.Join(forgejoContext, "presentation.json"), data, 0644); err != nil {
		return "", err
	}
	if err = ownedWrite(filepath.Join(context, "rootfs/usr/share/soda/host-image/presentation.json"), data, 0644); err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

// Complete binds all callers to one v3 payload and ordinary Podman storage.
func Complete(source, context, archives string, p appliancerelease.Payload, run nativebuild.BuildExec) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Format != 3 {
		return errors.New("new candidates require the shared-layout v3 payload")
	}
	root := filepath.Join(context, "rootfs")
	for _, name := range []string{"forgejo", "dashboard", "proxy"} {
		unit := map[string]string{"forgejo": "forgejo.container", "dashboard": "soda-dashboard.container", "proxy": "soda-proxy.container"}[name]
		original, err := os.ReadFile(filepath.Join(source, "appliance/services", unit))
		if err != nil {
			return err
		}
		body, err := LocalQuadlet(string(original), p.Images[name].Config)
		if err != nil {
			return err
		}
		if err = ownedWrite(filepath.Join(root, "usr/share/containers/systemd", unit), []byte(body), 0644); err != nil {
			return err
		}
	}
	// Retain public-tree verification at the final destinations, without copying.
	for _, dir := range []string{"etc/cockpit/branding", "usr/share/soda/fastfetch", "etc/fastfetch"} {
		if _, err := publicFiles(filepath.Join(root, dir)); err != nil {
			return err
		}
	}
	motd, err := os.Lstat(filepath.Join(root, "etc/motd"))
	if err != nil || motd.Mode() != 0644 {
		return errors.New("regular public MOTD required")
	}
	// Keep the public factory defaults alongside /etc, not another stage.
	for _, name := range []string{"forgejo.env", "proxy.Caddyfile"} {
		b, err := os.ReadFile(filepath.Join(root, "etc/soda", name))
		if err != nil {
			return err
		}
		if err = ownedWrite(filepath.Join(root, "usr/share/soda/defaults", name), b, 0644); err != nil {
			return err
		}
	}
	// Machine setup must supply the explicit private subnet. Images are not saved
	// here: the vendor helper reads their immutable IDs from the release owner.
	example := []byte("{\n  \"network\": \"soda-projects\",\n  \"bridge\": \"soda0\",\n  \"subnet\": \"\",\n  \"tailnet_management\": false\n}\n")
	if err := ownedWrite(filepath.Join(root, "usr/share/soda/defaults/host.example.json"), example, 0644); err != nil {
		return err
	}
	for from, to := range map[string]string{
		"appliance/host-image/soda-image-import.service": "usr/lib/systemd/system/soda-image-import.service",
		"appliance/host-image/retained-images.conf":      "usr/lib/systemd/system/soda-host.service.d/10-images.conf",
	} {
		b, err := os.ReadFile(filepath.Join(source, from))
		if err != nil {
			return err
		}
		if err = ownedWrite(filepath.Join(root, to), b, 0644); err != nil {
			return err
		}
	}
	dropin, err := os.ReadFile(filepath.Join(source, "appliance/host-image/retained-images.conf"))
	if err != nil {
		return err
	}
	// Keep the project unit's exact-fragment/drop-in admission contract unchanged:
	// put this dependency directly in its generated vendor fragment, not a drop-in.
	project := filepath.Join(root, "usr/lib/systemd/system/soda-project@.service")
	body, err := os.ReadFile(project)
	if err != nil {
		return err
	}
	body = []byte(strings.Replace(string(body), "[Unit]\n", string(dropin)+"\n", 1))
	if err = ownedWrite(project, body, 0644); err != nil {
		return err
	}
	if err = stageImages(archives, filepath.Join(root, "usr/share/soda/images"), p, run); err != nil {
		return err
	}
	// The complete payload is the sole resolved-image/defaults owner. Leave only
	// a pointer in the earlier host-content build marker, not stale scope data.
	if err := ownedWrite(filepath.Join(root, "usr/share/soda/host-image/build.json"), []byte("{\"Scope\":\"complete-local-payload\",\"ReleaseMetadata\":\"/usr/share/soda/release.json\"}\n"), 0644); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err = ownedWrite(filepath.Join(root, "usr/share/soda/release.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	// Normalize only fresh image context directories, not credentials or sources.
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0755)
		}
		return nil
	})
}

func LocalQuadlet(body, reference string) (string, error) {
	lines := strings.Split(body, "\n")
	image, container, unit := 0, 0, 0
	var out []string
	for _, line := range lines {
		switch {
		case line == "[Unit]":
			unit++
			out = append(out, line, "Requires=soda-image-import.service", "After=soda-image-import.service")
		case line == "[Container]":
			container++
			out = append(out, line, "Pull=never")
		case strings.HasPrefix(line, "Image="):
			image++
			out = append(out, "Image="+reference)
		case strings.HasPrefix(line, "Pull="):
		case strings.HasPrefix(line, "GlobalArgs="):
			return "", errors.New("unexpected existing Quadlet storage arguments")
		default:
			out = append(out, line)
		}
	}
	if image != 1 || container != 1 || unit != 1 {
		return "", errors.New("one fixed Quadlet container/image required")
	}
	return strings.Join(out, "\n"), nil
}
