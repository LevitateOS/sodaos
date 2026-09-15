package acceptance

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type ignitionFile struct {
	Path     string
	Contents struct{ Source, Compression string }
}

func decodeDataURI(source string) ([]byte, error) {
	parts := strings.SplitN(source, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:") {
		return nil, errors.New("inline data URI required")
	}
	data, err := url.PathUnescape(parts[1])
	if err != nil {
		return nil, errors.New("invalid inline encoding")
	}
	if !strings.HasSuffix(parts[0], ";base64") {
		return []byte(data), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, errors.New("invalid inline base64")
	}
	return decoded, nil
}

func gunzipBounded(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("invalid compressed inline data")
	}
	defer reader.Close()
	decoded, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
	if oversizedCompressedInline(err, decoded) {
		return nil, errors.New("invalid or oversized compressed inline data")
	}
	return decoded, nil
}

func oversizedCompressedInline(err error, decoded []byte) bool {
	return err != nil || len(decoded) > 1024*1024
}

func inlineData(source, compression string) ([]byte, error) {
	data, err := decodeDataURI(source)
	if err != nil {
		return nil, err
	}
	if compression == "" {
		return data, nil
	}
	if compression != "gzip" {
		return nil, errors.New("unsupported inline compression")
	}
	return gunzipBounded(data)
}

func isFixtureTrustFile(path string) bool {
	return path == "/etc/hostname" || path == "/etc/ssh/ssh_host_ed25519_key"
}

func matchFixtureHostname(body []byte, name string) bool {
	return strings.TrimSpace(string(body)) == name
}

func verifyPinnedHostKey(body []byte, host string, address net.Addr, callback ssh.HostKeyCallback) error {
	signer, err := ssh.ParsePrivateKey(body)
	if err != nil {
		return errors.New("invalid per-instance host key")
	}
	if err = callback(host, address, signer.PublicKey()); err != nil {
		return errors.New("ignition host key does not match pinned management trust")
	}
	return nil
}

func fixtureTrustPresent(matched, hostname bool) error {
	if !matched || !hostname {
		return errors.New("matching fixture hostname and pinned Ed25519 host key required in Ignition")
	}
	return nil
}

func applyFixtureTrustFile(f ignitionFile, name, host string, address net.Addr, callback ssh.HostKeyCallback, matched, hostname *bool) error {
	if !isFixtureTrustFile(f.Path) {
		return nil
	}
	body, err := inlineData(f.Contents.Source, f.Contents.Compression)
	if err != nil {
		return err
	}
	if f.Path == "/etc/hostname" {
		*hostname = matchFixtureHostname(body, name)
		return nil
	}
	if err = verifyPinnedHostKey(body, host, address, callback); err != nil {
		return err
	}
	*matched = true
	return nil
}

// Verify the supplied trust root against the selected private bootstrap before
// starting anything. No keyscan, trust replacement or product key API is used.
func VerifyFixtureTrust(path, name string, r Remote) error {
	data, err := PrivateFile(path)
	if err != nil {
		return err
	}
	var config struct {
		Storage struct {
			Files []ignitionFile
		}
	}
	if json.Unmarshal(data, &config) != nil {
		return errors.New("invalid private Ignition")
	}
	callback, err := knownhosts.New(r.KnownHosts)
	if err != nil {
		return errors.New("invalid pinned known_hosts")
	}
	host := net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
	address := &net.TCPAddr{IP: net.ParseIP(r.Host), Port: r.Port}
	matched, hostname := false, false
	for _, f := range config.Storage.Files {
		if err = applyFixtureTrustFile(f, name, host, address, callback, &matched, &hostname); err != nil {
			return err
		}
	}
	return fixtureTrustPresent(matched, hostname)
}
