package main

import (
	"context"
	"encoding/json"
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
	if changed || domain != endpoint.Identity || !running {
		if err = exec.CommandContext(ctx, "/usr/bin/systemctl", "restart", "forgejo.service").Run(); err != nil {
			return fmt.Errorf("native Forgejo configuration saved, restart failed: %w", err)
		}
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
	exposed := false
	for _, binding := range items[0].HostConfig.PortBindings["22/tcp"] {
		if binding.HostPort == "2222" && (binding.HostIP == ip || binding.HostIP == "0.0.0.0" || binding.HostIP == "") {
			exposed = true
		}
	}
	if !exposed {
		return "", false, fmt.Errorf("Forgejo Git SSH is not bound to Tailnet IP %s:2222; configure the intended private native listener before refreshing its advertised address", ip)
	}
	for _, value := range items[0].Config.Env {
		if strings.HasPrefix(value, "FORGEJO__server__SSH_DOMAIN=") {
			return strings.TrimPrefix(value, "FORGEJO__server__SSH_DOMAIN="), items[0].State.Running, nil
		}
	}
	return "", items[0].State.Running, nil
}
