package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type executorFunc func(context.Context, []byte, string, ...string) ([]byte, error)

func (f executorFunc) Run(c context.Context, b []byte, n string, a ...string) ([]byte, error) {
	return f(c, b, n, a...)
}

func testRun() factory.Run {
	now := time.Now()
	id := strings.Repeat("a", 32)
	return factory.Run{ID: id, AttemptID: strings.Repeat("b", 32), Role: factory.Implementation, InputSHA: strings.Repeat("c", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("d", 64), Harness: "codex-test", Model: "test", Resources: []factory.Resource{{Kind: "workspace", Name: factory.ResourceName(id, "workspace"), ID: strings.Repeat("e", 64)}}}
}

func TestCleanupRequiresRecordedOwnership(t *testing.T) {
	for _, owner := range []string{"other", strings.Repeat("a", 32)} {
		t.Run(owner, func(t *testing.T) {
			r := testRun()
			var removed []string
			w := Runtime{Exec: executorFunc(func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
				switch args[1] {
				case "exists":
					return nil, nil
				case "inspect":
					return json.Marshal(map[string]string{"id": r.Resources[0].ID, "name": r.Resources[0].Name, "owner": owner})
				default:
					removed = append(removed, strings.Join(args, " "))
					return nil, nil
				}
			})}
			err := w.Cleanup(context.Background(), &r)
			if owner != r.ID {
				if err == nil || len(removed) != 0 || r.CleanupComplete {
					t.Fatal("foreign resource was accepted")
				}
				return
			}
			if err != nil || !r.CleanupComplete {
				t.Fatalf("owned cleanup: %v", err)
			}
			if len(removed) != 2 || removed[0] != "stop --time 3 "+r.Resources[0].ID || removed[1] != "rm "+r.Resources[0].ID {
				t.Fatalf("unexpected deletion: %v", removed)
			}
		})
	}
}

func TestCleanupDistinguishesAbsenceFromEngineFailure(t *testing.T) {
	for _, code := range []int{1, 125} {
		r := testRun()
		calls := 0
		w := Runtime{Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
			calls++
			return nil, &ExitError{Code: code}
		})}
		err := w.Cleanup(context.Background(), &r)
		if calls != 1 {
			t.Fatal("cleanup continued after failed observation")
		}
		if (err == nil) != (code == 1) || r.CleanupComplete != (code == 1) {
			t.Fatalf("exit %d: %v", code, err)
		}
	}
}

func TestLaunchRequiresSubscriptionBeforeCallingModel(t *testing.T) {
	calls := 0
	w := Runtime{Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		calls++
		return nil, errors.New("logged out")
	})}
	if _, err := w.Launch(context.Background(), testRun()); err == nil || calls != 1 {
		t.Fatal("unauthenticated launch reached model")
	}
}

func TestLaunchRejectsMissingResultAfterSuccessfulProcess(t *testing.T) {
	calls := 0
	w := Runtime{Exec: executorFunc(func(_ context.Context, _ []byte, _ string, args ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte("Logged in using ChatGPT"), nil
		}
		if calls == 2 {
			command := strings.Join(args, " ")
			if !strings.Contains(command, "exec --interactive") {
				t.Fatal("prompt stdin is not forwarded")
			}
			if !strings.Contains(command, `sqlite_home="/workspace/.codex-state"`) || !strings.Contains(command, `log_dir="/workspace/.codex-log"`) {
				t.Fatal("shared conversation state")
			}
			return nil, nil
		}
		return nil, errors.New("missing")
	})}
	if _, err := w.Launch(context.Background(), testRun()); err == nil {
		t.Fatal("successful exit accepted without result")
	}
}

func TestCleanupKeepsDependenciesWhenWorkerCannotBeObserved(t *testing.T) {
	r := testRun()
	c := Config{Image: r.Image, HarnessVersion: "test", Model: r.Model}
	r.Resources = nil
	c.Bind(&r)
	calls := 0
	w := Runtime{Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		calls++
		return nil, &ExitError{Code: 125}
	})}
	if err := w.Cleanup(context.Background(), &r); err == nil || calls != 1 || r.CleanupComplete {
		t.Fatal("cleanup removed dependencies after worker failure")
	}
}
