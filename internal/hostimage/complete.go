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
func validRPMNames(names []string) error {
	seen := map[string]bool{}
	for _, n := range names {
		if !regexp.MustCompile(`^[a-z0-9][a-zA-Z0-9+._:-]*$`).MatchString(n) || seen[n] {
			return errors.New("invalid RPM lock entry")
		}
		seen[n] = true
	}
	return nil
}

func validRPMInventory(lines []string) error {
	if !slices.IsSorted(lines) {
		return errors.New("sorted RPM inventory required")
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9+._-]* [0-9]+:[a-zA-Z0-9+._~^-]+$`).MatchString(line) || seen[line] {
			return errors.New("invalid expected RPM inventory")
		}
		seen[line] = true
	}
	return nil
}

func lockMatchesBase(lock PackageLock, arch string, base Base, requested []string) bool {
	return lock.CoreOS == base.Release && lock.Architecture == arch && slices.Equal(lock.Requested, requested) && len(lock.Install) != 0 && len(lock.Inventory) != 0
}

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
	if !lockMatchesBase(lock, arch, base, requested) {
		return "", errors.New("host package lock does not match current base/provisioning")
	}
	if err = validRPMNames(lock.Install); err != nil {
		return "", err
	}
	if err = validRPMInventory(lock.Inventory); err != nil {
		return "", err
	}
	expected := []byte(strings.Join(lock.Inventory, "\n") + "\n")
	if err = ownedWrite(filepath.Join(context, "packages.list"), []byte(strings.Join(lock.Install, "\n")+"\n"), 0o644); err != nil {
		return "", err
	}
	if err = ownedWrite(filepath.Join(context, "packages.expected"), expected, 0o644); err != nil {
		return "", err
	}
	return hashBytes(expected), nil
}

func ownedWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
type publicFileSet struct {
	root  string
	files map[string]string
}

func (s *publicFileSet) visit(path string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(s.root, path)
	if err != nil {
		return err
	}
	info, err := d.Info()
	if err != nil {
		return err
	}
	if d.IsDir() {
		if info.Mode() != os.ModeDir|0o755 {
			return errors.New("public presentation directory must be 0755")
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return errors.New("public stage symlink/special file refused")
	}
	if info.Mode() != 0o644 {
		return errors.New("public presentation file must be 0644")
	}
	s.files[filepath.ToSlash(rel)], err = nativebuild.HashFile(path)
	return err
}

func publicFiles(source string) (map[string]string, error) {
	info, err := os.Lstat(source)
	if err != nil || !info.IsDir() {
		return nil, errors.New("real generated public directory required")
	}
	s := &publicFileSet{root: source, files: map[string]string{}}
	err = filepath.WalkDir(source, s.visit)
	return s.files, err
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
	if err = ownedWrite(filepath.Join(forgejoContext, "presentation.json"), data, 0o644); err != nil {
		return "", err
	}
	if err = ownedWrite(filepath.Join(context, "rootfs/usr/share/soda/host-image/presentation.json"), data, 0o644); err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

func validateCompletePayload(p appliancerelease.Payload) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Format != 3 {
		return errors.New("new candidates require the shared-layout v3 payload")
	}
	return nil
}

func writeCompleteQuadlets(source, root string, p appliancerelease.Payload) error {
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
		if err = ownedWrite(filepath.Join(root, "usr/share/containers/systemd", unit), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func verifyPublicBrandingAndMOTD(root string) error {
	for _, dir := range []string{"etc/cockpit/branding", "usr/share/soda/fastfetch", "etc/fastfetch"} {
		if _, err := publicFiles(filepath.Join(root, dir)); err != nil {
			return err
		}
	}
	motd, err := os.Lstat(filepath.Join(root, "etc/motd"))
	if err != nil || motd.Mode() != 0o644 {
		return errors.New("regular public MOTD required")
	}
	return nil
}

func writeFactoryDefaults(root string) error {
	for _, name := range []string{"forgejo.env", "proxy.Caddyfile"} {
		b, err := os.ReadFile(filepath.Join(root, "etc/soda", name))
		if err != nil {
			return err
		}
		if err = ownedWrite(filepath.Join(root, "usr/share/soda/defaults", name), b, 0o644); err != nil {
			return err
		}
	}
	// Machine setup must supply the explicit private subnet. Images are not saved
	// here: the vendor helper reads their immutable IDs from the release owner.
	example := []byte("{\n  \"network\": \"soda-projects\",\n  \"bridge\": \"soda0\",\n  \"subnet\": \"\",\n  \"tailnet_management\": false\n}\n")
	return ownedWrite(filepath.Join(root, "usr/share/soda/defaults/host.example.json"), example, 0o644)
}

func configureCompleteSystemd(source, root string) error {
	for from, to := range map[string]string{
		"appliance/host-image/soda-image-import.service": "usr/lib/systemd/system/soda-image-import.service",
		"appliance/host-image/retained-images.conf":      "usr/lib/systemd/system/soda-host.service.d/10-images.conf",
	} {
		b, err := os.ReadFile(filepath.Join(source, from))
		if err != nil {
			return err
		}
		if err = ownedWrite(filepath.Join(root, to), b, 0o644); err != nil {
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
	return ownedWrite(project, body, 0o644)
}

func writeReleaseMetadataAndNormalize(root string, p appliancerelease.Payload) error {
	// The complete payload is the sole resolved-image/defaults owner. Leave only
	// a pointer in the earlier host-content build marker, not stale scope data.
	if err := ownedWrite(filepath.Join(root, "usr/share/soda/host-image/build.json"), []byte("{\"Scope\":\"complete-local-payload\",\"ReleaseMetadata\":\"/usr/share/soda/release.json\"}\n"), 0o644); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err = ownedWrite(filepath.Join(root, "usr/share/soda/release.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	// Normalize only fresh image context directories, not credentials or sources.
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return nil
	})
}

// Complete binds all callers to one v3 payload and ordinary Podman storage.
func Complete(source, context, archives string, p appliancerelease.Payload, run nativebuild.BuildExec) error {
	if err := validateCompletePayload(p); err != nil {
		return err
	}
	root := filepath.Join(context, "rootfs")
	if err := writeCompleteQuadlets(source, root, p); err != nil {
		return err
	}
	if err := verifyPublicBrandingAndMOTD(root); err != nil {
		return err
	}
	if err := writeFactoryDefaults(root); err != nil {
		return err
	}
	if err := configureCompleteSystemd(source, root); err != nil {
		return err
	}
	if err := stageImages(archives, filepath.Join(root, "usr/share/soda/images"), p, run); err != nil {
		return err
	}
	return writeReleaseMetadataAndNormalize(root, p)
}

func rewriteQuadletLine(line, reference string, image, container, unit *int) ([]string, error) {
	switch {
	case line == "[Unit]":
		*unit++
		return []string{line, "Requires=soda-image-import.service", "After=soda-image-import.service"}, nil
	case line == "[Container]":
		*container++
		return []string{line, "Pull=never"}, nil
	case strings.HasPrefix(line, "Image="):
		*image++
		return []string{"Image=" + reference}, nil
	case strings.HasPrefix(line, "Pull="):
		return nil, nil
	case strings.HasPrefix(line, "GlobalArgs="):
		return nil, errors.New("unexpected existing Quadlet storage arguments")
	default:
		return []string{line}, nil
	}
}

func LocalQuadlet(body, reference string) (string, error) {
	lines := strings.Split(body, "\n")
	image, container, unit := 0, 0, 0
	var out []string
	for _, line := range lines {
		extra, err := rewriteQuadletLine(line, reference, &image, &container, &unit)
		if err != nil {
			return "", err
		}
		out = append(out, extra...)
	}
	if image != 1 || container != 1 || unit != 1 {
		return "", errors.New("one fixed Quadlet container/image required")
	}
	return strings.Join(out, "\n"), nil
}
