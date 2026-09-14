package installer

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
)

// CandidateLiveConfig is a media-only leaf. Its protected caller authenticates
// the candidate and supplies the already-compiled console's digest. That binary
// is in the same native, stream-verified rootfs: no separate executable download,
// compilation, disk selection, private input or reboot is performed here.
func CandidateLiveConfig(payload, destination []byte, manifest, consoleSHA256 string) ([]byte, error) {
	var p appliancerelease.Payload
	if json.Unmarshal(payload, &p) != nil || p.Validate() != nil {
		return nil, errors.New("complete ordinary-Podman candidate required")
	}
	var dest struct {
		Ignition struct{ Version string }
		Passwd   json.RawMessage
	}
	if json.Unmarshal(destination, &dest) != nil || dest.Ignition.Version != "3.5.0" || dest.Passwd != nil {
		return nil, errors.New("public converted destination template required")
	}
	sum := sha256.Sum256(payload)
	m := mediaIdentity{Format: 2, Architecture: p.Architecture, Release: p.CoreOS, Revision: p.Revision, InstallerVersion: "coreos-installer 0.26.0", HostManifest: manifest, PayloadSHA256: hex.EncodeToString(sum[:]), ConsoleSHA256: consoleSHA256}
	if m.validate(p.CoreOS, p.Architecture) != nil {
		return nil, errors.New("candidate media identity required")
	}
	media, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	inline := func(path string, contents []byte) map[string]any {
		return map[string]any{"path": path, "mode": 0644, "contents": map[string]string{"source": "data:;base64," + base64.StdEncoding.EncodeToString(contents)}}
	}
	// Ignition writes the native physical /var path, not through /usr/local.
	const liveData = "/var/usrlocal/share/soda-installer"
	files := []map[string]any{inline(liveData+"/media.json", media), inline(liveData+"/destination.ign", destination)}
	// These masks apply only to live Ignition, never to destination Ignition.
	var units []map[string]any
	for _, name := range []string{"getty@tty1.service", "forgejo.service", "soda-dashboard.service", "soda-proxy.service", "soda-host.service", "soda-host.socket", "soda-image-import.service"} {
		units = append(units, map[string]any{"name": name, "mask": true})
	}
	body := strings.Join([]string{"[Unit]", "Description=SodaOS installation console", "After=systemd-user-sessions.service NetworkManager.service", "Conflicts=getty@tty1.service", "[Service]", "Type=idle", "PrivateMounts=yes", "ExecStart=" + candidateInstallerBinary + " disk", "StandardInput=tty-force", "StandardOutput=tty", "StandardError=tty", "TTYPath=/dev/tty1", "TTYReset=yes", "TTYVHangup=yes", "Restart=no", "[Install]", "WantedBy=multi-user.target", ""}, "\n")
	units = append(units, map[string]any{"name": "soda-installer-console.service", "enabled": true, "contents": body})
	return json.Marshal(map[string]any{"ignition": map[string]string{"version": "3.5.0"}, "storage": map[string]any{"files": files}, "systemd": map[string]any{"units": units}})
}
