// Developer-access output setup and per-user SSH session proofs.
package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// setupDeveloperOutput creates the run-owned output directory and results.
func setupDeveloperOutput(root string, request *accessRequest) error {
	run, err := uuidHex()
	if err != nil {
		return err
	}
	request.outputName = "sodaspaces-access-" + run
	request.output = filepath.Join(root, request.outputName)
	if err := os.Mkdir(request.output, 0o700); err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	sum := sha256.Sum256([]byte(developerAccessSource))
	transport := "direct project IP"
	if request.sshConfig != "/dev/null" {
		transport = "explicit SSH configuration; not direct-route proof"
	}
	request.results = accessResults{
		Revision:     request.revision,
		Target:       request.target,
		Project:      request.project,
		Client:       hostname,
		ClientArch:   machineArch(),
		Transport:    transport,
		ScriptSHA256: hex.EncodeToString(sum[:]),
		Users:        []accessUserResult{},
	}
	return nil
}

// accessChecked runs one selected operation and requires exit zero.
func accessChecked(args []string, stdin []byte) ([]byte, error) {
	outcome, err := runBounded(args[0], args[1:], stdin, 45*time.Second)
	if err != nil {
		return nil, err
	}
	if outcome.exitCode != 0 {
		return nil, fmt.Errorf("selected access operation failed (exit %d)", outcome.exitCode)
	}
	return outcome.stdout, nil
}

// fetchAccessFingerprint reads the pinned key fingerprint.
func fetchAccessFingerprint(request *accessRequest) error {
	raw, err := accessChecked([]string{"ssh-keygen", "-lf", request.hostKeyPath}, nil)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return errors.New("invalid host key fingerprint")
	}
	if !regexp.MustCompile(`^SHA256:[A-Za-z0-9+/]{43}$`).MatchString(fields[1]) {
		return errors.New("invalid host key fingerprint")
	}
	request.fingerprint = fields[1]
	return nil
}

// writeAccessPayload stores the roundtrip payload.
func writeAccessPayload(request *accessRequest) error {
	request.payload = bytes.Repeat([]byte(request.outputName+"\n"), 8192)
	request.sourcePath = filepath.Join(request.output, "payload")
	return os.WriteFile(request.sourcePath, request.payload, 0o666)
}

// accessSSHOptions builds the exact per-user SSH options.
func accessSSHOptions(request *accessRequest, user accessRequestUser, known string) []string {
	return []string{
		"-F", request.sshConfig,
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		"-o", "IdentityAgent=none",
		"-o", "PreferredAuthentications=publickey",
		"-o", "ForwardAgent=no",
		"-o", "ClearAllForwardings=yes",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + known,
		"-i", user.keyFile,
	}
}

// checkAccessConnection validates the browser-advertised endpoint.
func checkAccessConnection(request *accessRequest, index int) (string, error) {
	user := request.users[index]
	connection := request.browser.Access.Users[index]
	if connection.ID != user.id || connection.Login != user.login {
		return "", errors.New("browser user does not match request")
	}
	native := connection.Connection
	if native.Environment.ID != request.project || !native.Environment.Running {
		return "", errors.New("browser project is not running")
	}
	addr, err := parseAccessMemberIP(request, native.Environment.IP)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(native.HostKey) != request.publicKey || native.Fingerprint != request.fingerprint {
		return "", errors.New("browser host key does not match pin")
	}
	return addr.String(), nil
}

// parseAccessMemberIP validates a member IP against the selected subnet.
func parseAccessMemberIP(request *accessRequest, value string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(value)
	if err != nil || !addr.Is4() {
		return netip.Addr{}, errors.New("project IPv4 required")
	}
	if !request.subnet.Contains(addr) || addr == request.subnet.Addr() || addr == request.broadcast {
		return netip.Addr{}, errors.New("project IP outside the selected subnet")
	}
	return addr, nil
}

// checkAccessIdentity requires the exact remote account.
func checkAccessIdentity(ssh []string, target, login string) error {
	raw, err := accessChecked(append(append([]string{}, ssh...), target, identityProbeCommand), nil)
	if err != nil {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) < 3 || lines[0] != login || lines[2] != "/home/"+login {
		return errors.New("remote identity does not match login")
	}
	id, err := parseAccessUID(lines[1])
	if err != nil || id == 0 {
		return errors.New("remote identity must be non-root")
	}
	return nil
}

// parseAccessUID parses a remote uid the way Python's int() does.
func parseAccessUID(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("invalid remote uid")
	}
	sign, digits, err := splitAccessSign(trimmed)
	if err != nil {
		return 0, err
	}
	if strings.Contains(digits, "_") {
		digits, err = stripAccessUnderscores(digits)
		if err != nil {
			return 0, err
		}
	}
	uid, err := parseAccessDigits(digits)
	if err != nil {
		return 0, err
	}
	return sign * uid, nil
}

// splitAccessSign separates an optional leading sign.
func splitAccessSign(trimmed string) (int, string, error) {
	sign := 1
	digits := trimmed
	if digits[0] == '+' || digits[0] == '-' {
		if digits[0] == '-' {
			sign = -1
		}
		digits = digits[1:]
	}
	if digits == "" {
		return 0, "", errors.New("invalid remote uid")
	}
	return sign, digits, nil
}

// stripAccessUnderscores removes Python-style digit separators.
func stripAccessUnderscores(digits string) (string, error) {
	if !regexp.MustCompile(`^\d(_?\d)*$`).MatchString(digits) {
		return "", errors.New("invalid remote uid")
	}
	return strings.ReplaceAll(digits, "_", ""), nil
}

// parseAccessDigits parses plain decimal digits.
func parseAccessDigits(digits string) (int, error) {
	uid := 0
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return 0, errors.New("invalid remote uid")
		}
		uid = uid*10 + int(digits[i]-'0')
	}
	return uid, nil
}

// checkAccessUIDMap requires a user-namespaced project.
func checkAccessUIDMap(ssh []string, target string) error {
	raw, err := accessChecked(append(append([]string{}, ssh...), target, "/usr/bin/head -n 1 /proc/self/uid_map"), nil)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(raw))
	if len(fields) < 2 {
		return errors.New("project root must not map to host root")
	}
	first, err := parseAccessUID(fields[0])
	if err != nil {
		return err
	}
	second, err := parseAccessUID(fields[1])
	if err != nil {
		return err
	}
	if first != 0 || second == 0 {
		return errors.New("project root must not map to host root")
	}
	return nil
}

// checkAccessPTY requires an interactive terminal with the login.
func checkAccessPTY(ssh []string, target, login string) error {
	raw, err := accessChecked(append(append([]string{}, ssh...), "-tt", target), []byte(ptyProbeInput))
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n") {
		if line == "SODA-PTY:"+login {
			return nil
		}
	}
	return errors.New("interactive PTY proof missing")
}

// checkAccessSession runs the identity, namespace and PTY proofs.
func checkAccessSession(ssh []string, target, login string) error {
	if err := checkAccessIdentity(ssh, target, login); err != nil {
		return err
	}
	if err := checkAccessUIDMap(ssh, target); err != nil {
		return err
	}
	return checkAccessPTY(ssh, target, login)
}
