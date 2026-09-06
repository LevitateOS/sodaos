package config

import (
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
	if _, _, err = net.SplitHostPort(c.Listen); err != nil {
		return c, errors.New("listen must be host:port")
	}
	for name, value := range map[string]string{"public_url": c.PublicURL, "forgejo_url": c.ForgejoURL, "forgejo_internal_url": c.ForgejoInternalURL} {
		if err := BaseURL(value); err != nil {
			return c, fmt.Errorf("%s: %w", name, err)
		}
	}
	for name, value := range map[string]string{"database": c.Database, "host_socket": c.HostSocket, "oauth_secret_file": c.OAuthSecretFile, "admin_token_file": c.AdminTokenFile} {
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
