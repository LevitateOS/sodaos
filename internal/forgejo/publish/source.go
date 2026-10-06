package publish

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// Source supplies a fresh complete bundle at an exact trusted remote revision.
// Publication under this remote is a separately persisted conditional operation.
func (c Config) Source(ctx context.Context, sha string) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if !factory.ValidCommit(sha) {
		return nil, errors.New("invalid source revision")
	}
	git, cleanup, err := c.sourceRepository(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	dir := git.dir
	if _, err = git.run(ctx, "fetch", "--no-tags", c.Remote, sha+":refs/heads/source"); err != nil {
		return nil, err
	}
	if _, err = git.run(ctx, "symbolic-ref", "HEAD", "refs/heads/source"); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "source.bundle")
	if _, err = git.run(ctx, "bundle", "create", path, "HEAD"); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 4<<20 {
		return nil, errors.New("source bundle exceeds workspace input limit")
	}
	return data, nil
}

func (c Config) BranchRevision(ctx context.Context, branch string) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}
	if branch == "" || strings.ContainsAny(branch, "\x00\r\n :~^?*[\\") {
		return "", errors.New("invalid source branch")
	}
	git, cleanup, err := c.sourceRepository(ctx)
	if err != nil {
		return "", err
	}
	defer cleanup()
	target := "refs/heads/" + branch
	out, err := git.run(ctx, "ls-remote", c.Remote, target)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 || fields[1] != target || !factory.ValidCommit(fields[0]) {
		return "", errors.New("source branch has no exact revision")
	}
	return fields[0], nil
}

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
