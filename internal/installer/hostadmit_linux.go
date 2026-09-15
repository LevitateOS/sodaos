package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/release/build"
)

func liveISOFromCmdline(data []byte) bool {
	for _, arg := range strings.Fields(string(data)) {
		if arg == "coreos.liveiso" || strings.HasPrefix(arg, "coreos.liveiso=") || strings.HasPrefix(arg, "coreos.live.rootfs_url=") {
			return true
		}
	}
	return false
}

func admitLiveInstaller(values map[string]string) error {
	var media mediaIdentity
	if err := build.ReadJSON(filepath.Join(dataDir, "media.json"), &media); err != nil {
		return errors.New("missing media identity")
	}
	// Selected CoreOS reports the Fedora major in VERSION_ID (44), and
	// the exact image release in IMAGE_VERSION (44.20260817.3.2).
	if err := media.validate(values["IMAGE_VERSION"], architecture()); err != nil {
		return err
	}
	observed, err := command(context.Background(), "coreos-installer", []string{"--version"}, nil)
	if err != nil || strings.TrimSpace(string(observed)) != media.InstallerVersion {
		return errors.New("unreviewed CoreOS Installer version")
	}
	return nil
}

func selinuxEnforcing() error {
	enforcing, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil || strings.TrimSpace(string(enforcing)) != "1" {
		return errors.New("SELinux must remain enforcing")
	}
	return nil
}

func coreOSHost(live bool) error {
	// Display branding in /etc must not become a stale copy of base identity.
	data, err := os.ReadFile("/usr/lib/os-release")
	if err != nil {
		return err
	}
	values := osRelease(data)
	if values["ID"] != "fedora" || values["VARIANT_ID"] != "coreos" {
		return errors.New("upstream Fedora CoreOS required")
	}
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return err
	}
	if live != liveISOFromCmdline(cmdline) {
		return errors.New("disk action requires the live ISO; continuation requires the installed host")
	}
	if live {
		if err := admitLiveInstaller(values); err != nil {
			return err
		}
	}
	return selinuxEnforcing()
}

func osRelease(data []byte) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[k] = strings.Trim(v, "\"'")
	}
	return values
}

func routes(ctx context.Context, run commandRunner) ([]string, error) {
	data, err := run(ctx, "ip", []string{"-json", "-4", "route", "show", "table", "all"}, nil)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Destination string `json:"dst"`
	}
	if err = json.Unmarshal(data, &entries); err != nil {
		return nil, errors.New("cannot inspect current routes")
	}
	var result []string
	for _, entry := range entries {
		result = append(result, entry.Destination)
	}
	return result, nil
}
