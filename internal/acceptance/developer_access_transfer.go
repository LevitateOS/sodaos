// Developer-access privilege and file-transfer proofs plus denial.
package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
