// Operator identity is the native pkexec/root boundary the runner commands
// execute under. It does not provision human host accounts.
package runners

import (
	"context"
	"errors"
	"os"
	"os/user"
	"strconv"
)

type PKExecIdentity struct {
	Username string
	UID      int
}
type Account struct {
	Username string
	UID      int
}
type HostAccounts struct{}

func NewHostAccounts() *HostAccounts { return &HostAccounts{} }
func (*HostAccounts) LookupAccount(_ context.Context, name string) (Account, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return Account{}, err
	}
	uid, err := strconv.Atoi(u.Uid)
	return Account{Username: u.Username, UID: uid}, err
}

func PKExecCaller() (PKExecIdentity, error) {
	return pkexecCaller(os.Getuid(), os.Geteuid(), os.Getenv("PKEXEC_UID"), os.Getenv("SUDO_UID"))
}

// pkexecCaller gates the native pkexec/root boundary on kernel-reported IDs
// first, then on both delegation records: pkexec names its caller in
// PKEXEC_UID and sudo names its caller in SUDO_UID, so either one naming a
// non-root originator rejects the invocation. Empty means no delegation
// evidence (direct root), which the IDs above already gate.
func pkexecCaller(uid, euid int, pkexecUID, sudoUID string) (PKExecIdentity, error) {
	if euid != 0 || uid != 0 {
		return PKExecIdentity{}, errors.New("host root operator required")
	}
	for _, origin := range []string{pkexecUID, sudoUID} {
		if origin != "" && origin != "0" {
			return PKExecIdentity{}, errors.New("non-operator pkexec caller rejected")
		}
	}
	return PKExecIdentity{Username: "root", UID: 0}, nil
}
