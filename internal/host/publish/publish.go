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
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type Config struct {
	Root           string
	Remote         string
	Username       string
	TokenFile      string
	ProtectedPaths []string
}

type Request struct {
	Attempt              factory.Attempt
	Run                  factory.Run
	Commit               string
	Bundle               []byte
	ProtectedCredentials []string
}

func Branch(attemptID string) string { return "soda/factory/" + attemptID }

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

func (r Request) Validate(now time.Time) error {
	a := r.Attempt
	if err := a.Authority(now); err != nil {
		return err
	}
	if err := r.Run.Validate(); err != nil {
		return err
	}
	if err := r.Run.Authority(a.ID, now); err != nil {
		return err
	}
	return a.CandidateFrom(r.Run, r.Commit, now)
}

// Candidate imports only a verified bundle into a fresh bare repository, then
// publishes one predetermined branch using an exact expected-commit lease.
// authorize must recheck durable state and Forgejo authority under the controller
// publication lock; cancellation uses the same lock.
func (c Config) Candidate(ctx context.Context, r Request, authorize func() error) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := r.Validate(time.Now()); err != nil {
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
	if err = git.checkCredentials(ctx, r); err != nil {
		return err
	}
	target := "refs/heads/" + Branch(r.Attempt.ID)
	if err = authorize(); err != nil {
		return err
	}
	if err = r.Validate(time.Now()); err != nil {
		return err
	}
	if err = git.verifyTarget(ctx, c.Remote, target, r.Attempt.Candidate); err != nil {
		return err
	}
	expected := r.Attempt.Candidate
	_, err = git.run(ctx, "push", "--force-with-lease="+target+":"+expected, c.Remote, r.Commit+":"+target)
	return err
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
	out, err = g.run(ctx, "diff", "--name-only", "-z", r.Attempt.Work.BaseSHA, r.Commit)
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

func (g *repository) verifyTarget(ctx context.Context, remote, target, expected string) error {
	out, err := g.run(ctx, "ls-remote", remote, target)
	if err != nil {
		return err
	}
	want := ""
	if expected != "" {
		want = expected + "\t" + target
	}
	if strings.TrimSpace(string(out)) != want {
		return errors.New("publication target differs from expected revision")
	}
	return nil
}
