package config

import (
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

type Config struct {
	Listen             string `json:"listen"`
	PublicURL          string `json:"public_url"`
	ForgejoURL         string `json:"forgejo_url"`
	ForgejoInternalURL string `json:"forgejo_internal_url"`
	Database           string `json:"database"`
	HostSocket         string `json:"host_socket"`
	OAuthClientID      string `json:"oauth_client_id"`
	OAuthSecretFile    string `json:"oauth_secret_file"`
	GrantKeyFile       string `json:"grant_key_file"`
	AdminTokenFile     string `json:"admin_token_file"`
	OperatorID         int64  `json:"operator_id"`
}

func Load(path string) (Config, error) {
	var c Config
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 65537))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, fmt.Errorf("configuration: %w", err)
	}
	if err = d.Decode(new(any)); err != io.EOF {
		return c, errors.New("configuration must contain one JSON object")
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8080"
	}
	listenHost, _, err := net.SplitHostPort(c.Listen)
	if err != nil || !net.ParseIP(listenHost).IsLoopback() {
		return c, errors.New("listen must be a loopback IP:port behind the private HTTPS proxy")
	}
	for name, value := range map[string]string{"public_url": c.PublicURL, "forgejo_url": c.ForgejoURL, "forgejo_internal_url": c.ForgejoInternalURL} {
		if err := BaseURL(value); err != nil {
			return c, fmt.Errorf("%s: %w", name, err)
		}
	}
	if !strings.HasPrefix(c.PublicURL, "https://") || !strings.HasPrefix(c.ForgejoURL, "https://") {
		return c, errors.New("browser origins must use HTTPS")
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	c.ForgejoURL = strings.TrimRight(c.ForgejoURL, "/")
	c.ForgejoInternalURL = strings.TrimRight(c.ForgejoInternalURL, "/")
	for name, value := range map[string]string{"database": c.Database, "host_socket": c.HostSocket, "oauth_secret_file": c.OAuthSecretFile, "admin_token_file": c.AdminTokenFile, "grant_key_file": c.GrantKeyFile} {
		if !filepath.IsAbs(value) {
			return c, fmt.Errorf("%s must be an absolute path", name)
		}
	}
	if c.OAuthClientID == "" || c.OperatorID <= 0 {
		return c, errors.New("oauth_client_id and operator_id are required; run operator setup first")
	}
	return c, nil
}

func BaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("must be an HTTP(S) origin without credentials, path, query or fragment")
	}
	return nil
}

// GrantKey reads a separately provisioned 32-byte base64 key. Never generate a
// replacement at startup: that would strand the persisted encrypted grants.
func GrantKey(path string) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0027 != 0 {
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
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0007 != 0 {
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
