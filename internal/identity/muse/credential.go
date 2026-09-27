package muse

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/levitateos/sodaos/internal/identity"
)

// CredentialValid accepts native device OAuth, including its CLI subscription key.
// A manually inserted PAYG key is not a subscription enrollment.
func CredentialValid(data []byte) bool {
	if !identity.CredentialValid(data) {
		return false
	}
	var wire struct {
		SchemaVersion int `json:"schema_version"`
		Providers     struct {
			Meta struct {
				AccessToken string `json:"access_token"`
				APIKey      string `json:"api_key"`
				APIBaseURL  string `json:"api_base_url"`
				Mechanism   string `json:"mechanism"`
				ObtainedVia string `json:"obtained_via"`
			} `json:"meta"`
		} `json:"providers"`
	}
	if json.Unmarshal(data, &wire) != nil {
		return false
	}
	m := wire.Providers.Meta
	return wire.SchemaVersion == 1 && m.Mechanism == "oauth" && m.ObtainedVia == "device_code" && m.APIBaseURL == "https://api.meta.ai/v1" && m.AccessToken != "" && m.APIKey != ""
}

func credentialFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 256<<10 {
		return nil, errors.New("invalid Muse credential file")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 256<<10+1))
	if err != nil || !CredentialValid(data) {
		return nil, errors.New("invalid Muse subscription credential")
	}
	return data, nil
}
