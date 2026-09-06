package acceptance

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

// Only a verified allowlisted bundle is streamed, never a checkout/.artifacts
// tree. The remote creates an exclusive private directory before extraction.
// This delivers bytes; it does not invoke the installer or activate services.
func (r Remote) TransferBundle(ctx context.Context, e *Evidence, source, dest, arch, revision, target string) (Result, error) {
	inv, err := nativebuild.Verify(source, arch, revision)
	if err != nil {
		return Result{Err: err}, nil
	}
	if !filepath.IsAbs(dest) || filepath.Base(dest) != arch || strings.ContainsAny(dest, "\r\n") {
		return Result{Err: errors.New("absolute new remote ARCH directory required")}, nil
	}
	manifestHash, err := nativebuild.HashFile(filepath.Join(source, "build-info.json"))
	if err != nil {
		return Result{Err: err}, nil
	}
	script := `import hashlib,json,os,pathlib,platform,subprocess,sys
if platform.system()!='Linux' or platform.machine()!=sys.argv[2] or platform.node()!=sys.argv[5]: raise SystemExit('native transfer target/platform mismatch')
p=pathlib.Path(sys.argv[1])
if not p.is_absolute() or p.parent.resolve()!=p.parent: raise SystemExit('real absolute destination required')
p.mkdir(mode=0o700)
# stdin contains only the sender's validated allowlist, not arbitrary tar input.
subprocess.run(['tar','--extract','--file=-','--directory='+str(p),'--no-same-owner','--no-overwrite-dir','--same-permissions'],check=True)
manifest=(p/'build-info.json').read_bytes()
if hashlib.sha256(manifest).hexdigest()!=sys.argv[4]: raise SystemExit('manifest transfer mismatch')
expected=json.loads(manifest)['Files']['tools/soda-artifacts']['sha256']
if hashlib.sha256((p/'tools/soda-artifacts').read_bytes()).hexdigest()!=expected: raise SystemExit('verifier transfer mismatch')
subprocess.run([str(p/'tools/soda-artifacts'),'verify','--source',str(p),'--arch',sys.argv[2],'--revision',sys.argv[3]],check=True)
`
	reader, writer := io.Pipe()
	c, err := r.Command([]string{"python3", "-c", script, dest, arch, revision, manifestHash, target}, reader)
	if err != nil {
		reader.Close()
		writer.Close()
		return Result{Err: err}, nil
	}
	finished := make(chan error, 1)
	go func() { err := streamBundle(writer, source, inv); _ = writer.CloseWithError(err); finished <- err }()
	result, evidenceErr := Execute(ctx, e, "transfer", c)
	_ = reader.Close()
	result.Err = errors.Join(result.Err, <-finished)
	return result, evidenceErr
}
func streamBundle(w io.Writer, source string, inv nativebuild.Inventory) error {
	tw := tar.NewWriter(w)
	root, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer root.Close()
	names := make([]string, 0, len(inv.Files)+2)
	for name := range inv.Files {
		names = append(names, name)
	}
	names = append(names, "build-info.json", "SHA256SUMS")
	sort.Strings(names)
	for _, name := range names {
		st, err := root.Lstat(name)
		if err != nil {
			return err
		}
		target := ""
		if entry, ok := inv.Files[name]; ok {
			if entry.Directory != st.IsDir() || entry.Mode != uint32(st.Mode().Perm()) {
				return errors.New("payload type/mode changed during transfer")
			}
			target = entry.Link
			if target != "" {
				actual, err := root.Readlink(name)
				if err != nil || actual != target {
					return errors.New("payload link changed during transfer")
				}
			} else if !entry.Directory && !st.Mode().IsRegular() {
				return errors.New("payload became a special file")
			}
		} else if !st.Mode().IsRegular() {
			return errors.New("non-regular transfer metadata")
		}
		h, err := tar.FileInfoHeader(st, target)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(name)
		h.Uid = 0
		h.Gid = 0
		h.Uname = ""
		h.Gname = ""
		if err = tw.WriteHeader(h); err != nil {
			return err
		}
		if st.Mode().IsRegular() {
			f, err := root.Open(name)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, f)
			err = errors.Join(err, f.Close())
			if err != nil {
				return err
			}
		}
	}
	return tw.Close()
}
