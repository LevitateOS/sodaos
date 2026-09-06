package web

import (
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Both the legacy form and JSON API use the same public-only key policy.
func normalizeDevelopmentKey(input string) (public, fingerprint string, err error) {
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(input))
	if err != nil || len(options) != 0 || len(strings.TrimSpace(string(rest))) != 0 {
		return "", "", errors.New("provide one public SSH key without authorized_keys options; never submit a private key")
	}
	return string(ssh.MarshalAuthorizedKey(key)), ssh.FingerprintSHA256(key), nil
}
