package build

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
	root, err := os.OpenRoot("/")
	if err != nil {
		return err
	}
	defer root.Close()
	return verifyInstalled(root, inv, revision, func(container string) (string, error) {
		phase, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(phase, "podman", "--remote=false", "container", "inspect", "--format", "{{.Image}}", container)
		cmd.WaitDelay = time.Second
		var output strings.Builder
		cmd.Stdout = &limitWriter{w: &output, remaining: 256}
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return strings.TrimSpace(output.String()), nil
	})
}

// Explicit filesystem/native-command inputs allow failure fixtures without
// reading or altering the machine running the source tests.
func verifyInstallMarkers(root *os.Root, revision string) error {
	st, err := root.Lstat("etc/soda/install-started")
	if err != nil || !st.Mode().IsRegular() || st.Size() > 64 {
		return errors.New("bounded regular install marker required")
	}
	marker, err := root.ReadFile("etc/soda/install-started")
	if err != nil || strings.TrimSpace(string(marker)) != revision {
		return errors.New("installed attempt is not bound to this source")
	}
	if st, err := root.Lstat("etc/soda/installed"); err != nil || !st.Mode().IsRegular() {
		return errors.New("regular installed marker required")
	}
	return nil
}

func resolveWritablePrefix(root *os.Root) (string, error) {
	localPrefix := "usr/local/"
	if local, err := root.Lstat("usr/local"); err == nil && local.Mode()&os.ModeSymlink != 0 {
		link, err := root.Readlink("usr/local")
		if err != nil || (link != "../var/usrlocal" && link != "/var/usrlocal") {
			return "", errors.New("unexpected CoreOS writable-prefix link")
		}
		localPrefix = "var/usrlocal/"
	}
	for _, retired := range []string{
		localPrefix + "share/cockpit/soda-runners",
		localPrefix + "share/cockpit/soda-tailscale",
		"usr/share/cockpit/soda-runners",
		"usr/share/cockpit/soda-tailscale",
	} {
		if _, e := root.Lstat(retired); !errors.Is(e, os.ErrNotExist) {
			return "", errors.New("retired Cockpit package remains; removal requires its own authorization")
		}
	}
	return localPrefix, nil
}

func isDeliveredFileSelected(name string) bool {
	if strings.HasPrefix(name, "rootfs/etc/systemd/system/soda-") ||
		strings.HasPrefix(name, "rootfs/usr/local/") ||
		strings.HasPrefix(name, "rootfs/etc/cockpit/branding/") ||
		strings.HasPrefix(name, "rootfs/var/lib/soda/forgejo/gitea/public/") ||
		name == "rootfs/etc/pam.d/cockpit" {
		return true
	}
	for _, sodaFile := range forgejoFiles {
		if name == sodaFile {
			return true
		}
	}
	return false
}

func verifyDeliveredFile(root *os.Root, dest string, file File) error {
	if file.Link != "" {
		target, err := root.Readlink(dest)
		if err != nil || target != file.Link {
			return errors.New("delivered link changed")
		}
		return nil
	}
	sum, err := HashAt(root, dest)
	if err != nil || sum != file.SHA256 {
		return fmt.Errorf("delivered bytes differ: %s", filepath.Base(dest))
	}
	st, err := root.Lstat(dest)
	if err != nil || uint32(st.Mode().Perm()) != file.Mode {
		return errors.New("delivered mode changed")
	}
	return nil
}

func verifyDeliveredInventory(root *os.Root, files map[string]File, localPrefix string) error {
	for name, file := range files {
		if file.Directory || !isDeliveredFileSelected(name) {
			continue
		}
		dest := strings.TrimPrefix(name, "rootfs/")
		if strings.HasPrefix(dest, "usr/local/") {
			dest = localPrefix + strings.TrimPrefix(dest, "usr/local/")
		}
		if err := verifyDeliveredFile(root, dest, file); err != nil {
			return err
		}
	}
	return nil
}

func verifyActiveServices(root *os.Root, images map[string]Image, inspect func(string) (string, error)) error {
	services := map[string]string{"forgejo": "soda-forgejo"}
	st, err := root.Lstat("etc/soda/activated")
	if err == nil {
		if !st.Mode().IsRegular() {
			return errors.New("regular activation marker required")
		}
		services["dashboard"] = "soda-dashboard"
		services["caddy"] = "soda-proxy"
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for image, container := range services {
		result, err := inspect(container)
		if err != nil {
			return fmt.Errorf("inspect native %s image: %w", image, err)
		}
		if strings.TrimPrefix(strings.TrimSpace(result), "sha256:") != strings.TrimPrefix(images[image].Config, "sha256:") {
			return errors.New("service is using a different image than the selected bundle")
		}
	}
	return nil
}

func verifyInstalled(root *os.Root, inv Inventory, revision string, inspect func(string) (string, error)) error {
	if err := verifyInstallMarkers(root, revision); err != nil {
		return err
	}
	localPrefix, err := resolveWritablePrefix(root)
	if err != nil {
		return err
	}
	if err := verifyDeliveredInventory(root, inv.Files, localPrefix); err != nil {
		return err
	}
	return verifyActiveServices(root, inv.Images, inspect)
}
