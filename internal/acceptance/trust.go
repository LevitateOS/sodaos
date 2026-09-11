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

func inlineData(source, compression string) ([]byte, error) {
	parts := strings.SplitN(source, ",", 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "data:") {
		return nil, errors.New("inline data URI required")
	}
	data, err := url.PathUnescape(parts[1])
	if err != nil {
		return nil, errors.New("invalid inline encoding")
	}
	if strings.HasSuffix(parts[0], ";base64") {
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, errors.New("invalid inline base64")
		}
		data = string(b)
	}
	if compression == "" {
		return []byte(data), nil
	}
	if compression != "gzip" {
		return nil, errors.New("unsupported inline compression")
	}
	reader, err := gzip.NewReader(bytes.NewReader([]byte(data)))
	if err != nil {
		return nil, errors.New("invalid compressed inline data")
	}
	defer reader.Close()
	decoded, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
	if err != nil || len(decoded) > 1024*1024 {
		return nil, errors.New("invalid or oversized compressed inline data")
	}
	return decoded, nil
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
			Files []struct {
				Path     string
				Contents struct{ Source, Compression string }
			}
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
		if f.Path != "/etc/hostname" && f.Path != "/etc/ssh/ssh_host_ed25519_key" {
			continue
		}
		body, err := inlineData(f.Contents.Source, f.Contents.Compression)
		if err != nil {
			return err
		}
		if f.Path == "/etc/hostname" {
			hostname = strings.TrimSpace(string(body)) == name
			continue
		}
		signer, err := ssh.ParsePrivateKey(body)
		if err != nil {
			return errors.New("invalid per-instance host key")
		}
		if err = callback(host, address, signer.PublicKey()); err != nil {
			return errors.New("Ignition host key does not match pinned management trust")
		}
		matched = true
	}
	if !matched || !hostname {
		return errors.New("matching fixture hostname and pinned Ed25519 host key required in Ignition")
	}
	return nil
}
