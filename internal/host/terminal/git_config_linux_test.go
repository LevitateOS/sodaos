//go:build linux

package terminal

import (
	"context"
	"io"
	"strings"
	"testing"
)

type gitConfigExecutor struct {
	t         *testing.T
	secret    string
	protected bool
	wrote     bool
}

func (e *gitConfigExecutor) Run(_ context.Context, body []byte, _ string, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	if strings.Contains(command, e.secret) {
		e.t.Fatal("capability exposed in argv")
	}
	if strings.Contains(command, "--mode=0600 /dev/null") {
		e.protected = true
	}
	if len(body) > 0 {
		if !e.protected || !strings.Contains(string(body), e.secret) || !strings.Contains(command, "/usr/bin/dd") {
			e.t.Fatal("unprotected capability input")
		}
		e.wrote = true
	}
	return nil, nil
}

func (e *gitConfigExecutor) RunReader(context.Context, io.Reader, string, ...string) ([]byte, error) {
	e.t.Fatal("unexpected reader call")
	return nil, nil
}

func TestGitCapabilityRestrictedConfigInput(t *testing.T) {
	executor := &gitConfigExecutor{t: t, secret: "synthetic-invocation-capability"}
	runtime := &GitRuntime{Exec: executor}
	if err := runtime.stageConfig(context.Background(), museCaller{Container: strings.Repeat("a", 64), UID: 1000, GID: 1000}, "/run/soda-git/"+strings.Repeat("b", 32), executor.secret); err != nil {
		t.Fatal(err)
	}
	if !executor.wrote {
		t.Fatal("missing protected config input")
	}
}
