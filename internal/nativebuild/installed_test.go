package nativebuild

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledIdentityUsesExactBytesAndActivationPhase(t *testing.T) {
	for _, mode := range []string{"before", "activated", "coreos-absolute", "coreos-relative", "revision", "bytes", "mode", "image", "inspection-error"} {
		t.Run(mode, func(t *testing.T) {
			path := t.TempDir()
			for _, dir := range []string{"etc/soda", "usr/local/libexec/soda"} {
				if err := os.MkdirAll(filepath.Join(path, dir), 0755); err != nil {
					t.Fatal(err)
				}
			}
			rev := fixtureRevision
			if mode == "revision" {
				rev = strings.Repeat("b", 40)
			}
			for name, body := range map[string]string{"etc/soda/install-started": rev + "\n", "etc/soda/installed": "", "usr/local/libexec/soda/fixture": "fixture bytes"} {
				if err := os.WriteFile(filepath.Join(path, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			file := filepath.Join(path, "usr/local/libexec/soda/fixture")
			hash, err := HashFile(file)
			if err != nil {
				t.Fatal(err)
			}
			image := "sha256:" + strings.Repeat("c", 64)
			inv := Inventory{Files: map[string]File{"rootfs/usr/local/libexec/soda/fixture": {SHA256: hash, Mode: 0600}}, Images: map[string]Image{"forgejo": {Config: image}, "dashboard": {Config: image}, "caddy": {Config: image}}}
			if mode == "bytes" {
				if err := os.WriteFile(file, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "mode" {
				if err := os.Chmod(file, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "activated" {
				if err := os.WriteFile(filepath.Join(path, "etc/soda/activated"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if strings.HasPrefix(mode, "coreos-") {
				if err := os.Mkdir(filepath.Join(path, "var"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(path, "usr/local"), filepath.Join(path, "var/usrlocal")); err != nil {
					t.Fatal(err)
				}
				target := "../var/usrlocal"
				if mode == "coreos-absolute" {
					target = "/var/usrlocal"
				}
				if err := os.Symlink(target, filepath.Join(path, "usr/local")); err != nil {
					t.Fatal(err)
				}
			}
			root, err := os.OpenRoot(path)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			called := map[string]bool{}
			err = verifyInstalled(root, inv, fixtureRevision, func(container string) (string, error) {
				called[container] = true
				if mode == "inspection-error" {
					return "", errors.New("synthetic native failure")
				}
				if mode == "image" {
					return "wrong", nil
				}
				return image, nil
			})
			if mode != "before" && mode != "activated" && !strings.HasPrefix(mode, "coreos-") {
				if err == nil {
					t.Fatal("accepted changed/failed installed observation")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !called["soda-forgejo"] || (mode == "activated" && len(called) != 3) || (mode == "before" && len(called) != 1) {
				t.Fatal("wrong activation-phase inspection")
			}
		})
	}
}
