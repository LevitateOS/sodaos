package publish

import (
	"context"
	"os"
)

func (c Config) sourceRepository(ctx context.Context) (*repository, func(), error) {
	if err := privateInputs(c); err != nil {
		return nil, nil, err
	}
	dir, err := os.MkdirTemp(c.Root, "source-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	git := &repository{dir: dir, token: c.TokenFile, user: c.Username}
	if err = git.initialize(ctx, []byte("source placeholder")); err != nil {
		cleanup()
		return nil, nil, err
	}
	return git, cleanup, nil
}
