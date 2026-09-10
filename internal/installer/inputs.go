// Package installer owns the bounded console-to-CoreOS installation adapter.
// Disk writes and appliance installation remain explicit operator actions.
package installer

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

func Hostname(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if !regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`).MatchString(label) {
			return false
		}
	}
	return true
}

func PublicKey(value string) (string, error) {
	if len(value) > 16384 || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("one SSH public key required")
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(value))
	if err != nil || len(options) != 0 || len(rest) != 0 {
		return "", errors.New("valid SSH public key without authorized_keys options required")
	}
	switch key.Type() {
	case ssh.KeyAlgoED25519, ssh.KeyAlgoRSA, ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521, ssh.KeyAlgoSKED25519, ssh.KeyAlgoSKECDSA256:
	default:
		return "", errors.New("unsupported operator SSH key type")
	}
	// Strip comments and normalize native wire encoding; never accept certificates.
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))), nil
}

func ProjectSubnet(value string, routes []string) error {
	subnet, err := netip.ParsePrefix(value)
	if err != nil || !subnet.Addr().Is4() || subnet != subnet.Masked() {
		return errors.New("canonical IPv4 project subnet required")
	}
	private := false
	for _, raw := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		p := netip.MustParsePrefix(raw)
		if p.Bits() <= subnet.Bits() && p.Contains(subnet.Addr()) {
			private = true
		}
	}
	if !private {
		return errors.New("RFC1918 project subnet required")
	}
	for _, raw := range routes {
		if raw == "default" || raw == "" {
			continue
		}
		p, e := netip.ParsePrefix(raw)
		if e != nil {
			if a, e := netip.ParseAddr(raw); e == nil {
				p = netip.PrefixFrom(a, a.BitLen())
			} else {
				return errors.New("cannot interpret current network route")
			}
		}
		if p.Bits() != 0 && p.Overlaps(subnet) {
			return errors.New("project subnet overlaps a current host route")
		}
	}
	return nil
}

// Destination extends a public, strictly Butane-converted template. The builder
// owns that template; this is not an arbitrary user-supplied Ignition interpreter.
// Butane is not required on the live OS and private inputs never reach its logs.
func Destination(template []byte, hostname, key, passwordHash, subnet string) ([]byte, error) {
	var normalized string
	if key != "" {
		var err error
		normalized, err = PublicKey(key)
		if err != nil {
			return nil, err
		}
	}
	if !Hostname(hostname) || ProjectSubnet(subnet, nil) != nil || !regexp.MustCompile(`^\$6\$[./a-zA-Z0-9]{1,16}\$[./a-zA-Z0-9]{86}$`).MatchString(passwordHash) {
		return nil, errors.New("invalid private provisioning inputs")
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(template, &config); err != nil {
		return nil, errors.New("invalid public destination template")
	}
	var ignition struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(config["ignition"], &ignition) != nil || ignition.Version != "3.5.0" {
		return nil, errors.New("expected converted Ignition 3.5.0 template")
	}
	if _, exists := config["passwd"]; exists {
		return nil, errors.New("public template must not contain accounts")
	}
	var storage map[string]json.RawMessage
	if err := json.Unmarshal(config["storage"], &storage); err != nil || storage == nil {
		return nil, errors.New("invalid public storage template")
	}
	var files []json.RawMessage
	if err := json.Unmarshal(storage["files"], &files); err != nil {
		return nil, errors.New("invalid public files template")
	}
	for _, file := range files {
		var entry struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(file, &entry) != nil || entry.Path == "/etc/hostname" || entry.Path == "/etc/soda-installer/project-subnet" {
			return nil, errors.New("provisioning path collision")
		}
	}
	for path, value := range map[string]string{"/etc/hostname": hostname + "\n", "/etc/soda-installer/project-subnet": subnet + "\n"} {
		entry, _ := json.Marshal(map[string]interface{}{"path": path, "mode": 0600, "contents": map[string]string{"source": "data:;base64," + base64.StdEncoding.EncodeToString([]byte(value))}})
		files = append(files, entry)
	}
	storage["files"], _ = json.Marshal(files)
	config["storage"], _ = json.Marshal(storage)
	root := map[string]interface{}{"name": "root", "passwordHash": passwordHash}
	if normalized != "" {
		root["sshAuthorizedKeys"] = []string{normalized}
	}
	config["passwd"], _ = json.Marshal(map[string]interface{}{"users": []interface{}{root}})
	return json.Marshal(config)
}
