package tailnet

import "strings"

// CanonicalMagicDNSName normalizes a Tailscale-supplied MagicDNS FQDN for use
// in connection instructions and certificate names.
func CanonicalMagicDNSName(value string) (string, error) {
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if len(name) > 253 || !strings.Contains(name, ".") || strings.HasSuffix(name, ".local") {
		return "", ErrInvalidMagicDNSName
	}
	for _, label := range strings.Split(name, ".") {
		if !validLabel(label) {
			return "", ErrInvalidMagicDNSName
		}
	}
	return name, nil
}

func validLabel(label string) bool {
	switch {
	case len(label) == 0:
		return false
	case len(label) > 63:
		return false
	case label[0] == '-':
		return false
	case label[len(label)-1] == '-':
		return false
	}
	return strings.IndexFunc(label, invalidLabelCharacter) == -1
}

func invalidLabelCharacter(character rune) bool {
	switch {
	case character == '-':
		return false
	case character >= 'a' && character <= 'z':
		return false
	case character >= '0' && character <= '9':
		return false
	default:
		return true
	}
}
