//go:build linux

package terminal

import (
	"context"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
)

// configureGitIdentity updates only Git's global author keys for the attested
// project login. Repository-local choices retain their normal precedence.
func (g *GitRuntime) configureGitIdentity(ctx context.Context, c museCaller, session identity.GitSession) error {
	if !gitIdentityValue(session.Name) || !gitIdentityValue(session.Email) {
		return identity.ErrDenied
	}
	for _, item := range [][2]string{{"user.name", session.Name}, {"user.email", session.Email}} {
		_, err := g.Exec.Run(ctx, nil, "/usr/bin/podman", "--remote=false", "exec", "--user="+strconv.Itoa(c.UID)+":"+strconv.Itoa(c.GID), "--env=HOME="+c.Home, c.Container, "/usr/bin/git", "config", "--global", "--replace-all", item[0], item[1])
		if err != nil {
			return err
		}
	}
	return nil
}

func gitIdentityValue(value string) bool {
	return value != "" && len(value) <= 1024 && !strings.ContainsAny(value, "\x00\r\n")
}

// stageConfig writes the invocation capability through restricted stdin, never
// argv or environment. The independent unit reads only this protected tmpfs file.
func (g *GitRuntime) stageConfig(ctx context.Context, c museCaller, path, capability string) error {
	runtime := &MuseRuntime{Exec: g.Exec}
	commands := [][]string{
		{"/usr/bin/install", "--directory", "--mode=0700", path},
		{"/usr/bin/mount", "--types=tmpfs", "--options=mode=0700,size=64k", "tmpfs", path},
		{"/usr/bin/install", "--mode=0600", "/dev/null", path + "/gitconfig"},
	}
	for _, command := range commands {
		if _, err := runtime.guest(ctx, c.Container, nil, command...); err != nil {
			return err
		}
	}
	body := []byte("[http]\n\textraHeader =\n\textraHeader = X-Soda-Git-Invocation: " + capability + "\n")
	defer clear(body)
	if _, err := runtime.guest(ctx, c.Container, body, "/usr/bin/dd", "of="+path+"/gitconfig", "status=none"); err != nil {
		return err
	}
	if _, err := runtime.guest(ctx, c.Container, nil, "/usr/bin/chown", strconv.Itoa(c.UID)+":"+strconv.Itoa(c.GID), path+"/gitconfig"); err != nil {
		return err
	}
	_, err := runtime.guest(ctx, c.Container, nil, "/usr/bin/chmod", "0711", path)
	return err
}

func (g *GitRuntime) removeConfig(ctx context.Context, container, path string) error {
	runtime := &MuseRuntime{Exec: g.Exec}
	id := strings.TrimPrefix(path, "/run/soda-git/")
	if !terminalID.MatchString(id) {
		return identity.ErrDenied
	}
	if _, err := runtime.guest(ctx, container, nil, "/usr/bin/test", "!", "-d", path); err == nil {
		return nil
	}
	if _, err := runtime.guest(ctx, container, nil, "/usr/bin/umount", path); err != nil {
		return err
	}
	_, err := runtime.guest(ctx, container, nil, "/usr/bin/rmdir", path)
	return err
}
