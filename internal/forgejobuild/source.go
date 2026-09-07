// Package forgejobuild prepares the locked Forgejo source for the core build.
// It does not compile, install, start Forgejo, or qualify a provider interface.
package forgejobuild

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxArchive = 128 << 20
	maxSource  = 256 << 20
	maxEntries = 20000
	maxPatch   = 8 << 20
)

var (
	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	patchPattern   = regexp.MustCompile(`^[0-9]{4}-[a-z0-9-]+\.patch$`)
)

type Patch struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type Lock struct {
	Schema        int     `json:"schema"`
	Version       string  `json:"version"`
	Commit        string  `json:"commit"`
	ArchiveSHA256 string  `json:"archive_sha256"`
	Patches       []Patch `json:"patches"`
}

// Prepare consumes a local archive, or retrieves the locked public commit when
// archivePath is empty. out must be a fresh directory under a real, non-writable-
// by-others parent. Failed attempts remain, without a prepared.json receipt.
func Prepare(ctx context.Context, lockPath, archivePath, out string) error {
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	return prepare(ctx, lockPath, archivePath, out, client)
}

func readRegular(name string, limit int64) ([]byte, error) {
	before, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("regular input file required")
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, after) {
		return nil, errors.New("input changed while opening")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("input exceeds size limit")
	}
	return b, nil
}

func hash(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func parseLock(b []byte) (Lock, error) {
	var lock Lock
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&lock); err != nil {
		return lock, errors.New("invalid Forgejo source lock")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return lock, errors.New("source lock must contain one object")
	}
	if lock.Schema != 1 || !versionPattern.MatchString(lock.Version) || !commitPattern.MatchString(lock.Commit) || !digestPattern.MatchString(lock.ArchiveSHA256) || lock.Patches == nil || len(lock.Patches) > 32 {
		return lock, errors.New("invalid Forgejo source lock fields")
	}
	for i, patch := range lock.Patches {
		if !patchPattern.MatchString(patch.File) || !strings.HasPrefix(patch.File, fmt.Sprintf("%04d-", i+1)) || !digestPattern.MatchString(patch.SHA256) {
			return lock, errors.New("invalid or unordered Forgejo patch series")
		}
	}
	return lock, nil
}

func prepare(ctx context.Context, lockPath, archivePath, out string, client *http.Client) error {
	lockBytes, err := readRegular(lockPath, 64<<10)
	if err != nil {
		return err
	}
	lock, err := parseLock(lockBytes)
	if err != nil {
		return err
	}
	// Snapshot verified patches before creating output or invoking an external tool.
	patchBytes := make([][]byte, len(lock.Patches))
	total := 0
	for i, patch := range lock.Patches {
		b, err := readRegular(filepath.Join(filepath.Dir(lockPath), "patches", patch.File), maxPatch)
		if err != nil {
			return err
		}
		total += len(b)
		if total > 32<<20 {
			return errors.New("patch series exceeds size limit")
		}
		if hash(b) != patch.SHA256 {
			return fmt.Errorf("patch digest mismatch: %s", patch.File)
		}
		patchBytes[i] = b
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	parent := filepath.Dir(out)
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	st, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if realParent != parent || !st.IsDir() || st.Mode().Perm()&0022 != 0 {
		return errors.New("output parent must be a real directory not writable by others")
	}
	if err = os.Mkdir(out, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(out)
	if err != nil {
		return err
	}
	defer root.Close()
	if err = root.WriteFile("source.lock.json", lockBytes, 0600); err != nil {
		return err
	}

	var archive []byte
	if archivePath != "" {
		archive, err = readRegular(archivePath, maxArchive)
	} else {
		url := "https://codeberg.org/forgejo/forgejo/archive/" + lock.Commit + ".tar.gz"
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			req.Header.Set("User-Agent", "SodaOS-Forgejo-source")
			var res *http.Response
			res, err = client.Do(req)
			if err != nil {
				return errors.New("Forgejo source download failed")
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("Forgejo source download HTTP %d", res.StatusCode)
			}
			archive, err = io.ReadAll(io.LimitReader(res.Body, maxArchive+1))
		}
	}
	if err != nil {
		return err
	}
	if len(archive) > maxArchive {
		return errors.New("source archive exceeds size limit")
	}
	// A private snapshot binds the bytes checked here to the bytes extracted below.
	if err = root.WriteFile("upstream.tar.gz", archive, 0600); err != nil {
		return err
	}
	if hash(archive) != lock.ArchiveSHA256 {
		return errors.New("Forgejo source archive digest mismatch")
	}
	if err = root.Mkdir("source", 0700); err != nil {
		return err
	}
	source, err := root.OpenRoot("source")
	if err != nil {
		return err
	}
	defer source.Close()
	if err = extract(ctx, source, archive, lock.Commit); err != nil {
		return err
	}
	if err = root.Mkdir("patches", 0700); err != nil {
		return err
	}
	for i, patch := range lock.Patches {
		if err = root.WriteFile("patches/"+patch.File, patchBytes[i], 0600); err != nil {
			return err
		}
		// No repository/index, global/system Git config, inherited GIT_* state,
		// hooks, fuzz, shell expansion or --unsafe-paths. Git owns patch semantics.
		for _, check := range []bool{true, false} {
			args := []string{"apply", "--no-index", "--whitespace=error-all"}
			if check {
				args = append(args, "--check")
			}
			args = append(args, "--", filepath.Join(out, "patches", patch.File))
			patchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			cmd := exec.CommandContext(patchCtx, "git", args...)
			cmd.Dir = filepath.Join(out, "source")
			cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + out, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CEILING_DIRECTORIES=" + out}
			err = cmd.Run() // do not echo patch payloads or arbitrary tool diagnostics
			cancel()
			if err != nil {
				return fmt.Errorf("Forgejo patch failed: %s", patch.File)
			}
		}
	}
	// Check all output types/paths again after git apply, and bind the source tree.
	treeHash, err := sourceHash(source)
	if err != nil {
		return err
	}
	for _, required := range []string{"LICENSE", "go.mod", "go.sum", "package.json", "package-lock.json", "Makefile", "Dockerfile", "docker/root/usr/bin/entrypoint"} {
		st, err := source.Lstat(required)
		if err != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("missing regular Forgejo build input: %s", required)
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	receipt, err := json.MarshalIndent(struct {
		LockSHA256   string `json:"lock_sha256"`
		SourceSHA256 string `json:"source_sha256"`
	}{hash(lockBytes), treeHash}, "", "  ")
	if err != nil {
		return err
	}
	return root.WriteFile("prepared.json", append(receipt, '\n'), 0600)
}

func validPath(name string) bool {
	if name == "" || name == "." || !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00") || path.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}

func validTarget(name, target string) bool {
	return target != "" && utf8.ValidString(target) && !path.IsAbs(target) && !strings.ContainsAny(target, "\\\x00") && validPath(path.Clean(path.Join(path.Dir(name), target)))
}

func extract(ctx context.Context, root *os.Root, archive []byte, commit string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gz.Close()
	// Bound headers and padding as well as declared file payloads.
	limited := &io.LimitedReader{R: gz, N: maxSource + 1}
	tarReader := tar.NewReader(limited)
	seen := map[string]bool{}
	links := map[string]string{}
	count := 0
	seenCommit, seenRoot := false, false
	for {
		if err = ctx.Err(); err != nil {
			return err
		}
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		count++
		if count > maxEntries || limited.N <= 0 {
			return errors.New("source archive expansion limit exceeded")
		}
		// git archive emits a global PAX comment identifying its source commit.
		// It is metadata, not a file to extract or an independent trust signature.
		if h.Typeflag == tar.TypeXGlobalHeader {
			if count != 1 || seenCommit || len(h.PAXRecords) != 1 || h.PAXRecords["comment"] != commit {
				return errors.New("unexpected Git archive commit metadata")
			}
			seenCommit = true
			continue
		}
		if !seenCommit {
			return errors.New("missing Git archive commit metadata")
		}
		name := strings.TrimSuffix(h.Name, "/")
		if name == "forgejo" && h.Typeflag == tar.TypeDir {
			if seenRoot || h.Mode&07000 != 0 {
				return errors.New("unsafe or duplicate archive root")
			}
			seenRoot = true
			continue
		}
		if !strings.HasPrefix(name, "forgejo/") {
			return errors.New("unexpected archive root")
		}
		name = strings.TrimPrefix(name, "forgejo/")
		if !validPath(name) || seen[name] || h.Mode&07000 != 0 || h.Size < 0 || h.Size > maxSource {
			return errors.New("unsafe or duplicate archive entry")
		}
		seen[name] = true
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if _, ok := links[parent]; ok {
				return errors.New("archive entry below symlink")
			}
		}
		if err = root.MkdirAll(path.Dir(name), 0755); err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			err = root.MkdirAll(name, 0755)
		case tar.TypeReg:
			var f *os.File
			mode := fs.FileMode(0644)
			if h.Mode&0111 != 0 {
				mode = 0755
			}
			f, err = root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tarReader)
			err = errors.Join(err, f.Close())
		case tar.TypeSymlink:
			if !validTarget(name, h.Linkname) {
				return errors.New("unsafe archive symlink")
			}
			links[name] = h.Linkname
		default:
			return errors.New("unsupported archive entry type")
		}
		if err != nil {
			return err
		}
	}
	// Consume the gzip footer: a truncated/checksum-invalid stream must not pass
	// merely because tar's end marker was encountered first.
	if _, err = io.Copy(io.Discard, limited); err != nil {
		return err
	}
	if limited.N <= 0 {
		return errors.New("source archive expansion limit exceeded")
	}
	for name, target := range links {
		if err = root.Symlink(target, name); err != nil {
			return err
		}
	}
	for name := range links {
		if _, err = root.Stat(name); err != nil {
			return errors.New("dangling or cyclic archive symlink")
		}
	}
	return nil
}

func sourceHash(root *os.Root) (string, error) {
	names := []string{}
	if err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name != "." {
			names = append(names, name)
		}
		if len(names) > maxEntries {
			return errors.New("patched source entry limit exceeded")
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(names)
	h := sha256.New()
	e := json.NewEncoder(h)
	for _, name := range names {
		if !validPath(name) {
			return "", errors.New("unsafe patched source path")
		}
		st, err := root.Lstat(name)
		if err != nil {
			return "", err
		}
		if st.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return "", errors.New("unsafe patched source mode")
		}
		kind, value := "", ""
		switch {
		case st.Mode().IsRegular():
			kind = "file"
			f, err := root.Open(name)
			if err != nil {
				return "", err
			}
			fileHash := sha256.New()
			_, err = io.Copy(fileHash, f)
			err = errors.Join(err, f.Close())
			if err != nil {
				return "", err
			}
			value = hex.EncodeToString(fileHash.Sum(nil))
		case st.IsDir():
			kind = "directory"
		case st.Mode()&os.ModeSymlink != 0:
			kind = "symlink"
			value, err = root.Readlink(name)
			if err != nil || !validTarget(name, value) {
				return "", errors.New("unsafe patched source symlink")
			}
			if _, err = root.Stat(name); err != nil {
				return "", errors.New("dangling patched source symlink")
			}
		default:
			return "", errors.New("unsupported patched source type")
		}
		if err = e.Encode([]any{name, kind, st.Mode().Perm(), value}); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
