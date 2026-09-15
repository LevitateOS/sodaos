package qualify

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/release/build"
)

// Headless fixture configuration uses Forgejo's native environment/CLI and Soda's
// existing setup/activation commands. It is not browser-onboarding qualification.
func configureFixtureForgejo(ctx context.Context) error {
	for _, path := range []string{"/etc/soda/dashboard.json", "/etc/soda/activated"} {
		if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
			return errors.New("fresh unconfigured fixture required")
		}
	}
	path := "/etc/soda/forgejo.env"
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains(string(b), "FORGEJO__security__INSTALL_LOCK=") {
		return errors.New("existing fixture installation policy requires inspection")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if err = os.WriteFile(path, append(b, []byte("\nFORGEJO__security__INSTALL_LOCK=true\n")...), 0o600); err != nil {
		return err
	}
	if err = exec.CommandContext(ctx, "systemctl", "restart", "forgejo.service").Run(); err != nil {
		return err
	}
	return waitFixtureForgejo(ctx)
}

func waitFixtureForgejo(ctx context.Context) error {
	client := http.Client{Timeout: 3 * time.Second}
	ready, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		req, e := http.NewRequestWithContext(ready, "GET", "http://127.0.0.1:3000/api/v1/version", nil)
		if e != nil {
			return e
		}
		res, e := client.Do(req)
		if e == nil {
			res.Body.Close()
			if res.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ready.Done():
			return ready.Err()
		case <-time.After(time.Second):
		}
	}
}

func configureFixtureSoda(ctx context.Context) error {
	token, err := exec.CommandContext(ctx, "podman", "exec", "--user", "git", "soda-forgejo", "forgejo", "admin", "user", "generate-access-token", "--username", fixtureLogin, "--token-name", "p9-local-setup", "--scopes", "write:user", "--raw").Output()
	if err != nil {
		return errors.New("native local fixture token creation failed")
	}
	path := filepath.Join(fixtureState, "setup-token")
	if err = build.WriteNew(path, []byte(strings.TrimSpace(string(token))), 0o600); err != nil {
		return err
	}
	// Only the local disposable Forgejo is contacted; no external provider, real
	// account, public endpoint or host trust installation participates.
	if err = exec.CommandContext(ctx, "/usr/bin/soda-setup", "--forgejo-url", "https://10.0.2.15", "--token-file", path).Run(); err != nil {
		return errors.New("native Soda fixture setup failed; inspect without re-registering")
	}
	if err = exec.CommandContext(ctx, "/usr/bin/soda-activate", "--bind-ip", "10.0.2.15", "--local-tls").Run(); err != nil {
		return errors.New("native fixture activation failed; do not replay")
	}
	if err = waitFixtureForgejo(ctx); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", "forgejo.service", "soda-dashboard.service", "soda-proxy.service", "soda-host.socket").Run()
}
