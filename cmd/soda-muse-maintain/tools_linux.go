//go:build linux

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/levitateos/sodaos/internal/identity/muse"
	"golang.org/x/sys/unix"
)

type tool struct {
	name string
	file *os.File
	size int64
}

func closeTools(sources []tool) {
	for _, source := range sources {
		_ = source.file.Close()
	}
}

func loadTools(root, digest, version string) ([]tool, error) {
	if version != muse.Version {
		return nil, errors.New("muse maintenance version differs from pinned release")
	}
	var sources []tool
	for _, name := range []string{"muse", "soda-identity-compose", "muse-native"} {
		source, err := openTool(filepath.Join(root, name), name)
		if err != nil {
			closeTools(sources)
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := verifyNative(sources[2].file, digest); err != nil {
		closeTools(sources)
		return nil, err
	}
	return sources, nil
}

func openTool(path, name string) (tool, error) {
	var source tool
	info, err := os.Lstat(path)
	if err != nil || !trustedTool(info) {
		return source, errors.New("public tool source must be a regular root-owned executable")
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return source, err
	}
	file := os.NewFile(uintptr(fd), name)
	info, err = file.Stat()
	if err != nil || !trustedTool(info) {
		_ = file.Close()
		return source, errors.New("public tool source changed")
	}
	return tool{name: name, file: file, size: info.Size()}, nil
}

func trustedTool(info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 || info.Mode().Perm()&0o111 == 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

func verifyNative(file *os.File, digest string) error {
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return errors.New("muse native digest mismatch")
	}
	_, err := file.Seek(0, io.SeekStart)
	return err
}
