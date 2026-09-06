package forgejo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var endpointName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.:-]*$`)

// UpdateSSHDomain changes only the native Git SSH advertisement, preserving
// explicit browser/OAuth origins and every unrelated Forgejo setting.
func UpdateSSHDomain(path, endpoint string) (bool, error) {
	if !endpointName.MatchString(endpoint) {
		return false, fmt.Errorf("invalid native Tailnet endpoint")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	key := "FORGEJO__server__SSH_DOMAIN="
	value := key + endpoint
	found := false
	out := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, key) {
			if !found {
				out = append(out, value)
				found = true
			}
			continue
		}
		out = append(out, line)
	}
	if !found {
		out = append(out, value)
	}
	updated := strings.Join(out, "\n") + "\n"
	if updated == string(b) {
		return false, nil
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".forgejo-env-")
	if err != nil {
		return false, err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.WriteString(updated)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return false, err
	}
	return true, os.Rename(f.Name(), path)
}
