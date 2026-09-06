package runners

import (
	"context"
	"errors"
	"github.com/levitateos/sodaos/internal/linuxhost"
)

type Authorizer interface {
	RequireAdministrator(context.Context, linuxhost.PKExecIdentity) error
}
type LinuxAccounts interface {
	LookupAccount(context.Context, string) (linuxhost.Account, error)
}
type LinuxAuthorizer struct{ Accounts LinuxAccounts }

func (a LinuxAuthorizer) RequireAdministrator(ctx context.Context, actor linuxhost.PKExecIdentity) error {
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
