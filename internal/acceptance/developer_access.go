// Opt-in Sodaspaces direct project-IP SSH/PTY/SCP/SFTP and sudo checks. Go
// port of the retired tests/installed/developer-access.py: consumes the
// native browser's public access result and an independently verified public
// host key while private authentication keys stay on this client. It retains
// only new run-owned probe directories. It is not appliance runtime code.
package acceptance

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"path/filepath"
)

//go:embed developer_access.go developer_access_users.go developer_access_session.go developer_access_transfer.go
var developerAccessSources embed.FS

// developerAccessFiles lists the probe sources in hash order.
var developerAccessFiles = []string{
	"developer_access.go",
	"developer_access_users.go",
	"developer_access_session.go",
	"developer_access_transfer.go",
}

// developerAccessDigest hashes the embedded probe sources.
func developerAccessDigest() (string, error) {
	sum := sha256.New()
	for _, name := range developerAccessFiles {
		data, err := developerAccessSources.ReadFile(name)
		if err != nil {
			return "", err
		}
		sum.Write(data)
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// DeveloperAccessUsage documents the probe's CLI surface.
const DeveloperAccessUsage = "soda-installed-probes developer-access ROOT_DIR"

// ptyProbeInput is piped to ssh -tt, byte-identical to the retired probe.
const ptyProbeInput = "test -t 0 && printf \"\\nSODA-PTY:%s\\n\" \"$(id -un)\"\nexit\n"

// identityProbeCommand reads the remote identity, byte-identical.
const identityProbeCommand = "id -un; id -u; printf \"%s\\n\" \"$HOME\""

// accessRequest is the validated probe request.
type accessRequest struct {
	target      string
	revision    string
	project     string
	subnet      netip.Prefix
	broadcast   netip.Addr
	browserPath string
	hostKeyPath string
	users       []accessRequestUser
	sshConfig   string
	browser     accessBrowser
	publicKey   string
	fingerprint string
	output      string
	outputName  string
	payload     []byte
	sourcePath  string
	knownPaths  []string
	endpoints   []string
	results     accessResults
}

// accessRequestUser is one validated requesting user.
type accessRequestUser struct {
	id            string
	login         string
	keyFile       string
	administrator bool
}

// accessBrowser mirrors the consumed browser result.
type accessBrowser struct {
	Target   string `json:"target"`
	Revision string `json:"revision"`
	Outcome  string `json:"outcome"`
	Access   struct {
		NativeJoinConfirmed bool   `json:"native_join_confirmed"`
		ReservationID       string `json:"reservation_id"`
		Users               []struct {
			ID         string `json:"id"`
			Login      string `json:"login"`
			Connection struct {
				Environment struct {
					ID      string `json:"id"`
					Running bool   `json:"running"`
					IP      string `json:"ip"`
				} `json:"environment"`
				HostKey     string `json:"host_key"`
				Fingerprint string `json:"fingerprint"`
			} `json:"connection"`
		} `json:"users"`
	} `json:"access"`
}

// accessUserResult preserves the retired per-user result order.
type accessUserResult struct {
	ID                   string `json:"id"`
	Login                string `json:"login"`
	IP                   string `json:"ip"`
	ProjectAdministrator bool   `json:"project_administrator"`
	DirectSSH            bool   `json:"direct_ssh"`
	SSHAuthentication    bool   `json:"ssh_authentication"`
	InteractivePTY       bool   `json:"interactive_pty"`
	SCPRoundtrip         bool   `json:"scp_roundtrip"`
	SFTPRoundtrip        bool   `json:"sftp_roundtrip"`
	ProbeDirectory       string `json:"probe_directory"`
}

// accessResults preserves the retired results.json key order.
type accessResults struct {
	Revision     string             `json:"revision"`
	Target       string             `json:"target"`
	Project      string             `json:"project"`
	Client       string             `json:"client"`
	ClientArch   string             `json:"client_arch"`
	Transport    string             `json:"transport"`
	ScriptSHA256 string             `json:"script_sha256"`
	Users        []accessUserResult `json:"users"`
	CrossUserKey string             `json:"cross_user_key,omitempty"`
	Outcome      string             `json:"outcome"`
}

// requestString extracts one required string field.
func requestString(raw map[string]json.RawMessage, key string) (string, error) {
	field, ok := raw[key]
	if !ok {
		return "", errors.New("access request field required: " + key)
	}
	var value string
	if json.Unmarshal(field, &value) != nil {
		return "", errors.New("access request field required: " + key)
	}
	return value, nil
}

// checkRequestKeys requires the exact request key set.
func checkRequestKeys(raw map[string]json.RawMessage) error {
	required := map[string]bool{"target": true, "revision": true, "project_id": true, "subnet": true, "browser_result": true, "host_key_file": true, "users": true}
	for key := range raw {
		if !required[key] && key != "ssh_config_file" {
			return errors.New("invalid access request keys")
		}
	}
	for key := range required {
		if _, ok := raw[key]; !ok {
			return errors.New("invalid access request keys")
		}
	}
	return nil
}

// decodeAccessRequest parses the private request with exact-keys checking.
func decodeAccessRequest(data []byte) (accessRequest, []byte, error) {
	var request accessRequest
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return request, nil, errors.New("invalid access request")
	}
	if err := checkRequestKeys(raw); err != nil {
		return request, nil, err
	}
	fields, err := requestStrings(raw)
	if err != nil {
		return request, nil, err
	}
	request.target, request.revision, request.project = fields[0], fields[1], fields[2]
	subnet, broadcast, err := parseAccessSubnet(fields[3])
	if err != nil {
		return request, nil, err
	}
	request.subnet, request.broadcast = subnet, broadcast
	request.browserPath, request.hostKeyPath = fields[4], fields[5]
	request.sshConfig = "/dev/null"
	if configRaw, ok := raw["ssh_config_file"]; ok {
		if json.Unmarshal(configRaw, &request.sshConfig) != nil {
			return request, nil, errors.New("invalid access request")
		}
	}
	usersRaw, ok := raw["users"]
	if !ok {
		return request, nil, errors.New("invalid access request")
	}
	return request, usersRaw, nil
}

// requestStrings extracts the six string fields in validation order.
func requestStrings(raw map[string]json.RawMessage) ([6]string, error) {
	var fields [6]string
	keys := [6]string{"target", "revision", "project_id", "subnet", "browser_result", "host_key_file"}
	for i, key := range keys {
		value, err := requestString(raw, key)
		if err != nil {
			return fields, err
		}
		fields[i] = value
	}
	return fields, nil
}

// parseAccessSubnet requires a strict private IPv4 network.
func parseAccessSubnet(value string) (netip.Prefix, netip.Addr, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil || !prefix.Addr().Is4() {
		return netip.Prefix{}, netip.Addr{}, errors.New("private IPv4 subnet required")
	}
	masked := prefix.Masked()
	if masked.Addr() != prefix.Addr() {
		return netip.Prefix{}, netip.Addr{}, errors.New("private IPv4 subnet required")
	}
	broadcast := prefixBroadcast(masked)
	if !masked.Addr().IsPrivate() || !broadcast.IsPrivate() {
		return netip.Prefix{}, netip.Addr{}, errors.New("private IPv4 subnet required")
	}
	return masked, broadcast, nil
}

// prefixBroadcast returns the last address of a masked IPv4 prefix.
func prefixBroadcast(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr().As4()
	bits := prefix.Bits()
	var mask [4]byte
	for i := 0; i < 32; i++ {
		if i < bits {
			mask[i/8] |= 1 << (7 - (i % 8))
		}
	}
	var last [4]byte
	for i := 0; i < 4; i++ {
		last[i] = addr[i] | ^mask[i]
	}
	return netip.AddrFrom4(last)
}

// loadAccessRequest reads and validates the full probe request.
func loadAccessRequest(root string) (accessRequest, error) {
	var request accessRequest
	data, err := privateFile(filepath.Join(root, "target.json"), 65536)
	if err != nil {
		return request, err
	}
	request, usersRaw, err := decodeAccessRequest(data)
	if err != nil {
		return request, err
	}
	if err := validateAccessIDs(request); err != nil {
		return request, err
	}
	if err := loadAccessBrowser(&request); err != nil {
		return request, err
	}
	if err := loadAccessHostKey(&request); err != nil {
		return request, err
	}
	return request, validateAccessUsers(&request, usersRaw)
}

// RunDeveloperAccess is the developer-access entrypoint: ROOT_DIR.
func RunDeveloperAccess(args []string, stdout io.Writer) error {
	if err := runDeveloperAccess(args, stdout); err != nil {
		return fail("Developer access incomplete; retained probe state; failure type: ", err)
	}
	return nil
}

func runDeveloperAccess(args []string, stdout io.Writer) error {
	RestrictUmask()
	if len(args) != 1 {
		return errors.New(DeveloperAccessUsage)
	}
	if err := privateDir(args[0], false); err != nil {
		return err
	}
	request, err := loadAccessRequest(args[0])
	if err != nil {
		return err
	}
	if err := setupDeveloperOutput(args[0], &request); err != nil {
		return err
	}
	if err := runAccessChecks(&request); err != nil {
		request.results.Outcome = "failed; retained partial access state"
		if writeErr := writeAccessResults(&request); writeErr != nil {
			return writeErr
		}
		return err
	}
	request.results.CrossUserKey = "public-key authentication denied"
	request.results.Outcome = "passed-scoped-access"
	if err := writeAccessResults(&request); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Declared memberships passed native SSH, PTY, SCP/SFTP, owner sudo and cross-user key denial."); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "Client placement/routing is recorded separately; not laptop, lifecycle or workload acceptance.")
	return err
}
