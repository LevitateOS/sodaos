// Package host is the privileged Unix client and thin executor facade over
// host/project, host/terminal and host/tailnet. The daemon server (mux,
// admission, dispatch, config load) is the Rust `soda-host` binary; this
// package keeps the Go Unix client surface and the executor implementations
// until their own cutovers. It is not the SQLite owner, browser OAuth
// surface or build/release controller.
package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	projectexec "github.com/levitateos/sodaos/internal/host/project"
)

type Config struct {
	MuseSHA256          string `json:"muse_sha256"`
	MuseVersion         string `json:"muse_version"`
	MuseSocket          string `json:"muse_socket"`
	IdentitySocket      string `json:"identity_socket"`
	CodexHarness        string `json:"codex_harness"`
	CodexHarnessSHA256  string `json:"codex_harness_sha256"`
	CodexHarnessVersion string `json:"codex_harness_version,omitempty"`
	TailnetManagement   bool   `json:"tailnet_management,omitempty"`
	TailnetImage        string `json:"tailnet_image,omitempty"`
	Image               string `json:"image"`
	Network             string `json:"network"`
	Subnet              string `json:"subnet"`
	Bridge              string `json:"bridge"`
}

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}
type Native struct{}

func (Native) HostNative() {}

func (n Native) Run(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
	return n.RunReader(ctx, bytes.NewReader(in), command, args...)
}

func (Native) RunReader(ctx context.Context, in io.Reader, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = in
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", command, err, stderr.String())
	}
	return out, nil
}

// projectRuntime maps daemon network/image config onto the privileged executor.
func projectRuntime(exec Executor, c Config) *projectexec.Runtime {
	return &projectexec.Runtime{
		Exec: exec,
		Config: projectexec.Config{
			MuseSocket: c.MuseSocket,
			Image:      c.Image,
			Network:    c.Network,
			Subnet:     c.Subnet,
			Bridge:     c.Bridge,
		},
	}
}
