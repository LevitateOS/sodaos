package runners

import (
	"context"
	"errors"
	"testing"
)

type fakeLinuxAccounts struct {
	account Account
	err     error
}

func (a fakeLinuxAccounts) LookupAccount(context.Context, string) (Account, error) {
	return a.account, a.err
}
func TestOnlyHostRootOperator(t *testing.T) {
	a := LinuxAuthorizer{Accounts: fakeLinuxAccounts{account: Account{Username: "root", UID: 0}}}
	if err := a.RequireAdministrator(context.Background(), PKExecIdentity{Username: "root", UID: 0}); err != nil {
		t.Fatal(err)
	}
	for _, actor := range []PKExecIdentity{{Username: "alice", UID: 1000}, {Username: "soda", UID: 2000}, {Username: "other", UID: 0}} {
		if a.RequireAdministrator(context.Background(), actor) == nil {
			t.Fatal("non-operator accepted")
		}
	}
}
func TestNativeOperatorLookupFailure(t *testing.T) {
	a := LinuxAuthorizer{Accounts: fakeLinuxAccounts{err: errors.New("native lookup failed")}}
	if a.RequireAdministrator(context.Background(), PKExecIdentity{Username: "root", UID: 0}) == nil {
		t.Fatal("lookup failure ignored")
	}
}
