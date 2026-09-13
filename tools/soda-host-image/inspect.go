package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
)

func inspectComplete(context, out, id string, capture buildCapture) error {
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
	generated, err := run("quadlet-inspect", "/usr/lib/systemd/system-generators/podman-system-generator", "--dryrun")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "generated-quadlets.txt"), []byte(generated+"\n"), 0600); err != nil {
		return err
	}
	if !strings.Contains(generated, "additionalimagestore=/usr/lib/bootc/storage") {
		return errors.New("native Quadlet generator lost bound image storage")
	}
	for _, name := range []string{"forgejo", "dashboard", "proxy"} {
		if !strings.Contains(generated, p.Images[name].Reference) {
			return errors.New("native Quadlet generator lost image binding")
		}
	}
	return nil
}
