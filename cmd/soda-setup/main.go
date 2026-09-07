// soda-setup is an operator-invoked one-time dashboard OAuth setup, not a daemon.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"os"
	"path/filepath"
	"strings"
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
	for _, u := range []string{*external, *internal} {
		if err := config.BaseURL(u); err != nil {
			return err
		}
	}
	if !strings.HasPrefix(*external, "https://") {
		return fmt.Errorf("browser origins must use HTTPS")
	}
	if !filepath.IsAbs(*out) {
		return fmt.Errorf("out must be absolute")
	}
	if _, err := os.Stat(*out); !os.IsNotExist(err) {
		return fmt.Errorf("configuration already exists or cannot be inspected; refusing overwrite")
	}
	token, err := config.Secret(*tokenPath)
	if err != nil {
		return err
	}
	client := forgejo.New(*internal)
	ctx := context.Background()
	u, err := client.Current(ctx, token)
	if err != nil {
		return err
	}
	if !u.Admin {
		return fmt.Errorf("Forgejo operator token is not a site administrator")
	}
	a, err := client.Application(ctx, token, (config.Config{ForgejoURL: strings.TrimRight(*external, "/")}).OAuthCallbackURL())
	if err != nil {
		return err
	}
	if a.ClientID == "" || a.Secret == "" {
		return fmt.Errorf("Forgejo returned an incomplete OAuth application")
	}
	dir := filepath.Dir(*out)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	secretPath := filepath.Join(dir, "oauth-secret")
	adminPath := filepath.Join(dir, "admin-token")
	keyPath := filepath.Join(dir, "grant-key")
	key := make([]byte, 32)
	rand.Read(key)
	for path, value := range map[string]string{secretPath: a.Secret, adminPath: token, keyPath: base64.StdEncoding.EncodeToString(key)} {
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return fmt.Errorf("OAuth application created, but credential write failed; inspect Forgejo applications before retrying: %w", e)
		}
		_, e = f.WriteString(value + "\n")
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
	}
	c := config.Config{Listen: "127.0.0.1:8080", ForgejoURL: strings.TrimRight(*external, "/"), ForgejoInternalURL: strings.TrimRight(*internal, "/"), Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", OAuthClientID: a.ClientID, OAuthSecretFile: secretPath, GrantKeyFile: keyPath, AdminTokenFile: adminPath, OperatorID: u.ID}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = json.NewEncoder(f).Encode(c); err != nil {
		return err
	}
	fmt.Println("Dashboard configuration created. Set native soda service ownership before enabling the dashboard. No host privilege was granted to a Forgejo user.")
	return nil
}
