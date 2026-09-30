package image

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

func inspectCompletePayload(context, out, id string, run func(string, string, ...string) (string, error)) (deliver.Payload, error) {
	observed, err := run("payload-inspect", "/usr/bin/cat", deliver.Path)
	if err != nil {
		return deliver.Payload{}, err
	}
	expected, err := os.ReadFile(filepath.Join(out, "payload.json"))
	if err != nil {
		return deliver.Payload{}, err
	}
	if observed != strings.TrimSpace(string(expected)) {
		return deliver.Payload{}, errors.New("image payload metadata differs from candidate")
	}
	var p deliver.Payload
	if err = json.Unmarshal(expected, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}

func inspectCompleteContent(context, out string, p deliver.Payload, run func(string, string, ...string) (string, error)) error {
	files, _, err := deliver.VerifyContent(p, filepath.Join(context, "rootfs", deliver.ImagesPath))
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
		installedPaths = append(installedPaths, deliver.ImagesPath+"/"+name)
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
	return nil
}

func inspectCompleteQuadlets(out string, p deliver.Payload, run func(string, string, ...string) (string, error)) error {
	generated, err := run("quadlet-inspect", "/usr/lib/systemd/system-generators/podman-system-generator", "--dryrun")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "generated-quadlets.txt"), []byte(generated+"\n"), 0o600); err != nil {
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

func inspectCompleteExtensionUnit(p deliver.Payload, run func(string, string, ...string) (string, error)) error {
	unit, err := run("extension-unit-inspect", "/usr/bin/cat", "/usr/lib/systemd/system/soda-extension-install.service")
	if err != nil {
		return err
	}
	if strings.Contains(unit, "localhost/soda-extension:dev") || !strings.Contains(unit, "--entrypoint=/usr/local/bin/gitea "+p.Images["extension"].Config+" ") {
		return errors.New("host image lost Soda extension installer binding")
	}
	return nil
}

func inspectCompleteServiceFiles(context string, run func(string, string, ...string) (string, error)) error {
	for _, path := range []string{
		"/usr/share/containers/systemd/forgejo.container",
		"/usr/lib/systemd/system/soda-extension-install.service",
	} {
		want, err := build.HashFile(filepath.Join(context, "rootfs", strings.TrimPrefix(path, "/")))
		if err != nil {
			return err
		}
		got, err := run("service-hash-inspect", "/usr/bin/sha256sum", path)
		if err != nil || got != want+"  "+path {
			return errors.New("host service differs from staged source")
		}
	}
	return nil
}

func inspectCompleteInventory(context string, run func(string, string, ...string) (string, error)) error {
	const path = "/usr/share/soda/host-image/content.json"
	want, err := os.ReadFile(filepath.Join(context, "rootfs", strings.TrimPrefix(path, "/")))
	if err != nil {
		return err
	}
	got, err := run("inventory-inspect", "/usr/bin/cat", path)
	if err != nil || got+"\n" != string(want) {
		return errors.New("host content inventory differs from staged source")
	}
	return nil
}

func inspectComplete(context, out, id string, capture build.BuildCapture) error {
	run := func(name, entry string, args ...string) (string, error) {
		command := []string{"--remote=false", "run", "--cidfile", filepath.Join(out, name+".cid"), "--network=none", "--read-only", "--entrypoint=" + entry, id}
		command = append(command, args...)
		return capture(context, "podman", command...)
	}
	p, err := inspectCompletePayload(context, out, id, run)
	if err != nil {
		return err
	}
	if err = inspectCompleteContent(context, out, p, run); err != nil {
		return err
	}
	consoleHash, err := build.HashFile(filepath.Join(out, "tools/soda-installer"))
	if err != nil {
		return err
	}
	console, err := run("installer-inspect", "/usr/bin/sha256sum", "/usr/libexec/soda/soda-install")
	if err != nil || console != consoleHash+"  /usr/libexec/soda/soda-install" {
		return errors.New("image installer differs from prebuilt tool")
	}
	if err = inspectCompleteQuadlets(out, p, run); err != nil {
		return err
	}
	if err = inspectCompleteExtensionUnit(p, run); err != nil {
		return err
	}
	if err = inspectCompleteServiceFiles(context, run); err != nil {
		return err
	}
	return inspectCompleteInventory(context, run)
}
