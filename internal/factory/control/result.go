package control

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// credentialStrings inspects the exact delegated or returned state, not a later
// read of enrollment that a human login could have replaced.
func credentialStrings(data []byte) ([]string, error) {
	var auth struct {
		APIKey string            `json:"OPENAI_API_KEY"`
		Tokens map[string]string `json:"tokens"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, errors.New("invalid enrolled credential state")
	}
	values := []string{}
	if auth.APIKey != "" {
		values = append(values, auth.APIKey)
	}
	for _, key := range []string{"access_token", "refresh_token", "id_token", "account_id"} {
		if value := auth.Tokens[key]; value != "" {
			values = append(values, value)
		}
	}
	return values, nil
}

func resultBytes(result factory.Result, secrets []string) ([]byte, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(data) > 300<<10 {
		return nil, errors.New("agent result exceeds retained output limit")
	}
	for _, secret := range secrets {
		encoded, err := json.Marshal(secret)
		if err != nil {
			return nil, err
		}
		if secret != "" && strings.Contains(string(data), string(encoded[1:len(encoded)-1])) {
			return nil, errors.New("agent result contains protected credential material")
		}
	}
	return data, nil
}

// retainResult retains only the bounded structured contract, never the transcript.
// It also runs before publication so known enrolled secrets cannot enter PR text.
func (c *Controller) retainResult(r factory.Run, result factory.Result, secrets []string) error {
	data, err := resultBytes(result, secrets)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.Config.Workspace.Root, r.ID, "result.json"), data, 0o600)
}
