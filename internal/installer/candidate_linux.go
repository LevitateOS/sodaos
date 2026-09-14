package installer

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

const candidateInstallerBinary = "/usr/libexec/soda/soda-install"

func (m mediaIdentity) validContent() bool {
	switch m.Format {
	case 0:
		return nativebuild.Digest(m.BundleSHA256) && m.HostManifest == "" && m.PayloadSHA256 == "" && m.ConsoleSHA256 == ""
	case 2:
		return m.BundleSHA256 == "" && strings.HasPrefix(m.HostManifest, "sha256:") && nativebuild.Digest(strings.TrimPrefix(m.HostManifest, "sha256:")) && nativebuild.Digest(m.PayloadSHA256) && nativebuild.Digest(m.ConsoleSHA256)
	default:
		return false
	}
}

// The authenticated minimal ISO anchors native stream verification before live
// Ignition runs. Verify its expected Soda payload and all local image content, not an
// ISO mount or rpm-ostree status (the live EROFS root is not a booted deployment).
func candidateRequirement(m mediaIdentity, root string) (uint64, error) {
	if m.Format != 2 || m.validate(m.Release, architecture()) != nil || m.InstallerVersion != "coreos-installer 0.26.0" {
		return 0, errors.New("invalid candidate media identity")
	}
	console, err := nativebuild.HashFile(filepath.Join(root, candidateInstallerBinary))
	if err != nil || console != m.ConsoleSHA256 {
		return 0, errors.New("candidate installer differs from prebuilt tool")
	}
	payload := filepath.Join(root, appliancerelease.Path)
	hash, err := nativebuild.HashFile(payload)
	if err != nil || hash != m.PayloadSHA256 {
		return 0, errors.New("live candidate payload differs from authenticated media")
	}
	p, err := appliancerelease.Load(payload)
	if err != nil || p.Revision != m.Revision || p.Architecture != m.Architecture || p.CoreOS != m.Release {
		return 0, errors.New("live candidate release mismatch")
	}
	_, total, err := appliancerelease.VerifyContent(p, filepath.Join(root, appliancerelease.ImagesPath))
	return total, err
}

func candidateDestination(template, factory []byte, choices diskInstallChoices) ([]byte, error) {
	var defaults map[string]any
	if json.Unmarshal(factory, &defaults) != nil || len(defaults) != 4 || defaults["subnet"] != "" || defaults["tailnet_management"] != false {
		return nil, errors.New("invalid native machine defaults")
	}
	for _, key := range []string{"network", "bridge"} {
		if value, ok := defaults[key].(string); !ok || value == "" {
			return nil, errors.New("missing native machine network defaults")
		}
	}
	defaults["subnet"] = choices.subnet
	machine, err := json.Marshal(defaults)
	if err != nil {
		return nil, err
	}
	destination, err := Destination(template, choices.hostname, "", choices.passwordHash, choices.subnet)
	if err != nil {
		return nil, err
	}
	var config map[string]json.RawMessage
	if err = json.Unmarshal(destination, &config); err != nil {
		return nil, err
	}
	var storage map[string]json.RawMessage
	if err = json.Unmarshal(config["storage"], &storage); err != nil {
		return nil, err
	}
	var files []map[string]any
	if err = json.Unmarshal(storage["files"], &files); err != nil {
		return nil, err
	}
	for _, file := range files {
		if file["path"] == "/etc/soda/host.json" {
			return nil, errors.New("machine configuration collision")
		}
		if file["path"] == "/etc/soda-installer/project-subnet" {
			file["path"] = "/etc/soda/host.json"
			file["contents"] = map[string]string{"source": "data:;base64," + base64.StdEncoding.EncodeToString(machine)}
		}
	}
	storage["files"], err = json.Marshal(files)
	if err != nil {
		return nil, err
	}
	config["storage"], err = json.Marshal(storage)
	if err != nil {
		return nil, err
	}
	return json.Marshal(config)
}
