// Package config loads and validates dashboard and related operator JSON. It is
// not a secret store, schema migrator or runtime daemon.
package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SodaPath is the fixed browser namespace on the configured Forgejo origin.
// Caddy forwards this prefix unchanged; the Go service owns its routing.
const SodaPath = "/-/soda"

type Config struct {
	Listen             string `json:"listen"`
	ForgejoURL         string `json:"forgejo_url"`
	ForgejoInternalURL string `json:"forgejo_internal_url"`
	Database           string `json:"database"`
	HostSocket         string `json:"host_socket"`
	IdentitySocket     string `json:"identity_socket"`
	GrantKeyFile       string `json:"grant_key_file"`
	OperatorID         int64  `json:"operator_id"`
	// Optional native background admission inputs. When set, all three are
	// required: the operator-configured shared service callback socket, the
	// expected native host peer UID verified against kernel credentials,
	// and the restricted native PAT file read via Secret.
	ForgejoBackgroundSocket         string  `json:"forgejo_background_socket"`
	ForgejoBackgroundHostUID        *uint32 `json:"forgejo_background_host_uid"`
	ForgejoBackgroundCredentialFile string  `json:"forgejo_background_credential_file"`
	// Optional separate native reviewer credential. It must belong to the
	// repository policy's distinct review-only operation binding.
	ForgejoReviewCredentialFile string `json:"forgejo_review_credential_file"`
	// Optional native webhook intake secret file. When set, the factory
	// intake route verifies delivery HMACs against it; when unset or
	// unreadable, intake refuses as unavailable instead of guessing.
	FactoryIntakeSecretFile string `json:"factory_intake_secret_file"`
	// Optional private publication workspace root. When unset, the
	// backend uses its private data directory. The directory must be
	// backend-private; validated bundles and push workspaces live here,
	// never inside a Project.
	FactoryPublicationRoot string `json:"factory_publication_root"`
}

// BackgroundServiceConfigured reports whether native background service
// admission inputs are present.
func (c Config) BackgroundServiceConfigured() bool {
	return c.ForgejoBackgroundSocket != "" || c.ForgejoBackgroundHostUID != nil || c.ForgejoBackgroundCredentialFile != ""
}

func decodeConfig(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	// Read the whole file before decoding: a LimitReader around the decoder
	// would exhaust the limit during the first Decode and hide trailing
	// bytes past it as a clean EOF in the second.
	contents, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(contents) > 65536 {
		return c, errors.New("configuration exceeds 64 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(contents))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, fmt.Errorf("configuration: %w", err)
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return c, errors.New("configuration must contain one JSON object")
	}
	return c, nil
}

func validateListen(c *Config) error {
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8080"
	}
	listenHost, _, err := net.SplitHostPort(c.Listen)
	if err != nil || !net.ParseIP(listenHost).IsLoopback() {
		return errors.New("listen must be a loopback IP:port behind the private HTTPS proxy")
	}
	return nil
}

func validateConfigURLs(c *Config) error {
	for name, value := range map[string]string{"forgejo_url": c.ForgejoURL, "forgejo_internal_url": c.ForgejoInternalURL} {
		if err := BaseURL(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if !strings.HasPrefix(c.ForgejoURL, "https://") {
		return errors.New("browser origin must use HTTPS")
	}
	c.ForgejoURL = strings.TrimRight(c.ForgejoURL, "/")
	c.ForgejoInternalURL = strings.TrimRight(c.ForgejoInternalURL, "/")
	return nil
}

func validateConfigPaths(c Config) error {
	if c.IdentitySocket != "" && !filepath.IsAbs(c.IdentitySocket) {
		return errors.New("identity_socket must be an absolute path")
	}
	for name, value := range map[string]string{"database": c.Database, "host_socket": c.HostSocket, "grant_key_file": c.GrantKeyFile} {
		if !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be an absolute path", name)
		}
	}
	if c.OperatorID <= 0 {
		return errors.New("operator_id is required; run operator setup first")
	}
	if c.FactoryIntakeSecretFile != "" && !filepath.IsAbs(c.FactoryIntakeSecretFile) {
		return errors.New("factory_intake_secret_file must be an absolute path")
	}
	if c.FactoryPublicationRoot != "" && !filepath.IsAbs(c.FactoryPublicationRoot) {
		return errors.New("factory_publication_root must be an absolute path")
	}
	if c.ForgejoReviewCredentialFile != "" {
		if !filepath.IsAbs(c.ForgejoReviewCredentialFile) {
			return errors.New("forgejo_review_credential_file must be an absolute path")
		}
		if !c.BackgroundServiceConfigured() {
			return errors.New("forgejo_review_credential_file requires background service inputs")
		}
	}
	if c.BackgroundServiceConfigured() {
		if !filepath.IsAbs(c.ForgejoBackgroundSocket) {
			return errors.New("forgejo_background_socket must be an absolute path")
		}
		if c.ForgejoBackgroundHostUID == nil {
			return errors.New("forgejo_background_host_uid is required with background service inputs")
		}
		if !filepath.IsAbs(c.ForgejoBackgroundCredentialFile) {
			return errors.New("forgejo_background_credential_file must be an absolute path")
		}
	}
	return nil
}

func Load(path string) (Config, error) {
	c, err := decodeConfig(path)
	if err != nil {
		return c, err
	}
	if err = validateListen(&c); err != nil {
		return c, err
	}
	if err = validateConfigURLs(&c); err != nil {
		return c, err
	}
	if c.IdentitySocket == "" {
		c.IdentitySocket = "/run/soda/identity/admin.sock"
	}
	if err = validateConfigPaths(c); err != nil {
		return c, err
	}
	return c, nil
}

func originURL(u *url.URL) bool {
	return u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/")
}

func BaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || !originURL(u) {
		return errors.New("must be an HTTP(S) origin without credentials, path, query or fragment")
	}
	return nil
}

// GrantKey reads a separately provisioned 32-byte base64 key. Never generate a
// replacement at startup: that would strand the persisted encrypted grants.
func GrantKey(path string) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o027 != 0 {
		return nil, errors.New("grant key must be a restricted regular file (0600 or 0640)")
	}
	value, err := Secret(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, errors.New("grant key must encode exactly 32 bytes")
	}
	return key, nil
}

func Secret(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("cannot open configured credential file")
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o007 != 0 {
		return "", errors.New("credential must be a regular file inaccessible to other users")
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(b) > 65536 {
		return "", errors.New("cannot read credential file")
	}
	value := strings.TrimSpace(string(b))
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("credential is empty or malformed")
	}
	return value, nil
}
