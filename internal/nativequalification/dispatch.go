package nativequalification

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

// Dispatch freezes independently checked input bytes in root custody. The builder
// cannot write this snapshot, the qualifier work, or the final controller receipt.
func Dispatch(ctx context.Context, c Config, candidate, revision, custody string, log io.Writer) error {
	if os.Geteuid() != 0 {
		return errors.New("protected qualification dispatcher required")
	}
	if err := acceptance.TrustedExecutable(c.Executable); err != nil {
		return err
	}
	driver, err := nativebuild.HashFile(c.Executable)
	if err != nil {
		return err
	}
	a, err := ReadArtifact(c.Baseline)
	if err != nil {
		return err
	}
	b, err := ReadArtifact(candidate)
	if err != nil {
		return err
	}
	if b.Payload.Revision != revision {
		return errors.New("candidate differs from frozen source revision")
	}
	if err = SameBaseScenario(a, b); err != nil {
		return err
	}
	media, err := ReadMedia(candidate, b)
	if err != nil {
		return err
	}
	if media.CompressionMode != "" {
		return errors.New("production qualification refuses fast development media")
	}
	sum, err := nativebuild.HashFile(c.BaselineDisk)
	if err != nil || sum != c.BaselineDiskSHA256 {
		return errors.New("populated baseline disk differs from operator admission")
	}
	if err = nativebuild.FreshDirectory(custody); err != nil {
		return err
	}
	if err = nativebuild.FreshDirectory(c.Work); err != nil {
		return err
	}
	u, err := user.Lookup("soda-qualifier")
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	if err = os.Chown(c.Work, uid, gid); err != nil {
		return err
	}
	input := filepath.Join(custody, "input")
	if err = os.Mkdir(input, 0755); err != nil {
		return err
	}
	frozen := filepath.Join(input, "candidate")
	if err = os.Mkdir(frozen, 0755); err != nil {
		return err
	}
	files := map[string]string{}
	for k, v := range b.Files {
		files[k] = v
	}
	for _, name := range []string{"media/media.json", "media/" + media.ISO.Path, "media/" + media.Rootfs.Path, "tools/soda-installer"} {
		h, e := nativebuild.HashFile(filepath.Join(candidate, name))
		if e != nil {
			return e
		}
		files[name] = h
	}
	for name, want := range files {
		dst := filepath.Join(frozen, name)
		if err = os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "cp", "--reflink=auto", "--", filepath.Join(candidate, name), dst)
		if err = cmd.Run(); err != nil {
			return err
		}
		if err = os.Chmod(dst, 0444); err != nil {
			return err
		}
		got, e := nativebuild.HashFile(dst)
		if e != nil || got != want {
			return errors.New("snapshot changed during independent admission")
		}
	}
	// Clone the already populated fixture; never write the baseline or seed.
	for source, name := range map[string]string{c.BaselineDisk: "a.qcow2", c.BaselineVariables: "a-vars.fd", c.BaselineKey: "a-key", c.BaselineKnownHosts: "a-known-hosts", c.BaselineState: "a-state.json", c.BaselinePassword: "a-password"} {
		dst := filepath.Join(c.Work, name)
		if err = exec.CommandContext(ctx, "cp", "--reflink=auto", "--", source, dst).Run(); err != nil {
			return err
		}
		if err = os.Chown(dst, uid, gid); err != nil {
			return err
		}
		if err = os.Chmod(dst, 0600); err != nil {
			return err
		}
	}
	cloneSum, err := nativebuild.HashFile(filepath.Join(c.Work, "a.qcow2"))
	if err != nil || cloneSum != sum {
		return errors.New("baseline clone verification failed")
	}
	if err = writeNewJSON(filepath.Join(input, "baseline.json"), a); err != nil {
		return err
	}
	if err = os.Chmod(filepath.Join(input, "baseline.json"), 0444); err != nil {
		return err
	}
	runConfig := c
	runConfig.Work = "/run/soda-p9-work"
	runConfig.Baseline = ""
	if err = writeNewJSON(filepath.Join(input, "config.json"), runConfig); err != nil {
		return err
	}
	if err = os.Chmod(filepath.Join(input, "config.json"), 0444); err != nil {
		return err
	}
	// Root custody stays private; its mounted public input subtree must remain
	// traversable by the qualifier even when the controller uses umask 077.
	if err = filepath.WalkDir(input, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return os.Chmod(p, 0755)
		}
		return nil
	}); err != nil {
		return err
	}
	if err = writeNewJSON(filepath.Join(custody, "admission.json"), map[string]any{"driver_sha256": driver, "baseline": a.Candidate, "candidate": b.Candidate, "files": files, "baseline_disk_sha256": sum}); err != nil {
		return err
	}
	w := acceptance.Worker{Name: "soda-qualify-" + filepath.Base(c.Work), User: "soda-qualifier", Executable: c.Executable, Directory: "/run/soda-p9-work", ReadOnly: []string{input + ":/run/soda-p9-input"}, Writable: []string{c.Work + ":/run/soda-p9-work"}, Environment: []string{"HOME=/run/soda-p9-work", "XDG_RUNTIME_DIR=/run/soda-p9-work/runtime", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}, Arguments: []string{"--worker-qualify", "--qualification-config", "/run/soda-p9-input/config.json"}}
	if err = w.Run(ctx, log, log); err != nil {
		return err
	}
	// Keep protected evidence copies, not a pointer to mutable worker assertions.
	for _, name := range []string{"evidence", "install-evidence", "update-evidence"} {
		if err = exec.CommandContext(ctx, "cp", "-a", "--reflink=auto", filepath.Join(c.Work, name), filepath.Join(custody, name)).Run(); err != nil {
			return err
		}
	}
	if err = filepath.WalkDir(custody, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("qualification evidence symlink refused")
		}
		return os.Chown(p, 0, 0)
	}); err != nil {
		return err
	}
	for name, want := range files {
		got, e := nativebuild.HashFile(filepath.Join(frozen, name))
		if e != nil || got != want {
			return errors.New("qualification snapshot changed")
		}
		got, e = nativebuild.HashFile(filepath.Join(candidate, name))
		if e != nil || got != want {
			return errors.New("original qualified input changed")
		}
	}
	if err = Unchanged(candidate, b); err != nil {
		return err
	}
	got, err := nativebuild.HashFile(c.BaselineDisk)
	if err != nil || got != sum {
		return errors.New("baseline disk was modified")
	}
	var result Receipt
	if err = rd.ReadJSON(filepath.Join(custody, "evidence/qualification.json"), &result); err != nil {
		return err
	}
	if err = result.Validate(a, b, media.HostManifest); err != nil {
		return err
	}
	if err = writeNewJSON(filepath.Join(custody, "qualified.json"), map[string]any{"scope": "native-install-upgrade-recovery", "driver_sha256": driver, "candidate": b.Candidate, "receipt": result, "final_signing": "not connected"}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(log, "P9 qualified evidence:", filepath.Join(custody, "qualified.json"))
	return err
}
