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

func validateDiskVMSocketAndPort(work string, port int) error {
	if len(filepath.Join(work, "qmp.sock")) >= 100 || port < 1024 || port > 65535 {
		return errors.New("short QMP path and explicit unprivileged SSH port required")
	}
	if _, err := os.Lstat(filepath.Join(work, "qmp.sock")); !errors.Is(err, os.ErrNotExist) {
		return errors.New("fresh unoccupied QMP socket required")
	}
	return nil
}

func validateDiskVMConfig(c VMConfig, e *Evidence) error {
	if c.Architecture != "x86_64" || nativebuild.RequireNative(c.Architecture) != nil {
		return errors.New("native x86_64 qualification required")
	}
	if !strings.HasPrefix(c.Name, "soda-native-") || c.Work == "" || disjointVMWork(c.Work, e) != nil {
		return errors.New("private distinct VM work/evidence required")
	}
	return validateDiskVMSocketAndPort(c.Work, c.SSH.Port)
}

func validateDiskFile(disk string) error {
	st, err := os.Lstat(disk)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o222 == 0 {
		return errors.New("disposable writable disk required")
	}
	return nil
}

func validateDiskVMPaths(c VMConfig, disk, variables string) error {
	for _, p := range []string{disk, variables, c.Firmware, c.QEMU, c.Work} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.ContainsAny(p, ",\n\r") {
			return errors.New("safe absolute VM path required")
		}
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil || resolved != p {
			return errors.New("VM paths must exist without symlink traversal")
		}
	}
	return validateDiskFile(disk)
}

func recordDiskVMInputs(paths []string, e *Evidence) error {
	for _, path := range paths {
		sum, err := nativebuild.HashFile(path)
		if err != nil {
			return err
		}
		if err = e.WriteJSON("input-"+filepath.Base(path)+".json", map[string]string{"sha256": sum}); err != nil {
			return err
		}
	}
	return nil
}

func buildDiskVMArgs(c VMConfig, disk, variables string) []string {
	return []string{
		"-name", c.Name, "-machine", "q35,accel=kvm", "-cpu", "host", "-smp", "4", "-m", "12288", "-nodefaults", "-no-user-config", "-vga", "std", "-display", "none", "-monitor", "none", "-serial", "stdio",
		"-drive", "if=pflash,format=raw,readonly=on,file=" + c.Firmware, "-drive", "if=pflash,format=raw,file=" + variables,
		"-drive", "if=none,id=target-disk,format=qcow2,file=" + disk,
		"-device", "virtio-blk-pci,drive=target-disk,serial=soda-qualification",
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:" + strconv.Itoa(c.SSH.Port) + "-:22",
		"-qmp", "unix:" + filepath.Join(c.Work, "qmp.sock") + ",server=on,wait=off",
	}
}

func recordAndAppendDiskVMISO(args []string, iso string, e *Evidence) ([]string, error) {
	if iso == "" {
		return args, nil
	}
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
	return append(args,
		"-drive", "if=none,id=install-media-drive,format=raw,readonly=on,file="+iso,
		"-device", "ide-cd,id=install-media,drive=install-media-drive,bootindex=1",
	), nil
}

// LaunchDiskVM boots an independently admitted, disposable installed disk (or a
// blank installer target), without bypassing the installer through Ignition. The
// caller owns disk creation/admission and initial console/SSH provisioning. VM
// process custody and shutdown are shared with ordinary native acceptance VMs.
func LaunchDiskVM(ctx context.Context, c VMConfig, disk, variables, iso string, e *Evidence) (*VM, error) {
	if err := validateDiskVMConfig(c, e); err != nil {
		return nil, err
	}
	if err := validateDiskVMPaths(c, disk, variables); err != nil {
		return nil, err
	}
	if err := recordDiskVMInputs([]string{disk, variables, c.Firmware, c.QEMU}, e); err != nil {
		return nil, err
	}
	args, err := recordAndAppendDiskVMISO(buildDiskVMArgs(c, disk, variables), iso, e)
	if err != nil {
		return nil, err
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
