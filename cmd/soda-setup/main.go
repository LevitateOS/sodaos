// soda-setup is an operator-invoked one-time dashboard OAuth setup, not a daemon.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("run through the operator's native root access")
	}
	external := flag.String("forgejo-url", "", "Forgejo HTTPS origin")
	internal := flag.String("forgejo-internal-url", "http://127.0.0.1:3000", "native Forgejo origin")
	tokenPath := flag.String("token-file", "", "operator Forgejo access token file")
	out := flag.String("out", "/etc/soda/dashboard.json", "new dashboard configuration")
	flag.Parse()
	return setup(*external, *internal, *tokenPath, *out)
}

func writeSetupSecret(path, value string) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if e != nil {
		return fmt.Errorf("OAuth application created, but credential write failed; inspect Forgejo applications before retrying: %w", e)
	}
	_, e = f.WriteString(value + "\n")
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}

func writeSetupConfig(out string, c config.Config) error {
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c)
}

func admitSetupPaths(external, internal, out string) error {
	for _, u := range []string{external, internal} {
		if err := config.BaseURL(u); err != nil {
			return err
		}
	}
	if !strings.HasPrefix(external, "https://") {
		return fmt.Errorf("browser origins must use HTTPS")
	}
	if !filepath.IsAbs(out) {
		return fmt.Errorf("out must be absolute")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		return fmt.Errorf("configuration already exists or cannot be inspected; refusing overwrite")
	}
	return nil
}

func setupOAuthApp(external, internal, token string) (forgejo.User, forgejo.Application, error) {
	var zero forgejo.User
	var app forgejo.Application
	client := forgejo.New(internal)
	ctx := context.Background()
	u, err := client.Current(ctx, token)
	if err != nil {
		return zero, app, err
	}
	if !u.Admin {
		return zero, app, fmt.Errorf("forgejo operator token is not a site administrator")
	}
	a, err := client.Application(ctx, token, (config.Config{ForgejoURL: strings.TrimRight(external, "/")}).OAuthCallbackURL())
	if err != nil {
		return zero, app, err
	}
	if a.ClientID == "" || a.Secret == "" {
		return zero, app, fmt.Errorf("forgejo returned an incomplete OAuth application")
	}
	return u, a, nil
}

func setup(external, internal, tokenPath, out string) error {
	if err := admitSetupPaths(external, internal, out); err != nil {
		return err
	}
	token, err := config.Secret(tokenPath)
	if err != nil {
		return err
	}
	u, a, err := setupOAuthApp(external, internal, token)
	if err != nil {
		return err
	}
	dir := filepath.Dir(out)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	secretPath := filepath.Join(dir, "oauth-secret")
	keyPath := filepath.Join(dir, "grant-key")
	key := make([]byte, 32)
	rand.Read(key)
	for path, value := range map[string]string{secretPath: a.Secret, keyPath: base64.StdEncoding.EncodeToString(key)} {
		if err := writeSetupSecret(path, value); err != nil {
			return err
		}
	}
	c := config.Config{Listen: "127.0.0.1:8080", ForgejoURL: strings.TrimRight(external, "/"), ForgejoInternalURL: strings.TrimRight(internal, "/"), Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", OAuthClientID: a.ClientID, OAuthSecretFile: secretPath, GrantKeyFile: keyPath, OperatorID: u.ID}
	if err = writeSetupConfig(out, c); err != nil {
		return err
	}
	fmt.Println("Dashboard configuration created. Set native soda service ownership before enabling the dashboard. No host privilege was granted to a Forgejo user.")
	return nil
}
