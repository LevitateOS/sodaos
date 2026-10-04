// Requesting-user and browser validation for the developer-access probe.
package acceptance

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
)

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
