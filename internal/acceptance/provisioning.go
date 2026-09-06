package acceptance

import (
	"encoding/json"
	"errors"
	"strings"
)

// ProvisioningSecrets collects private bootstrap values before serial capture.
// One explicit fw_cfg Ignition document is used, never a guessed merge with
// another disk config. Additional personal inputs still need --secret-file.
func ProvisioningSecrets(path string) ([][]byte, error) {
	data, err := PrivateFile(path)
	if err != nil {
		return nil, err
	}
	var config struct {
		Ignition struct {
			Version string
			Config  struct {
				Merge   []json.RawMessage
				Replace json.RawMessage
			}
		}
		Passwd struct {
			Users []struct{ PasswordHash string }
		}
		Storage struct {
			Files []struct {
				Path     string
				Contents struct{ Source string }
			}
		}
	}
	if json.Unmarshal(data, &config) != nil || !strings.HasPrefix(config.Ignition.Version, "3.") || len(config.Ignition.Config.Merge) != 0 || len(config.Ignition.Config.Replace) != 0 {
		return nil, errors.New("single complete Ignition v3 input required; external merge/replace is not supported")
	}
	var secrets [][]byte
	for _, user := range config.Passwd.Users {
		if user.PasswordHash != "" {
			secrets = append(secrets, []byte(user.PasswordHash))
		}
	}
	for _, file := range config.Storage.Files {
		if !strings.HasPrefix(file.Path, "/etc/ssh/ssh_host_") || strings.HasSuffix(file.Path, ".pub") {
			continue
		}
		source := file.Contents.Source
		decoded, err := inlineData(source)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, []byte(source), decoded)
		for _, line := range strings.Split(string(decoded), "\n") {
			if len(line) > 0 {
				secrets = append(secrets, []byte(line))
			}
		}
	}
	return secrets, nil
}
