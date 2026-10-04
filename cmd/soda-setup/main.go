// soda-setup records the native operator identity and Soda service configuration.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// postgresSecretDir holds the mode-0600 PostgreSQL credential files setup
// generates: super/forgejo/soda .passwd plus the soda connection URL. The
// database units stay skipped until these exist, so live media shows a
// clean skip instead of a failed database.
const postgresSecretDir = "/etc/soda/postgres"

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
	return setup(*external, *internal, *tokenPath, *out, postgresSecretDir)
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

func writeSecretFile(path, value string) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if e != nil {
		return fmt.Errorf("create %s: %w", path, e)
	}
	_, e = f.WriteString(value + "\n")
	ce := f.Close()
	if e != nil {
		_ = os.Remove(path)
		return e
	}
	if ce != nil {
		_ = os.Remove(path)
		return ce
	}
	return nil
}

func randomPassword() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate database password: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// provisionPostgresSecrets generates the database credential files: one
// hex password per role plus the soda connection URL. Files land O_EXCL so
// a pre-existing secret is never overwritten; on failure this run removes
// only the files it created, preserving anything already there.
func provisionPostgresSecrets(dir string) (created []string, dsnPath string, err error) {
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", err
	}
	passwords := map[string]string{}
	for _, role := range []string{"super", "forgejo", "soda"} {
		var pw string
		pw, err = randomPassword()
		if err != nil {
			break
		}
		path := filepath.Join(dir, role+".passwd")
		if err = writeSecretFile(path, pw); err != nil {
			break
		}
		created = append(created, path)
		passwords[role] = pw
	}
	if err == nil {
		dsnPath = filepath.Join(dir, "soda.dsn")
		dsn := "postgres://soda:" + passwords["soda"] + "@soda-postgres:5432/soda?sslmode=disable"
		if err = writeSecretFile(dsnPath, dsn); err == nil {
			created = append(created, dsnPath)
		}
	}
	if err != nil {
		for _, path := range created {
			_ = os.Remove(path)
		}
		return nil, "", err
	}
	return created, dsnPath, nil
}

func removeOwnFiles(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func setup(external, internal, tokenPath, out, pgDir string) error {
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
	pgCreated, dsnPath, err := provisionPostgresSecrets(pgDir)
	if err != nil {
		_ = os.Remove(keyPath)
		return err
	}
	c := config.Config{Listen: "127.0.0.1:8080", ForgejoURL: strings.TrimRight(external, "/"), ForgejoInternalURL: strings.TrimRight(internal, "/"), DatabaseDSNFile: dsnPath, HostSocket: "/run/soda/host.sock", GrantKeyFile: keyPath, OperatorID: u.ID}
	if err = writeSetupConfig(out, c); err != nil {
		// This run created the key and database secrets through O_EXCL, so
		// no other setup owns them; remove them so a retry is not blocked
		// by our own partial state.
		_ = os.Remove(keyPath)
		removeOwnFiles(pgCreated)
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
