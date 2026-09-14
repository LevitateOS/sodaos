package hostimage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

func inspectComplete(context, out, id string, capture nativebuild.BuildCapture) error {
	run := func(name, entry string, args ...string) (string, error) {
		command := []string{"--remote=false", "run", "--cidfile", filepath.Join(out, name+".cid"), "--network=none", "--read-only", "--entrypoint=" + entry, id}
		command = append(command, args...)
		return capture(context, "podman", command...)
	}
	observed, err := run("payload-inspect", "/usr/bin/cat", appliancerelease.Path)
	if err != nil {
		return err
	}
	expected, err := os.ReadFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return err
	}
	if observed != strings.TrimSpace(string(expected)) {
		return errors.New("image payload metadata differs from candidate")
	}
	var p appliancerelease.Payload
	if err = json.Unmarshal(expected, &p); err != nil {
		return err
	}
	if err = p.Validate(); err != nil {
		return err
	}
	files, _, err := appliancerelease.VerifyContent(p, filepath.Join(context, "rootfs", appliancerelease.ImagesPath))
	if err != nil {
		return err
	}
	var paths []string
	for name := range files {
		paths = append(paths, name)
	}
	slices.Sort(paths)
	var installedPaths []string
	for _, name := range paths {
		installedPaths = append(installedPaths, appliancerelease.ImagesPath+"/"+name)
	}
	sums, err := run("content-inspect", "/usr/bin/sha256sum", installedPaths...)
	if err != nil {
		return err
	}
	lines := strings.Split(sums, "\n")
	if len(lines) != len(paths) {
		return errors.New("incomplete embedded content inventory")
	}
	for i, name := range paths {
		if lines[i] != files[name]+"  "+installedPaths[i] {
			return errors.New("host content differs from payload")
		}
	}
	consoleHash, err := nativebuild.HashFile(filepath.Join(out, "tools/soda-installer"))
	if err != nil {
		return err
	}
	console, err := run("installer-inspect", "/usr/bin/sha256sum", "/usr/libexec/soda/soda-install")
	if err != nil || console != consoleHash+"  /usr/libexec/soda/soda-install" {
		return errors.New("image installer differs from prebuilt tool")
	}
	generated, err := run("quadlet-inspect", "/usr/lib/systemd/system-generators/podman-system-generator", "--dryrun")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "generated-quadlets.txt"), []byte(generated+"\n"), 0600); err != nil {
		return err
	}
	if strings.Contains(generated, "additionalimagestore") || !strings.Contains(generated, "soda-image-import.service") {
		return errors.New("native Quadlet generator lost ordinary Podman import ordering")
	}
	for _, name := range []string{"forgejo", "dashboard", "proxy"} {
		if !strings.Contains(generated, p.Images[name].Config) {
			return errors.New("native Quadlet generator lost image binding")
		}
	}
	return nil
}
