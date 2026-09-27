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
test ! -L /usr/local/bin/soda-identity-compose
if test -e /usr/local/bin/soda-identity-compose; then test -f /usr/local/bin/soda-identity-compose; fi
stage=$(mktemp -d /usr/local/bin/.soda-git-maintain.XXXXXXXX)
trap 'rm -rf -- "$stage"' EXIT
tar --extract --file=- --directory="$stage" --no-same-owner
chmod 0755 "$stage/git-remote-soda" "$stage/soda-identity-compose"
mv -T -- "$stage/git-remote-soda" /usr/local/bin/git-remote-soda
mv -T -- "$stage/soda-identity-compose" /usr/local/bin/soda-identity-compose
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
	compose, err := openTool(filepath.Join(tools, "soda-identity-compose"), "soda-identity-compose")
	if err != nil {
		closeTools([]tool{source})
		return err
	}
	defer closeTools([]tool{source, compose})
	if err := stagePublicTools(ctx, target, []tool{source, compose}, installGitScript, nil); err != nil {
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
