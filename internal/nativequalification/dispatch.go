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
	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

func hashDriver(executable string) (string, error) {
	if err := acceptance.TrustedExecutable(executable); err != nil {
		return "", err
	}
	return nativebuild.HashFile(executable)
}

func admittedDiskHash(path, want string) (string, error) {
	sum, err := nativebuild.HashFile(path)
	if err != nil || sum != want {
		return "", errors.New("populated baseline disk differs from operator admission")
	}
	return sum, nil
}

func admitDispatchArtifacts(c Config, candidate, revision string) (Artifact, Artifact, hostimage.Media, string, error) {
	a, err := ReadArtifact(c.Baseline)
	if err != nil {
		return Artifact{}, Artifact{}, hostimage.Media{}, "", err
	}
	b, err := ReadArtifact(candidate)
	if err != nil {
		return a, Artifact{}, hostimage.Media{}, "", err
	}
	if b.Payload.Revision != revision {
		return a, b, hostimage.Media{}, "", errors.New("candidate differs from frozen source revision")
	}
	if err = SameBaseScenario(a, b); err != nil {
		return a, b, hostimage.Media{}, "", err
	}
	media, err := ReadMedia(candidate, b)
	if err != nil {
		return a, b, media, "", err
	}
	if media.CompressionMode != "" {
		return a, b, media, "", errors.New("production qualification refuses fast development media")
	}
	sum, err := admittedDiskHash(c.BaselineDisk, c.BaselineDiskSHA256)
	return a, b, media, sum, err
}

func qualifierUIDGID() (int, int, error) {
	u, err := user.Lookup("soda-qualifier")
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

func prepareDispatchCustody(c Config, custody string) (input string, uid, gid int, err error) {
	if err = nativebuild.FreshDirectory(custody); err != nil {
		return input, uid, gid, err
	}
	if err = nativebuild.FreshDirectory(c.Work); err != nil {
		return input, uid, gid, err
	}
	uid, gid, err = qualifierUIDGID()
	if err != nil {
		return input, uid, gid, err
	}
	if err = os.Chown(c.Work, uid, gid); err != nil {
		return input, uid, gid, err
	}
	input = filepath.Join(custody, "input")
	if err = os.Mkdir(input, 0o755); err != nil {
		return input, uid, gid, err
	}
	err = os.Mkdir(filepath.Join(input, "candidate"), 0o755)
	return input, uid, gid, err
}

func hashCandidateMedia(candidate string, media hostimage.Media, files map[string]string) error {
	for _, name := range []string{"media/media.json", "media/" + media.ISO.Path, "media/" + media.Rootfs.Path, "tools/soda-installer"} {
		h, e := nativebuild.HashFile(filepath.Join(candidate, name))
		if e != nil {
			return e
		}
		files[name] = h
	}
	return nil
}

func freezeCandidateFiles(ctx context.Context, candidate, frozen string, files map[string]string) error {
	for name, want := range files {
		dst := filepath.Join(frozen, name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "cp", "--reflink=auto", "--", filepath.Join(candidate, name), dst)
		if err := cmd.Run(); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o444); err != nil {
			return err
		}
		got, e := nativebuild.HashFile(dst)
		if e != nil || got != want {
			return errors.New("snapshot changed during independent admission")
		}
	}
	return nil
}

func cloneBaselineIntoWork(ctx context.Context, c Config, uid, gid int) error {
	// Clone the already populated fixture; never write the baseline or seed.
	for source, name := range map[string]string{c.BaselineDisk: "a.qcow2", c.BaselineVariables: "a-vars.fd", c.BaselineKey: "a-key", c.BaselineKnownHosts: "a-known-hosts", c.BaselineState: "a-state.json", c.BaselinePassword: "a-password"} {
		dst := filepath.Join(c.Work, name)
		if err := exec.CommandContext(ctx, "cp", "--reflink=auto", "--", source, dst).Run(); err != nil {
			return err
		}
		if err := os.Chown(dst, uid, gid); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func verifyCloneHash(work, want string) error {
	cloneSum, err := nativebuild.HashFile(filepath.Join(work, "a.qcow2"))
	if err != nil || cloneSum != want {
		return errors.New("baseline clone verification failed")
	}
	return nil
}

func chmodPublicInputDir(p string, d os.DirEntry, e error) error {
	if e != nil {
		return e
	}
	if d.IsDir() {
		return os.Chmod(p, 0o755)
	}
	return nil
}

func writeDispatchInputs(input string, c Config, a Artifact) error {
	if err := writeNewJSON(filepath.Join(input, "baseline.json"), a); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(input, "baseline.json"), 0o444); err != nil {
		return err
	}
	runConfig := c
	runConfig.Work = "/run/soda-p9-work"
	runConfig.Baseline = ""
	if err := writeNewJSON(filepath.Join(input, "config.json"), runConfig); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(input, "config.json"), 0o444); err != nil {
		return err
	}
	// Root custody stays private; its mounted public input subtree must remain
	// traversable by the qualifier even when the controller uses umask 077.
	return filepath.WalkDir(input, chmodPublicInputDir)
}

func snapshotDispatchInputs(ctx context.Context, c Config, candidate string, media hostimage.Media, a, b Artifact, uid, gid int, input, sum string) (map[string]string, error) {
	frozen := filepath.Join(input, "candidate")
	files := map[string]string{}
	for k, v := range b.Files {
		files[k] = v
	}
	if err := hashCandidateMedia(candidate, media, files); err != nil {
		return nil, err
	}
	if err := freezeCandidateFiles(ctx, candidate, frozen, files); err != nil {
		return nil, err
	}
	if err := cloneBaselineIntoWork(ctx, c, uid, gid); err != nil {
		return nil, err
	}
	if err := verifyCloneHash(c.Work, sum); err != nil {
		return nil, err
	}
	if err := writeDispatchInputs(input, c, a); err != nil {
		return nil, err
	}
	return files, nil
}

func runQualifier(ctx context.Context, c Config, input string, log io.Writer) error {
	w := acceptance.Worker{Name: "soda-qualify-" + filepath.Base(c.Work), User: "soda-qualifier", Executable: c.Executable, Directory: "/run/soda-p9-work", ReadOnly: []string{input + ":/run/soda-p9-input"}, Writable: []string{c.Work + ":/run/soda-p9-work"}, Environment: []string{"HOME=/run/soda-p9-work", "XDG_RUNTIME_DIR=/run/soda-p9-work/runtime", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}, Arguments: []string{"--worker-qualify", "--qualification-config", "/run/soda-p9-input/config.json"}}
	return w.Run(ctx, log, log)
}

func copyProtectedEvidence(ctx context.Context, work, custody string) error {
	for _, name := range []string{"evidence", "install-evidence", "update-evidence"} {
		if err := exec.CommandContext(ctx, "cp", "-a", "--reflink=auto", filepath.Join(work, name), filepath.Join(custody, name)).Run(); err != nil {
			return err
		}
	}
	return nil
}

func refuseEvidenceSymlink(p string, d os.DirEntry, e error) error {
	if e != nil {
		return e
	}
	if d.Type()&os.ModeSymlink != 0 {
		return errors.New("qualification evidence symlink refused")
	}
	return os.Chown(p, 0, 0)
}

func verifyFrozenInputs(candidate, frozen string, files map[string]string, b Artifact, disk, sum string) error {
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
	if err := Unchanged(candidate, b); err != nil {
		return err
	}
	got, err := nativebuild.HashFile(disk)
	if err != nil || got != sum {
		return errors.New("baseline disk was modified")
	}
	return nil
}

func sealQualifiedReceipt(custody, driver string, a, b Artifact, media hostimage.Media, result Receipt) error {
	if err := result.Validate(a, b, media.HostManifest); err != nil {
		return err
	}
	return writeNewJSON(filepath.Join(custody, "qualified.json"), map[string]any{"scope": "native-install-upgrade-recovery", "driver_sha256": driver, "candidate": b.Candidate, "receipt": result, "final_signing": "not connected"})
}

func finishDispatch(ctx context.Context, c Config, candidate, frozen, custody, driver, sum string, a, b Artifact, media hostimage.Media, files map[string]string, log io.Writer) error {
	if err := runQualifier(ctx, c, filepath.Dir(frozen), log); err != nil {
		return err
	}
	if err := copyProtectedEvidence(ctx, c.Work, custody); err != nil {
		return err
	}
	if err := filepath.WalkDir(custody, refuseEvidenceSymlink); err != nil {
		return err
	}
	if err := verifyFrozenInputs(candidate, frozen, files, b, c.BaselineDisk, sum); err != nil {
		return err
	}
	var result Receipt
	if err := rd.ReadJSON(filepath.Join(custody, "evidence/qualification.json"), &result); err != nil {
		return err
	}
	if err := sealQualifiedReceipt(custody, driver, a, b, media, result); err != nil {
		return err
	}
	_, err := fmt.Fprintln(log, "P9 qualified evidence:", filepath.Join(custody, "qualified.json"))
	return err
}

// Dispatch freezes independently checked input bytes in root custody. The builder
// cannot write this snapshot, the qualifier work, or the final controller receipt.
func Dispatch(ctx context.Context, c Config, candidate, revision, custody string, log io.Writer) error {
	if os.Geteuid() != 0 {
		return errors.New("protected qualification dispatcher required")
	}
	driver, err := hashDriver(c.Executable)
	if err != nil {
		return err
	}
	a, b, media, sum, err := admitDispatchArtifacts(c, candidate, revision)
	if err != nil {
		return err
	}
	input, uid, gid, err := prepareDispatchCustody(c, custody)
	if err != nil {
		return err
	}
	files, err := snapshotDispatchInputs(ctx, c, candidate, media, a, b, uid, gid, input, sum)
	if err != nil {
		return err
	}
	if err = writeNewJSON(filepath.Join(custody, "admission.json"), map[string]any{"driver_sha256": driver, "baseline": a.Candidate, "candidate": b.Candidate, "files": files, "baseline_disk_sha256": sum}); err != nil {
		return err
	}
	return finishDispatch(ctx, c, candidate, filepath.Join(input, "candidate"), custody, driver, sum, a, b, media, files, log)
}
