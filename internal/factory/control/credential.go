package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/factory"
)

func credentialSHA(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *Controller) authPath() string {
	return filepath.Join(c.Config.Workspace.CredentialHome, "auth.json")
}

func (c *Controller) checkCredentialStream(ctx context.Context) error {
	runs, err := c.Store.FactoryUnreturnedCredentials(ctx)
	if err != nil {
		return err
	}
	state, err := privateFile(c.authPath())
	if err != nil {
		return err
	}
	for _, r := range runs {
		if !r.CleanupComplete || credentialSHA(state) == r.CredentialSeedSHA {
			return errors.New("credential state was not returned; reconcile and reauthenticate")
		}
	}
	return nil
}

func (c *Controller) delegateCredential(ctx context.Context, r *factory.Run) error {
	state, err := privateFile(c.authPath())
	if err != nil {
		return err
	}
	if !json.Valid(state) {
		return errors.New("invalid enrolled credential state")
	}
	r.CredentialDelegated = true
	r.CredentialSeedSHA = credentialSHA(state)
	if err = c.Store.SaveFactoryRun(ctx, *r); err != nil {
		return err
	}
	return c.Workspace.SeedCredential(ctx, *r, state)
}

func (c *Controller) returnCredential(ctx context.Context, r *factory.Run) error {
	state, err := c.Workspace.CaptureCredential(ctx, *r)
	if err != nil {
		return err
	}
	if err = saveCredential(c.authPath(), r.CredentialSeedSHA, state); err != nil {
		return err
	}
	r.CredentialReturned = true
	return c.Store.SaveFactoryRun(ctx, *r)
}

// The stream lease excludes factory writers. A changed enrollment is preserved;
// the controller never restores stale state over a newer human login.
func saveCredential(path, seed string, state []byte) error {
	if !validCredentialState(state) {
		return errors.New("invalid returned credential state")
	}
	current, err := privateFile(path)
	if err != nil {
		return err
	}
	if credentialSHA(current) != seed {
		return errors.New("credential enrollment changed during execution")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".auth-return-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err = writeCredential(file, state); err != nil {
		return err
	}
	current, err = privateFile(path)
	if err != nil {
		return err
	}
	if credentialSHA(current) != seed {
		return errors.New("credential enrollment changed during execution")
	}
	return os.Rename(file.Name(), path)
}

func writeCredential(file *os.File, state []byte) error {
	defer func() { _ = file.Close() }()
	if _, err := file.Write(state); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func (c *Controller) returnDelegatedCredential(ctx context.Context, r *factory.Run) error {
	if !r.CredentialDelegated || r.CredentialReturned {
		return nil
	}
	return c.returnCredential(ctx, r)
}

func validCredentialState(state []byte) bool {
	return len(state) > 0 && len(state) <= 256<<10 && json.Valid(state)
}
