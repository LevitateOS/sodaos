package runners

import (
	"context"
	"errors"
)

type Authorizer interface {
	RequireAdministrator(context.Context, PKExecIdentity) error
}
type LinuxAccounts interface {
	LookupAccount(context.Context, string) (Account, error)
}
type LinuxAuthorizer struct{ Accounts LinuxAccounts }

func (a LinuxAuthorizer) RequireAdministrator(ctx context.Context, actor PKExecIdentity) error {
	if actor.UID != 0 || actor.Username != "root" {
		return errors.New("host root operator required")
	}
	account, err := a.Accounts.LookupAccount(ctx, actor.Username)
	if err != nil {
		return err
	}
	if account.UID != 0 || account.Username != actor.Username {
		return errors.New("native operator identity changed")
	}
	return nil
}
