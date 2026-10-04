// Opt-in Sodaspaces direct project-IP SSH/PTY/SCP/SFTP and sudo checks. Go
// port of the retired tests/installed/developer-access.py: consumes the
// native browser's public access result and an independently verified public
// host key while private authentication keys stay on this client. It retains
// only new run-owned probe directories. It is not appliance runtime code.
package acceptance

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//go:embed developer_access.go
var developerAccessSource string

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

// validateAccessIDs checks target, environment binding, revision and project.
func validateAccessIDs(request accessRequest) error {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,252}$`).MatchString(request.target) {
		return errors.New("invalid access target")
	}
	if os.Getenv("SODA_NATIVE_VALIDATE") != request.target {
		return errors.New("explicit native validation required")
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(request.revision) {
		return errors.New("full revision required")
	}
	if !regexp.MustCompile(`^p[0-9a-f]{24}$`).MatchString(request.project) {
		return errors.New("invalid project identifier")
	}
	if request.sshConfig != "/dev/null" {
		if _, err := privateFile(request.sshConfig, 16384); err != nil {
			return err
		}
	}
	return nil
}

// loadAccessBrowser reads and validates the consumed browser result.
func loadAccessBrowser(request *accessRequest) error {
	data, err := privateFile(request.browserPath, 65536)
	if err != nil {
		return err
	}
	if json.Unmarshal(data, &request.browser) != nil {
		return errors.New("invalid browser result")
	}
	browser := &request.browser
	if browser.Target != request.target || browser.Revision != request.revision {
		return errors.New("browser result does not match request")
	}
	if browser.Outcome != "passed-scoped-journey" || !browser.Access.NativeJoinConfirmed {
		return errors.New("scoped browser journey required")
	}
	if browser.Access.ReservationID != request.project {
		return errors.New("browser reservation does not match project")
	}
	return nil
}

// loadAccessHostKey reads and normalizes the pinned host key.
func loadAccessHostKey(request *accessRequest) error {
	data, err := privateFile(request.hostKeyPath, 65536)
	if err != nil {
		return err
	}
	public := strings.TrimSpace(string(data))
	if !regexp.MustCompile(`^ssh-ed25519 [A-Za-z0-9+/]{68}(?: [^\r\n]*)?$`).MatchString(public) {
		return errors.New("invalid pinned host key")
	}
	request.publicKey = strings.Join(strings.Fields(public)[:2], " ")
	return nil
}

// validateAccessUser checks one requesting user record.
func validateAccessUser(raw json.RawMessage) (accessRequestUser, error) {
	var user accessRequestUser
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return user, errors.New("invalid access user")
	}
	if err := checkAccessUserKeys(fields); err != nil {
		return user, err
	}
	id, err := requestString(fields, "id")
	if err != nil {
		return user, err
	}
	if !regexp.MustCompile(`^[1-9][0-9]{0,18}$`).MatchString(id) {
		return user, errors.New("invalid access user")
	}
	login, err := checkAccessUserLogin(fields)
	if err != nil {
		return user, err
	}
	administrator, err := checkAccessUserAdmin(fields)
	if err != nil {
		return user, err
	}
	keyFile, err := requestString(fields, "key_file")
	if err != nil {
		return user, err
	}
	if _, err := privateFile(keyFile, 16384); err != nil {
		return user, err
	}
	user = accessRequestUser{id: id, login: login, keyFile: keyFile, administrator: administrator}
	return user, nil
}

// checkAccessUserKeys requires the exact user key set.
func checkAccessUserKeys(fields map[string]json.RawMessage) error {
	if len(fields) != 4 {
		return errors.New("invalid access user")
	}
	for _, key := range []string{"id", "login", "key_file", "administrator"} {
		if _, ok := fields[key]; !ok {
			return errors.New("invalid access user")
		}
	}
	return nil
}

// checkAccessUserLogin validates a non-root login.
func checkAccessUserLogin(fields map[string]json.RawMessage) (string, error) {
	login, err := requestString(fields, "login")
	if err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`).MatchString(login) || login == "root" {
		return "", errors.New("invalid access user")
	}
	return login, nil
}

// checkAccessUserAdmin requires a strict boolean administrator flag.
func checkAccessUserAdmin(fields map[string]json.RawMessage) (bool, error) {
	var flagged any
	if json.Unmarshal(fields["administrator"], &flagged) != nil {
		return false, errors.New("invalid access user")
	}
	administrator, ok := flagged.(bool)
	if !ok {
		return false, errors.New("invalid access user")
	}
	return administrator, nil
}

// validateAccessUsers checks both requesting users and their order.
func validateAccessUsers(request *accessRequest, usersRaw []byte) error {
	var rawUsers []json.RawMessage
	if json.Unmarshal(usersRaw, &rawUsers) != nil {
		return errors.New("two access users required")
	}
	if len(rawUsers) != 2 || len(request.browser.Access.Users) != 2 {
		return errors.New("two access users required")
	}
	users := make([]accessRequestUser, 0, 2)
	for _, raw := range rawUsers {
		user, err := validateAccessUser(raw)
		if err != nil {
			return err
		}
		users = append(users, user)
	}
	if users[0].id == users[1].id {
		return errors.New("two access users required")
	}
	if !users[0].administrator || users[1].administrator {
		return errors.New("first user must own project administration")
	}
	request.users = users
	return nil
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

// checkAccessSudo requires owner elevation or explicit refusal.
func checkAccessSudo(ssh []string, target string, administrator bool) error {
	outcome, err := runBounded(ssh[0], append(append([]string{}, ssh[1:]...), target, "sudo -n /usr/bin/id -u"), nil, 45*time.Second)
	if err != nil {
		return err
	}
	if administrator {
		if outcome.exitCode != 0 || string(bytes.TrimSpace(outcome.stdout)) != "0" {
			return errors.New("owner sudo proof failed")
		}
		return nil
	}
	if outcome.exitCode != 1 {
		return errors.New("member sudo refusal proof failed")
	}
	denied := false
	for _, marker := range []string{"password is required", "not allowed", "not in the sudoers"} {
		if bytes.Contains(outcome.stderr, []byte(marker)) {
			denied = true
		}
	}
	if !denied {
		return errors.New("member sudo refusal proof failed")
	}
	return nil
}

// quoteAccessPath quotes one sftp batch path.
func quoteAccessPath(path string) (string, error) {
	if strings.ContainsAny(path, "\r\n") {
		return "", errors.New("sftp path must be one line")
	}
	return "\"" + strings.ReplaceAll(strings.ReplaceAll(path, "\\", "\\\\"), "\"", "\\\"") + "\"", nil
}

// checkAccessTransfer runs the mkdir, SCP and SFTP roundtrips.
func checkAccessTransfer(request *accessRequest, ssh []string, target, login string) error {
	destination := "/home/" + login + "/" + request.outputName
	if _, err := accessChecked(append(append([]string{}, ssh...), target, "umask 077; mkdir "+destination), nil); err != nil {
		return err
	}
	received := filepath.Join(request.output, login+"-scp")
	if _, err := accessChecked(append([]string{"scp"}, append(append([]string{}, ssh[1:]...), request.sourcePath, target+":"+destination+"/scp")...), nil); err != nil {
		return err
	}
	if _, err := accessChecked(append([]string{"scp"}, append(append([]string{}, ssh[1:]...), target+":"+destination+"/scp", received)...), nil); err != nil {
		return err
	}
	data, err := os.ReadFile(received)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, request.payload) {
		return errors.New("SCP roundtrip mismatch")
	}
	return checkAccessSFTP(request, ssh, target, destination, login)
}

// checkAccessSFTP runs the SFTP roundtrip.
func checkAccessSFTP(request *accessRequest, ssh []string, target, destination, login string) error {
	copyPath := filepath.Join(request.output, login+"-sftp")
	source, err := quoteAccessPath(request.sourcePath)
	if err != nil {
		return err
	}
	copyQuote, err := quoteAccessPath(copyPath)
	if err != nil {
		return err
	}
	batch := "put " + source + " " + destination + "/sftp\nget " + destination + "/sftp " + copyQuote + "\n"
	sftp := append([]string{"sftp"}, append(append([]string{}, ssh[1:]...), "-b", "-", target)...)
	if _, err := accessChecked(sftp, []byte(batch)); err != nil {
		return err
	}
	data, err := os.ReadFile(copyPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, request.payload) {
		return errors.New("SFTP roundtrip mismatch")
	}
	return nil
}

// probeAccessUser runs every proof for one user.
func probeAccessUser(request *accessRequest, index int) error {
	user := request.users[index]
	ip, err := checkAccessConnection(request, index)
	if err != nil {
		return err
	}
	request.endpoints = append(request.endpoints, ip)
	known := filepath.Join(request.output, user.login+"-known-hosts")
	if err := os.WriteFile(known, []byte(ip+" "+request.publicKey+"\n"), 0o666); err != nil {
		return err
	}
	request.knownPaths = append(request.knownPaths, known)
	ssh := append([]string{"ssh"}, accessSSHOptions(request, user, known)...)
	target := user.login + "@" + ip
	if err := checkAccessSession(ssh, target, user.login); err != nil {
		return err
	}
	if err := checkAccessSudo(ssh, target, user.administrator); err != nil {
		return err
	}
	if err := checkAccessTransfer(request, ssh, target, user.login); err != nil {
		return err
	}
	request.results.Users = append(request.results.Users, accessUserResult{
		ID: user.id, Login: user.login, IP: ip, ProjectAdministrator: user.administrator,
		DirectSSH: request.sshConfig == "/dev/null", SSHAuthentication: true, InteractivePTY: true,
		SCPRoundtrip: true, SFTPRoundtrip: true, ProbeDirectory: "/home/" + user.login + "/" + request.outputName,
	})
	return nil
}

// checkCrossUserDenial requires public-key refusal across users.
func checkCrossUserDenial(request *accessRequest) error {
	first, second := request.users[0], request.users[1]
	known := request.knownPaths[len(request.knownPaths)-1]
	options := accessSSHOptions(request, accessRequestUser{id: first.id, login: first.login, keyFile: first.keyFile}, known)
	args := append(append([]string{"ssh"}, options...), second.login+"@"+request.endpoints[1], "true")
	outcome, err := runBounded(args[0], args[1:], nil, 45*time.Second)
	if err != nil {
		return err
	}
	if outcome.exitCode != 255 || !bytes.Contains(outcome.stderr, []byte("Permission denied (publickey")) {
		return errors.New("cross-user key denial proof failed")
	}
	return nil
}

// runAccessChecks executes the fingerprint, payload, user and denial proofs.
func runAccessChecks(request *accessRequest) error {
	if err := fetchAccessFingerprint(request); err != nil {
		return err
	}
	if err := writeAccessPayload(request); err != nil {
		return err
	}
	for index := range request.users {
		if err := probeAccessUser(request, index); err != nil {
			return err
		}
	}
	if request.endpoints[0] != request.endpoints[1] {
		return errors.New("user endpoints differ")
	}
	return checkCrossUserDenial(request)
}

// writeAccessResults records results.json with the retired shape.
func writeAccessResults(request *accessRequest) error {
	encoded, err := json.MarshalIndent(request.results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(request.output, "results.json"), append(encoded, '\n'), 0o666)
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
