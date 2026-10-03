// soda-setup records the native operator identity and Soda service configuration.
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
		return fmt.Errorf("create credential encryption key: %w", e)
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

func setupOperator(internal, token string) (forgejo.User, error) {
	client := forgejo.New(internal)
	ctx := context.Background()
	u, err := client.Current(ctx, token)
	if err != nil {
		return forgejo.User{}, err
	}
	if !u.Admin {
		return forgejo.User{}, fmt.Errorf("forgejo operator token is not a site administrator")
	}
	return u, nil
}

func revokeBootstrapToken(internal, token string) error {
	client := forgejo.New(internal)
	return client.RevokeCurrentToken(context.Background(), token)
}

func setup(external, internal, tokenPath, out string) error {
	if err := admitSetupPaths(external, internal, out); err != nil {
		return err
	}
	token, err := config.Secret(tokenPath)
	if err != nil {
		return err
	}
	u, err := setupOperator(internal, token)
	if err != nil {
		return err
	}
	dir := filepath.Dir(out)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	keyPath := filepath.Join(dir, "grant-key")
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return fmt.Errorf("generate identity encryption key: %w", err)
	}
	if err := writeSetupSecret(keyPath, base64.StdEncoding.EncodeToString(key)); err != nil {
		return err
	}
	c := config.Config{Listen: "127.0.0.1:8080", ForgejoURL: strings.TrimRight(external, "/"), ForgejoInternalURL: strings.TrimRight(internal, "/"), Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", GrantKeyFile: keyPath, OperatorID: u.ID}
	if err = writeSetupConfig(out, c); err != nil {
		// This run created the key through O_EXCL, so no other setup owns
		// it; remove it so a retry is not blocked by our own partial state.
		_ = os.Remove(keyPath)
		return err
	}
	// Revoke only after the configuration is durable: earlier failures keep
	// the bootstrap token so the operator can retry with it.
	if err := revokeBootstrapToken(internal, token); err != nil {
		return fmt.Errorf("dashboard configuration written, but the bootstrap token is still live; revoke it in Forgejo Settings > Applications, then continue activation with the installed soda-activate command: %v", err)
	}
	fmt.Println("Dashboard configuration created and the bootstrap token revoked. Set native soda service ownership before enabling the dashboard. No host privilege was granted to a Forgejo user.")
	return nil
}
