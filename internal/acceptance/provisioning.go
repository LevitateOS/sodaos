package acceptance

import (
	"encoding/json"
	"errors"
	"strings"
)

type ignitionSecretsConfig struct {
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
		Files []ignitionFile
	}
}

func singleCompleteIgnitionV3(version string, merge []json.RawMessage, replace json.RawMessage) bool {
	return strings.HasPrefix(version, "3.") && len(merge) == 0 && len(replace) == 0
}

func passwordHashSecrets(users []struct{ PasswordHash string }) [][]byte {
	var secrets [][]byte
	for _, user := range users {
		if user.PasswordHash != "" {
			secrets = append(secrets, []byte(user.PasswordHash))
		}
	}
	return secrets
}

func isPrivateSSHHostKeyPath(path string) bool {
	return strings.HasPrefix(path, "/etc/ssh/ssh_host_") && !strings.HasSuffix(path, ".pub")
}

func hostKeyMaterialSecrets(source string, decoded []byte) [][]byte {
	secrets := [][]byte{[]byte(source), decoded}
	for _, line := range strings.Split(string(decoded), "\n") {
		if len(line) > 0 {
			secrets = append(secrets, []byte(line))
		}
	}
	return secrets
}

func appendHostKeySecrets(secrets [][]byte, files []ignitionFile) ([][]byte, error) {
	for _, file := range files {
		if !isPrivateSSHHostKeyPath(file.Path) {
			continue
		}
		decoded, err := inlineData(file.Contents.Source, file.Contents.Compression)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, hostKeyMaterialSecrets(file.Contents.Source, decoded)...)
	}
	return secrets, nil
}

// ProvisioningSecrets collects private bootstrap values before serial capture.
// One explicit fw_cfg Ignition document is used, never a guessed merge with
// another disk config. Additional personal inputs still need --secret-file.
func ProvisioningSecrets(path string) ([][]byte, error) {
	data, err := PrivateFile(path)
	if err != nil {
		return nil, err
	}
	var config ignitionSecretsConfig
	if json.Unmarshal(data, &config) != nil || !singleCompleteIgnitionV3(config.Ignition.Version, config.Ignition.Config.Merge, config.Ignition.Config.Replace) {
		return nil, errors.New("single complete Ignition v3 input required; external merge/replace is not supported")
	}
	return appendHostKeySecrets(passwordHashSecrets(config.Passwd.Users), config.Storage.Files)
}
