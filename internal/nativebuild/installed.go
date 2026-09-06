package nativebuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VerifyInstalled binds immutable delivered files and current service image
// identities to a bundle. It does not compare mutable config/databases/project
// containers or replace the core's acceptance assertions.
func VerifyInstalled(ctx context.Context, bundle, arch, revision string) error {
	if err := RequireNative(arch); err != nil {
		return err
	}
	if os.Geteuid() != 0 {
		return errors.New("native root inspection required")
	}
	inv, err := Verify(bundle, arch, revision)
	if err != nil {
		return err
	}
	marker, err := os.ReadFile("/etc/soda/install-started")
	if err != nil || strings.TrimSpace(string(marker)) != revision {
		return errors.New("installed attempt is not bound to this source")
	}
	if _, err = os.Stat("/etc/soda/installed"); err != nil {
		return err
	}
	for name, file := range inv.Files {
		if file.Directory {
			continue
		}
		selected := strings.HasPrefix(name, "rootfs/usr/local/") || strings.HasPrefix(name, "rootfs/etc/cockpit/branding/") || strings.HasPrefix(name, "rootfs/var/lib/soda/forgejo/gitea/public/") || name == "rootfs/etc/pam.d/cockpit" || name == "rootfs/etc/cockpit/users.override.json"
		if !selected {
			continue
		}
		dest := "/" + strings.TrimPrefix(name, "rootfs/")
		if file.Link != "" {
			target, err := os.Readlink(dest)
			if err != nil || target != file.Link {
				return errors.New("delivered link changed")
			}
			continue
		}
		sum, err := HashFile(dest)
		if err != nil || sum != file.SHA256 {
			return fmt.Errorf("delivered bytes differ: %s", filepath.Base(dest))
		}
		st, err := os.Lstat(dest)
		if err != nil || uint32(st.Mode().Perm()) != file.Mode {
			return errors.New("delivered mode changed")
		}
	}
	services := map[string]string{"forgejo": "soda-forgejo"}
	if _, err = os.Stat("/etc/soda/activated"); err == nil {
		services["dashboard"] = "soda-dashboard"
		services["caddy"] = "soda-proxy"
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for image, container := range services {
		phase, cancel := context.WithTimeout(ctx, 30*time.Second)
		cmd := exec.CommandContext(phase, "podman", "--remote=false", "container", "inspect", "--format", "{{.Image}}", container)
		cmd.WaitDelay = time.Second
		result, err := cmd.Output()
		cancel()
		if err != nil {
			return fmt.Errorf("inspect native %s image: %w", image, err)
		}
		if strings.TrimPrefix(strings.TrimSpace(string(result)), "sha256:") != strings.TrimPrefix(inv.Images[image].Config, "sha256:") {
			return errors.New("service is using a different image than the selected bundle")
		}
	}
	return nil
}
