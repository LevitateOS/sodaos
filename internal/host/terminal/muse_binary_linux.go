//go:build linux

package terminal

import (
	"context"
	"encoding/binary"
	"runtime"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
)

func (m *MuseRuntime) verifyGuestBinary(ctx context.Context, container, child string) error {
	if !containerID.MatchString(m.BinarySHA256) {
		return identity.ErrDenied
	}
	prefix := []string{}
	if child != "" {
		prefix = []string{"/usr/bin/podman", "--remote=false", "exec", child}
	}
	hash, err := m.guest(ctx, container, nil, append(prefix, "/usr/bin/sha256sum", "/usr/local/libexec/soda/muse")...)
	if err != nil || (len(strings.Fields(string(hash))) != 2 || strings.Fields(string(hash))[0] != m.BinarySHA256) {
		return identity.ErrDenied
	}
	header, err := m.guest(ctx, container, nil, append(prefix, "/usr/bin/head", "--bytes=64", "/usr/local/libexec/soda/muse")...)
	if err != nil {
		return identity.ErrDenied
	}
	if err = museELF(header, runtime.GOARCH); err != nil {
		return err
	}
	if m.BinaryVersion == "" {
		return identity.ErrDenied
	}
	_, err = m.guest(ctx, container, nil, append(prefix, "/usr/local/bin/muse", "--soda-check", m.BinaryVersion)...)
	return err
}

func museELF(header []byte, arch string) error {
	if len(header) != 64 || string(header[:4]) != "\x7fELF" || header[4] != 2 || header[5] != 1 {
		return identity.ErrDenied
	}
	machine := binary.LittleEndian.Uint16(header[18:20])
	if arch == "amd64" && machine == 62 {
		return nil
	}
	if arch == "arm64" && machine == 183 {
		return nil
	}
	return identity.ErrDenied
}
