//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func checkRuntime(version string) error {
	private, err := os.MkdirTemp("", "soda-muse-check-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(private) }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "/usr/local/libexec/soda/muse", "--version")
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "HOME=" + private, "XDG_CONFIG_HOME=" + private + "/config", "XDG_STATE_HOME=" + private + "/state", "XDG_CACHE_HOME=" + private + "/cache", "XDG_DATA_HOME=" + private + "/data", "TMPDIR=" + private, "TBH_CREDENTIAL_BACKEND=file"}
	body, err := command.Output()
	if err != nil {
		return fmt.Errorf("muse native runtime prerequisite failed: %w", err)
	}
	if strings.TrimSpace(string(body)) != "Muse Code 1.4.0 ("+version+")" {
		return fmt.Errorf("muse version differs from pinned %s", version)
	}
	return nil
}
