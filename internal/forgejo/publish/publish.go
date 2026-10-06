// Package publish owns the narrow Forgejo publication process. Agent Git
// configuration, hooks and Forgejo credentials never share a working directory.
package publish

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

type Config struct {
	Root           string
	Remote         string
	Username       string
	TokenFile      string
	ProtectedPaths []string
}

type Request struct {
	Run                  factory.Run
	BaseSHA              string
	Commit               string
	Bundle               []byte
	ProtectedCredentials []string
}

func (c Config) Validate() error {
	if err := validateRemote(c.Remote); err != nil {
		return err
	}
	if !filepath.IsAbs(c.Root) || !filepath.IsAbs(c.TokenFile) || c.Username == "" {
		return errors.New("explicit publisher configuration required")
	}
	return nil
}

func validateRemote(remote string) error {
	u, err := url.Parse(remote)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Host == "" {
		return errors.New("invalid publication remote")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (localFixture(u.Hostname())) {
		return nil
	}
	return errors.New("publication requires TLS or a local fixture")
}

func (r Request) Validate() error {
	if r.Run.Role != project.RoleCoder {
		return errors.New("only the coding role may publish a candidate")
	}
	if err := r.Run.Validate(); err != nil {
		return err
	}
	if !factory.ValidCommit(r.BaseSHA) || !factory.ValidCommit(r.Commit) || r.Commit == r.Run.InputSHA {
		return errors.New("candidate is not a fresh exact commit")
	}
	if len(r.Bundle) == 0 || len(r.Bundle) > 4<<20 {
		return errors.New("candidate bundle exceeds input limit")
	}
	return nil
}

// ValidateCandidate imports only a verified bundle into a fresh bare
// repository, then checks the exact candidate, its ancestry, protected paths
// and credential material. It performs no push; publication is a separately
// persisted conditional operation built on this validation.
func (c Config) ValidateCandidate(ctx context.Context, r Request) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	git, cleanup, err := c.prepare(ctx, r)
	if err != nil {
		return err
	}
	defer cleanup()
	if err = git.validateCandidate(ctx, r, c.ProtectedPaths); err != nil {
		return err
	}
	return git.checkCredentials(ctx, r)
}

func (c Config) prepare(ctx context.Context, r Request) (*repository, func(), error) {
	if err := privateInputs(c); err != nil {
		return nil, nil, err
	}
	dir, err := os.MkdirTemp(c.Root, "publication-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	git := &repository{dir: dir, token: c.TokenFile, user: c.Username}
	if err = git.initialize(ctx, r.Bundle); err != nil {
		cleanup()
		return nil, nil, err
	}
	return git, cleanup, nil
}

func privateInputs(c Config) error {
	for _, path := range []string{c.Root, c.TokenFile} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o077 != 0 || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("publisher inputs must be private")
		}
		if path == c.Root && !info.IsDir() {
			return errors.New("publisher root must be a directory")
		}
		if path == c.TokenFile && !info.Mode().IsRegular() {
			return errors.New("publisher token must be a regular file")
		}
	}
	return nil
}

func protected(name string, paths []string) bool {
	for _, prefix := range append([]string{".forgejo/", ".github/workflows/"}, paths...) {
		if name == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (g *repository) validateCandidate(ctx context.Context, r Request, paths []string) error {
	out, err := g.run(ctx, "bundle", "list-heads", filepath.Join(g.dir, "candidate.bundle"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) != r.Commit+" HEAD" {
		return errors.New("candidate bundle does not match reported commit")
	}
	if _, err = g.run(ctx, "fetch", "--no-tags", filepath.Join(g.dir, "candidate.bundle"), "HEAD:refs/heads/candidate"); err != nil {
		return err
	}
	if _, err = g.run(ctx, "merge-base", "--is-ancestor", r.Run.InputSHA, r.Commit); err != nil {
		return errors.New("candidate is not based on admitted input")
	}
	out, err = g.run(ctx, "diff", "--name-only", "-z", r.BaseSHA, r.Commit)
	if err != nil {
		return err
	}
	for _, name := range strings.Split(string(out), "\x00") {
		if protected(name, paths) {
			return fmt.Errorf("candidate changes protected policy or CI path")
		}
	}
	return nil
}

func localFixture(host string) bool { return host == "127.0.0.1" || host == "localhost" }
