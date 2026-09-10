// Package linuxhost provides only the native operator identity boundary needed
// by the retained runner code. It does not provision human host accounts.
package linuxhost

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
type Native struct{}

func NewNative() *Native { return &Native{} }
func (*Native) LookupAccount(_ context.Context, name string) (Account, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return Account{}, err
	}
	uid, err := strconv.Atoi(u.Uid)
	return Account{Username: u.Username, UID: uid}, err
}
func PKExecCaller() (PKExecIdentity, error) {
	return pkexecCaller(os.Getuid(), os.Geteuid(), os.Getenv("PKEXEC_UID"))
}

func pkexecCaller(uid, euid int, originalUID string) (PKExecIdentity, error) {
	if euid != 0 || uid != 0 {
		return PKExecIdentity{}, errors.New("host root operator required")
	}
	if originalUID != "" && originalUID != "0" {
		return PKExecIdentity{}, errors.New("non-operator pkexec caller rejected")
	}
	return PKExecIdentity{Username: "root", UID: 0}, nil
}
