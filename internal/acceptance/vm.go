// QEMU ownership adapted from soda-os bc1d3e0; CoreOS fixtures only.
package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

type VMConfig struct {
	Name, Architecture, CoreOSLock, BaseReceipt, Ignition, QEMU, Firmware, Variables, Work string
	DiskGiB                                                                                int
	SSH                                                                                    Remote
}
type VM struct {
	config   VMConfig
	process  *Process
	qmp      QMPClient
	outputs  []io.WriteCloser
	evidence *Evidence
	attempt  int
	once     sync.Once
	closeErr error
}

func (c VMConfig) preflight(e *Evidence) error {
	if err := nativebuild.RequireNative(c.Architecture); err != nil {
		return err
	}
	if c.DiskGiB < 1 || c.DiskGiB > 1024 {
		return errors.New("fresh disk size must be 1..1024 GiB")
	}
	if !regexp.MustCompile(`^soda-native-[a-z0-9-]+$`).MatchString(c.Name) {
		return errors.New("fresh soda-native-* fixture name required")
	}
	for _, p := range []string{c.CoreOSLock, c.BaseReceipt, c.Ignition, c.QEMU, c.Firmware, c.Variables, c.Work, c.SSH.Key, c.SSH.KnownHosts} {
		if !filepath.IsAbs(p) || strings.ContainsAny(p, ",\n\r") {
			return errors.New("absolute paths without QEMU separators required")
		}
	}
	if len(filepath.Join(c.Work, "qmp.sock")) > 100 {
		return errors.New("select a shorter private work directory for QMP")
	}
	for _, pair := range [][2]string{{c.Work, e.Path()}, {e.Path(), c.Work}} {
		r, err := filepath.Rel(pair[0], pair[1])
		if err != nil {
			return err
		}
		if r == "." || (!strings.HasPrefix(r, ".."+string(filepath.Separator)) && r != "..") {
			return errors.New("VM work and evidence must be disjoint")
		}
	}
	if c.SSH.Host != "127.0.0.1" || c.SSH.User != "root" {
		return errors.New("CoreOS fixture SSH must be root on loopback")
	}
	if _, err := c.SSH.Args(); err != nil {
		return err
	}
	if _, err := PrivateFile(c.Ignition); err != nil {
		return err
	}
	if err := VerifyFixtureTrust(c.Ignition, c.Name, c.SSH); err != nil {
		return err
	}
	if _, err := os.Lstat(c.Work); !errors.Is(err, os.ErrNotExist) {
		return errors.New("fresh unoccupied VM work path required")
	}
	for _, p := range []string{c.QEMU, c.Firmware, c.Variables} {
		s, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if !s.Mode().IsRegular() || s.Size() <= 0 || s.Size() > 128<<20 {
			return errors.New("bounded regular QEMU/firmware input required")
		}
	}
	if _, err := exec.LookPath(c.QEMU); err != nil {
		return err
	}
	for _, tool := range []string{"qemu-img", "ssh"} {
		if _, err := exec.LookPath(tool); err != nil {
			return err
		}
	}
	kvm, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err = kvm.Close(); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(c.SSH.Port)))
	if err != nil {
		return err
	}
	return listener.Close()
}
func LaunchVM(ctx context.Context, c VMConfig, e *Evidence) (*VM, error) {
	if c.DiskGiB == 0 {
		c.DiskGiB = 64
	}
	if err := c.preflight(e); err != nil {
		return nil, err
	}
	var base nativebuild.VerifiedBase
	if err := nativebuild.ReadJSON(c.BaseReceipt, &base); err != nil {
		return nil, err
	}
	l, img, err := nativebuild.ReadCoreOS(c.CoreOSLock, c.Architecture)
	if err != nil {
		return nil, err
	}
	if base.Architecture != c.Architecture || base.Release != l.Release || base.SHA256 != img.UncompressedSHA256 {
		return nil, errors.New("base does not match selected CoreOS input")
	}
	if !filepath.IsAbs(base.Path) || strings.ContainsAny(base.Path, ",\n\r") {
		return nil, errors.New("unsafe base path")
	}
	baseStat, err := os.Lstat(base.Path)
	if err != nil {
		return nil, err
	}
	if baseStat.Mode().Perm()&0222 != 0 {
		return nil, errors.New("verified base must be read-only; fetch a fresh cache, never chmod a live base")
	}
	if !regexp.MustCompile(`^(?:[A-F0-9]{40}|[A-F0-9]{64})$`).MatchString(base.Signer) {
		return nil, errors.New("verified base receipt lacks selected signer")
	}
	sum, err := nativebuild.HashFile(base.Path)
	if err != nil || sum != base.SHA256 {
		return nil, errors.Join(err, errors.New("base checksum mismatch"))
	}
	firmwareHash, err := nativebuild.HashFile(c.Firmware)
	if err != nil {
		return nil, err
	}
	varsHash, err := nativebuild.HashFile(c.Variables)
	if err != nil {
		return nil, err
	}
	description := struct{ Name, Architecture, BaseSHA256, Release, FirmwareSHA256, VariablesSHA256, Work string }{c.Name, c.Architecture, base.SHA256, base.Release, firmwareHash, varsHash, c.Work}
	if err = e.WriteJSON("fixture.json", description); err != nil {
		return nil, err
	}
	version, err := Execute(ctx, e, "qemu-version", Command{Name: c.QEMU, Args: []string{"--version"}})
	if err != nil || version.Err != nil {
		return nil, errors.Join(err, version.Err)
	}
	version, err = Execute(ctx, e, "qemu-img-version", Command{Name: "qemu-img", Args: []string{"--version"}})
	if err != nil || version.Err != nil {
		return nil, errors.Join(err, version.Err)
	}
	// qemu-img obtains its normal image locks; never request unsafe -U access.
	r, err := Execute(ctx, e, "base-info", Command{Name: "qemu-img", Args: []string{"info", "--output=json", base.Path}})
	if err != nil || r.Err != nil {
		return nil, errors.Join(err, r.Err)
	}
	var info struct {
		Format  string `json:"format"`
		Backing string `json:"backing-filename"`
		Size    int64  `json:"virtual-size"`
	}
	if err = json.Unmarshal(r.Stdout, &info); err != nil {
		return nil, err
	}
	if info.Format != "qcow2" || info.Backing != "" || info.Size <= 0 || info.Size > int64(c.DiskGiB)<<30 {
		return nil, errors.New("standalone qcow2 base required")
	}
	if err = nativebuild.FreshDirectory(c.Work); err != nil {
		return nil, err
	}
	disk := filepath.Join(c.Work, "disk.qcow2")
	r, err = Execute(ctx, e, "disk-create", Command{Name: "qemu-img", Args: []string{"create", "-f", "qcow2", "-F", "qcow2", "-b", base.Path, disk, strconv.Itoa(c.DiskGiB) + "G"}})
	if err != nil || r.Err != nil {
		return nil, errors.Join(err, r.Err)
	}
	vars, err := os.ReadFile(c.Variables)
	if err != nil {
		return nil, err
	}
	if err = nativebuild.WriteNew(filepath.Join(c.Work, "vars.fd"), vars, 0600); err != nil {
		return nil, err
	}
	v := &VM{config: c, evidence: e}
	if err = v.start(ctx); err != nil {
		return v, errors.Join(err, v.Close())
	}
	return v, nil
}
func (c VMConfig) args() []string {
	machine := "q35"
	if c.Architecture == "aarch64" {
		machine = "virt"
	}
	return []string{"-name", c.Name, "-machine", machine + ",accel=kvm", "-cpu", "host", "-smp", "4", "-m", "8192", "-display", "none", "-monitor", "none", "-serial", "stdio",
		"-drive", "if=pflash,format=raw,readonly=on,file=" + c.Firmware, "-drive", "if=pflash,format=raw,file=" + filepath.Join(c.Work, "vars.fd"),
		"-drive", "if=virtio,format=qcow2,file=" + filepath.Join(c.Work, "disk.qcow2"), "-fw_cfg", "name=opt/com.coreos/config,file=" + c.Ignition,
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:" + strconv.Itoa(c.SSH.Port) + "-:22", "-qmp", "unix:" + filepath.Join(c.Work, "qmp.sock") + ",server=on,wait=off"}
}
func (v *VM) start(ctx context.Context) error {
	v.attempt++
	label := fmt.Sprintf("boot-%d", v.attempt)
	out, err := v.evidence.Writer(label + ".serial")
	if err != nil {
		return err
	}
	v.outputs = append(v.outputs, out)
	stderr, err := v.evidence.Writer(label + ".stderr")
	if err != nil {
		return err
	}
	v.outputs = append(v.outputs, stderr)
	v.process, err = StartProcess(ctx, Command{Name: v.config.QEMU, Args: v.config.args()}, out, stderr)
	if err != nil {
		return err
	}
	v.qmp = QMPClient{Socket: filepath.Join(v.config.Work, "qmp.sock")}
	ready, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		var status map[string]any
		if err = v.qmp.Execute(ready, "query-status", "status", nil, &status); err == nil {
			break
		}
		select {
		case <-v.process.Done():
			return errors.Join(errors.New("QEMU exited before readiness"), v.process.Wait(ctx))
		case <-ready.Done():
			return ready.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	sshCtx, stop := context.WithTimeout(ctx, 10*time.Minute)
	defer stop()
	process := v.process
	go func() {
		select {
		case <-process.Done():
			stop()
		case <-sshCtx.Done():
		}
	}()
	err = v.config.SSH.WaitReady(sshCtx)
	select {
	case <-v.process.Done():
		return errors.Join(errors.New("QEMU exited during SSH readiness"), v.process.Wait(ctx), err)
	default:
		return err
	}
}

// Restart reuses exactly this instance's disk and NVRAM. No data is removed.
func (v *VM) Restart(ctx context.Context) error {
	if err := v.powerDown(ctx); err != nil {
		return err
	}
	return v.start(ctx)
}
func (v *VM) powerDown(ctx context.Context) error {
	if v.process == nil {
		return nil
	}
	shutdown, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := v.qmp.Execute(shutdown, "system_powerdown", "powerdown", nil, nil); err != nil {
		return err
	}
	if err := v.process.Wait(shutdown); err != nil {
		return err
	}
	v.process = nil
	if err := os.Remove(v.qmp.Socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return v.closeOutputs()
}
func (v *VM) closeOutputs() error {
	var err error
	for _, w := range v.outputs {
		err = errors.Join(err, w.Close())
	}
	v.outputs = nil
	return err
}
func (v *VM) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-v.process.Done():
		return errors.Join(errors.New("guest exited unexpectedly"), v.process.Wait(ctx))
	}
}

// Close keeps disks/provisioning for inspection, even on failure. No enrollment
// is inferred or revoked; those explicit actions belong to their owning checks.
func (v *VM) Close() error {
	v.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		v.closeErr = v.powerDown(ctx)
		if v.process != nil {
			v.closeErr = errors.Join(v.closeErr, v.process.Stop())
			select {
			case <-v.process.Done():
			default:
				// A kernel-stuck child still owns the capture writers. Do not
				// close them concurrently or describe their retention as complete.
				process, outputs := v.process, v.outputs
				v.outputs = nil
				go func() {
					<-process.Done()
					for _, out := range outputs {
						_ = out.Close()
					}
				}()
				v.closeErr = errors.Join(v.closeErr, errors.New("VM capture still owned by incomplete cleanup"))
				return
			}
		}
		v.closeErr = errors.Join(v.closeErr, v.closeOutputs())
	})
	return v.closeErr
}
