package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func restartForgejoIfNeeded(ctx context.Context, changed bool, domain, identity string, running bool) error {
	if changed || domain != identity || !running {
		if err := exec.CommandContext(ctx, "/usr/bin/systemctl", "restart", "forgejo.service").Run(); err != nil {
			return fmt.Errorf("native Forgejo configuration saved, restart failed: %w", err)
		}
	}
	return nil
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
	// Do not advertise a Tailnet address while the native port only binds a LAN IP.
	// Raw inspection may contain credentials; never print it or include it in errors.
	live, err := exec.CommandContext(ctx, "/usr/bin/podman", "inspect", "soda-forgejo").Output()
	if err != nil {
		return fmt.Errorf("cannot inspect native Forgejo; inspect its operator service journal")
	}
	domain, running, err := publishedState(live, endpoint.IPv4)
	if err != nil {
		return err
	}
	changed, err := forgejo.UpdateSSHDomain("/etc/soda/forgejo.env", endpoint.Identity)
	if err != nil {
		return err
	}
	if err = restartForgejoIfNeeded(ctx, changed, domain, endpoint.Identity, running); err != nil {
		return err
	}
	fmt.Println("Forgejo SSH address refreshed; configured browser/OAuth origins preserved.")
	return nil
}

func publishedState(data []byte, ip string) (string, bool, error) {
	var items []struct {
		Config     struct{ Env []string }
		State      struct{ Running bool }
		HostConfig struct {
			PortBindings map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string
			}
		}
	}
	if json.Unmarshal(data, &items) != nil || len(items) != 1 {
		return "", false, fmt.Errorf("cannot read native Forgejo network state")
	}
	if !sshBoundToIP(items[0].HostConfig.PortBindings["22/tcp"], ip) {
		return "", false, fmt.Errorf("forgejo Git SSH is not bound to Tailnet IP %s:2222; configure the intended private native listener before refreshing its advertised address", ip)
	}
	domain, running := sshDomainFromEnv(items[0].Config.Env, items[0].State.Running)
	return domain, running, nil
}

func sshBoundToIP(bindings []struct {
	HostIP   string `json:"HostIp"`
	HostPort string
}, ip string,
) bool {
	for _, binding := range bindings {
		if binding.HostPort == "2222" && (binding.HostIP == ip || binding.HostIP == "0.0.0.0" || binding.HostIP == "") {
			return true
		}
	}
	return false
}

func sshDomainFromEnv(env []string, running bool) (string, bool) {
	for _, value := range env {
		if strings.HasPrefix(value, "FORGEJO__server__SSH_DOMAIN=") {
			return strings.TrimPrefix(value, "FORGEJO__server__SSH_DOMAIN="), running
		}
	}
	return "", running
}
