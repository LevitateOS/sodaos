package main

import (
	"context"
	"fmt"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/tailnet"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("host operator required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	endpoint, err := tailnet.New(tailnet.Options{}).Endpoint(ctx)
	if err != nil {
		return err
	}
	changed, err := forgejo.UpdateSSHDomain("/etc/soda/forgejo.env", endpoint.Identity)
	if err != nil {
		return err
	}
	live, inspectErr := exec.CommandContext(ctx, "/usr/bin/podman", "inspect", "--format", "{{range .Config.Env}}{{println .}}{{end}}", "soda-forgejo").Output()
	matches := false
	for _, line := range strings.Split(string(live), "\n") {
		if line == "FORGEJO__server__SSH_DOMAIN="+endpoint.Identity {
			matches = true
		}
	}
	if changed || inspectErr != nil || !matches {
		if err = exec.CommandContext(ctx, "/usr/bin/systemctl", "restart", "forgejo.service").Run(); err != nil {
			return fmt.Errorf("native Forgejo configuration saved, restart failed: %w", err)
		}
	}
	fmt.Println("Forgejo SSH address refreshed; configured browser/OAuth origins preserved.")
	return nil
}
