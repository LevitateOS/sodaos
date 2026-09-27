//go:build linux

package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/host"
)

const installGitScript = `
set -eu
test ! -L /usr
test ! -L /usr/local
test ! -L /usr/local/bin
test -d /usr/local/bin
test ! -L /usr/local/bin/git-remote-soda
if test -e /usr/local/bin/git-remote-soda; then test -f /usr/local/bin/git-remote-soda; fi
stage=$(mktemp -d /usr/local/bin/.soda-git-maintain.XXXXXXXX)
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/git-remote-soda"
mv -T -- "$stage/git-remote-soda" /usr/local/bin/git-remote-soda
`

func maintainGit(o options, c host.Config) error {
	if c.GitSocket == "" {
		if o.bindOnly {
			return nil
		}
		return errors.New("native Git project runtime is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	target, err := waitProject(ctx, o.project)
	if err != nil {
		return err
	}
	if !o.bindOnly {
		if err := installProjectGit(ctx, target, o.tools, c.ForgejoURL); err != nil {
			return err
		}
	}
	if err := prepareLaunchInterface(ctx, target, "git"); err != nil {
		return err
	}
	return attachLaunchInterface(ctx, target, c.GitSocket, "git")
}

func installProjectGit(ctx context.Context, target observation, tools, origin string) error {
	source, err := openTool(filepath.Join(tools, "git-remote-soda"), "git-remote-soda")
	if err != nil {
		return err
	}
	defer closeTools([]tool{source})
	if err := stagePublicTools(ctx, target, []tool{source}, installGitScript, nil); err != nil {
		return err
	}
	if err := confirmProject(ctx, target); err != nil {
		return err
	}
	if _, err := podman(ctx, nil, "exec", "--user", "0:0", target.ID, "/usr/bin/git", "config", "--system", "url.soda://.insteadOf", strings.TrimSuffix(origin, "/")+"/"); err != nil {
		return err
	}
	return nil
}
