// Native artifact support; no product policy or release qualification.
package nativebuild

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
)

func OCIArchitecture(arch string) (string, error) {
	switch arch {
	case "x86_64":
		return "amd64", nil
	case "aarch64":
		return "arm64", nil
	}
	return "", errors.New("expected x86_64 or aarch64")
}
func RequireNative(arch string) error {
	a, err := OCIArchitecture(arch)
	if err != nil {
		return err
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != a {
		return errors.New("matching-native Linux required")
	}
	return nil
}
func Digest(s string) bool   { return regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(s) }
func Revision(s string) bool { return regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(s) }
func HashFile(path string) (string, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("regular non-symlink file required")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashAt keeps path resolution confined to the caller's open directory.
func HashAt(root *os.Root, name string) (string, error) {
	st, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("regular non-symlink file required")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(st, actual) {
		return "", errors.New("file changed before hashing")
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func FreshDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("absolute new directory required")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	if parent != filepath.Clean(filepath.Dir(path)) {
		return errors.New("symlinked parent refused")
	}
	return os.Mkdir(path, 0700)
}
func PrivateDestination(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("absolute private output required")
	}
	parent := filepath.Dir(path)
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return err
	}
	st, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if resolved != parent || st.Mode().Perm()&0077 != 0 {
		return errors.New("real private output parent required")
	}
	if _, err = os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return errors.New("output already exists or cannot be inspected")
	}
	return nil
}
func WriteNew(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Close())
}
func ReadJSON(path string, v any) error {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer root.Close()
	_, err = ReadJSONAt(root, filepath.Base(path), v)
	return err
}

// ReadJSONAt returns the digest of the exact bounded bytes it decoded, not a
// later reopening of the filename. The directory remains caller-owned.
func ReadJSONAt(root *os.Root, name string, v any) (string, error) {
	st, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if !st.Mode().IsRegular() || st.Size() > 4<<20 {
		return "", errors.New("bounded regular JSON input required")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(st, actual) {
		return "", errors.New("JSON input changed before reading")
	}
	data, err := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if err != nil {
		return "", err
	}
	if len(data) > 4<<20 {
		return "", errors.New("JSON input exceeds limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(v); err != nil {
		return "", err
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return "", errors.New("trailing JSON data")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
