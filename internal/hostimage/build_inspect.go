package hostimage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	var archivePaths []string
	for _, name := range appliancerelease.Names {
		archivePaths = append(archivePaths, appliancerelease.ImagesPath+"/"+name+".oci")
	}
	sums, err := run("archive-inspect", "/usr/bin/sha256sum", archivePaths...)
	if err != nil {
		return err
	}
	lines := strings.Split(sums, "\n")
	if len(lines) != len(appliancerelease.Names) {
		return errors.New("incomplete embedded archive inventory")
	}
	for i, name := range appliancerelease.Names {
		if lines[i] != p.Images[name].ArchiveSHA256+"  "+archivePaths[i] {
			return errors.New("host archive content differs from payload")
		}
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
