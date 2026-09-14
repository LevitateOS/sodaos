package acceptance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// LaunchDiskVM boots an independently admitted, disposable installed disk (or a
// blank installer target), without bypassing the installer through Ignition. The
// caller owns disk creation/admission and initial console/SSH provisioning. VM
// process custody and shutdown are shared with ordinary native acceptance VMs.
func LaunchDiskVM(ctx context.Context, c VMConfig, disk, variables, iso string, e *Evidence) (*VM, error) {
	if c.Architecture != "x86_64" || nativebuild.RequireNative(c.Architecture) != nil {
		return nil, errors.New("native x86_64 qualification required")
	}
	if !strings.HasPrefix(c.Name, "soda-native-") || c.Work == "" || disjointVMWork(c.Work, e) != nil {
		return nil, errors.New("private distinct VM work/evidence required")
	}
	for _, p := range []string{disk, variables, c.Firmware, c.QEMU, c.Work} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, ",\n\r") {
			return nil, errors.New("safe absolute VM path required")
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil || resolved != p {
			return nil, errors.New("VM paths must exist without symlink traversal")
		}
	}
	if len(filepath.Join(c.Work, "qmp.sock")) >= 100 || c.SSH.Port < 1024 || c.SSH.Port > 65535 {
		return nil, errors.New("short QMP path and explicit unprivileged SSH port required")
	}
	if _, err := os.Lstat(filepath.Join(c.Work, "qmp.sock")); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("fresh unoccupied QMP socket required")
	}
	st, err := os.Lstat(disk)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0222 == 0 {
		return nil, errors.New("disposable writable disk required")
	}
	for _, path := range []string{disk, variables, c.Firmware, c.QEMU} {
		sum, err := nativebuild.HashFile(path)
		if err != nil {
			return nil, err
		}
		if err = e.WriteJSON("input-"+filepath.Base(path)+".json", map[string]string{"sha256": sum}); err != nil {
			return nil, err
		}
	}
	args := []string{"-name", c.Name, "-machine", "q35,accel=kvm", "-cpu", "host", "-smp", "4", "-m", "12288", "-nodefaults", "-no-user-config", "-vga", "std", "-display", "none", "-monitor", "none", "-serial", "stdio",
		"-drive", "if=pflash,format=raw,readonly=on,file=" + c.Firmware, "-drive", "if=pflash,format=raw,file=" + variables,
		"-drive", "if=virtio,format=qcow2,file=" + disk + ",serial=soda-qualification",
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:" + strconv.Itoa(c.SSH.Port) + "-:22",
		"-qmp", "unix:" + filepath.Join(c.Work, "qmp.sock") + ",server=on,wait=off"}
	if iso != "" {
		if !filepath.IsAbs(iso) || strings.ContainsAny(iso, ",\n\r") {
			return nil, errors.New("safe ISO path required")
		}
		sum, err := nativebuild.HashFile(iso)
		if err != nil {
			return nil, err
		}
		if err = e.WriteJSON("input-iso.json", map[string]string{"sha256": sum}); err != nil {
			return nil, err
		}
		args = append(args, "-drive", "if=none,id=install-media-drive,format=raw,readonly=on,file="+iso, "-device", "ide-cd,id=install-media,drive=install-media-drive,bootindex=1")
	}
	v := &VM{config: c, evidence: e, bootArgs: args}
	if err = v.start(ctx); err != nil {
		return v, errors.Join(err, v.Close())
	}
	return v, nil
}

func (v *VM) Console() QMPClient { return v.qmp }

func (v *VM) EjectInstallationMedia(ctx context.Context) error {
	if err := v.qmp.Execute(ctx, "eject", "remove-install-media", map[string]any{"id": "install-media", "force": true}, nil); err != nil {
		return err
	}
	var blocks []struct {
		Device   string `json:"device"`
		Inserted any    `json:"inserted"`
	}
	if err := v.qmp.Execute(ctx, "query-block", "verify-media-removal", nil, &blocks); err != nil {
		return err
	}
	found := false
	for _, b := range blocks {
		if b.Device == "install-media-drive" {
			found = true
			if b.Inserted != nil {
				return errors.New("installation media remains inserted")
			}
		}
	}
	if !found {
		return errors.New("installation media backend missing from readback")
	}
	return v.evidence.WriteJSON(fmt.Sprintf("media-removed-%d.json", v.attempt), blocks)
}
