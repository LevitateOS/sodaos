//go:build linux

package terminal

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

type takeoverExec struct {
	calls   [][]string
	inspect string
	destDir bool
	srcDir  bool
	failOn  string
}

func (f *takeoverExec) Run(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	joined := strings.Join(args, " ")
	switch {
	case strings.Contains(joined, "inspect"):
		return []byte(f.inspect), nil
	case strings.Contains(joined, "/usr/bin/test -d ") && strings.Contains(joined, "factory-takeover/"):
		if f.destDir {
			return nil, nil
		}
		return nil, errors.New("exit 1")
	case strings.Contains(joined, "/usr/bin/test -d "):
		if f.srcDir {
			return nil, nil
		}
		return nil, errors.New("exit 1")
	case strings.Contains(joined, "/usr/bin/test -e "):
		return nil, errors.New("exit 1")
	case f.failOn != "" && strings.Contains(joined, f.failOn):
		return nil, errors.New("step failed")
	default:
		return nil, nil
	}
}

func (f *takeoverExec) RunReader(ctx context.Context, in io.Reader, name string, args ...string) ([]byte, error) {
	return f.Run(ctx, nil, name, args...)
}

const (
	takeoverProject = "p123456789012345678901234"
	takeoverPrep    = "f0123456789abcdef01234567"
)

var takeoverContainer = strings.Repeat("c", 64)

func takeoverInspect(project string, running bool) string {
	state := "false"
	if running {
		state = "true"
	}
	return `{"id":"` + takeoverContainer + `","running":` + state + `,"project":"` + project + `","owner":"7","privileged":false,"userns":"private","mappings":{}}`
}

func TestFactoryTakeoverCopySucceeds(t *testing.T) {
	run := strings.Repeat("a", 32)
	exec := &takeoverExec{inspect: takeoverInspect(takeoverProject, true), srcDir: true}
	s := &Service{Exec: exec}
	dest, reused, err := s.FactoryTakeoverCopy(context.Background(), takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "alice", run)
	if err != nil || reused || dest != domain.TakeoverDestination("alice", run) {
		t.Fatal(dest, reused, err)
	}
	joined := ""
	for _, call := range exec.calls {
		flat := strings.Join(call, " ")
		joined += flat + "\n"
		if strings.Contains(flat, "inspect") {
			continue
		}
		for _, arg := range call {
			if strings.ContainsAny(arg, ";|&$`\\\"'") {
				t.Fatalf("shell metacharacter in argv: %q", call)
			}
		}
	}
	for _, want := range []string{
		"/usr/bin/cp -a /home/soda-coder/checkouts/" + takeoverPrep + "/. ",
		"/usr/bin/rm -rf " + dest + ".partial/.git " + dest + ".partial/.soda-home",
		"/usr/bin/git -C " + dest + ".partial init -q",
		"/usr/bin/chown -R --reference=/home/alice " + dest + ".partial",
		"/usr/bin/mv -T " + dest + ".partial " + dest,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing step %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "/home/alice/factory-takeover/.partial") {
		t.Fatal("partial path escaped the run destination")
	}
}

func TestFactoryTakeoverRefusesStaleIncarnation(t *testing.T) {
	run := strings.Repeat("a", 32)
	exec := &takeoverExec{inspect: takeoverInspect(takeoverProject, true), srcDir: true}
	s := &Service{Exec: exec}
	if _, _, err := s.FactoryTakeoverCopy(context.Background(), takeoverProject, strings.Repeat("d", 64), domain.RoleCoder, takeoverPrep, "alice", run); !errors.Is(err, identity.ErrStale) {
		t.Fatalf("replaced container: %v", err)
	}
	for _, call := range exec.calls {
		if strings.Contains(strings.Join(call, " "), "exec") {
			t.Fatalf("stale incarnation reached the native boundary: %q", call)
		}
	}
	stopped := &takeoverExec{inspect: takeoverInspect(takeoverProject, false), srcDir: true}
	s.Exec = stopped
	if _, _, err := s.FactoryTakeoverCopy(context.Background(), takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "alice", run); !errors.Is(err, identity.ErrStale) {
		t.Fatalf("stopped project: %v", err)
	}
}

func TestFactoryTakeoverReusesExistingDestination(t *testing.T) {
	run := strings.Repeat("a", 32)
	exec := &takeoverExec{inspect: takeoverInspect(takeoverProject, true), srcDir: true, destDir: true}
	s := &Service{Exec: exec}
	dest, reused, err := s.FactoryTakeoverCopy(context.Background(), takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "alice", run)
	if err != nil || !reused || dest != domain.TakeoverDestination("alice", run) {
		t.Fatal(dest, reused, err)
	}
	for _, call := range exec.calls {
		if strings.Contains(strings.Join(call, " "), "/usr/bin/cp") {
			t.Fatal("existing destination copied again")
		}
	}
}

func TestFactoryTakeoverValidatesBeforeExecuting(t *testing.T) {
	exec := &takeoverExec{inspect: takeoverInspect(takeoverProject, true), srcDir: true}
	s := &Service{Exec: exec}
	run := strings.Repeat("a", 32)
	for _, bad := range [][6]string{
		{takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "root", run},
		{takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "../x", run},
		{takeoverProject, takeoverContainer, "alice", takeoverPrep, "alice", run},
		{takeoverProject, "short", domain.RoleCoder, takeoverPrep, "alice", run},
	} {
		if _, _, err := s.FactoryTakeoverCopy(context.Background(), bad[0], bad[1], bad[2], bad[3], bad[4], bad[5]); !errors.Is(err, identity.ErrDenied) {
			t.Fatalf("invalid takeover: %v", err)
		}
	}
	if len(exec.calls) != 0 {
		t.Fatalf("invalid takeover executed: %q", exec.calls)
	}
	missing := &takeoverExec{inspect: takeoverInspect(takeoverProject, true)}
	s.Exec = missing
	if _, _, err := s.FactoryTakeoverCopy(context.Background(), takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, "alice", run); err == nil || !strings.Contains(err.Error(), "no retained work") {
		t.Fatalf("missing checkout: %v", err)
	}
}
