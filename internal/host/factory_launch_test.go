package host

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/project"
)

// The manifest/phase rule is enforced before the daemon ever sees the
// request: an implement launch without its exact manifest bytes never
// reaches the host boundary.
func TestFactoryLaunchRefusesManifestViolationBeforeIO(t *testing.T) {
	c := NewClient("unused")
	c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid launch reached host")
		return nil, nil
	})
	prompt := []byte("trivial coding task")
	run := project.FactoryRun{
		Deadline: time.Now().Add(time.Hour), Actor: 1001,
		ID: strings.Repeat("a", 32), Project: "p" + strings.Repeat("b", 24),
		Role: project.RoleCoder, Preparation: "f" + strings.Repeat("c", 24),
		Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1",
		Assignment: project.FactoryPromptDigest(prompt), SourceCommit: strings.Repeat("e", 40),
		Connection:     "subscription",
		ExecutionPhase: project.ExecutionImplement, ManifestDigest: strings.Repeat("d", 64),
	}
	launch := project.FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("f", 64)}
	if _, err := c.FactoryLaunch(context.Background(), launch); err == nil {
		t.Fatal("manifest-free implement launch sent to host")
	}
}

// D13 wire scope: the reassess-only 320 KiB exception is enforced before
// the daemon ever sees the request; other phases keep 64 KiB.
func TestFactoryLaunchRefusesPromptBoundViolationBeforeIO(t *testing.T) {
	c := NewClient("unused")
	c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid launch reached host")
		return nil, nil
	})
	launchFor := func(phase string, size int) project.FactoryLaunch {
		prompt := make([]byte, size)
		for i := range prompt {
			prompt[i] = 'p'
		}
		run := project.FactoryRun{
			Deadline: time.Now().Add(time.Hour), Actor: 1001,
			ID: strings.Repeat("a", 32), Project: "p" + strings.Repeat("b", 24),
			Role: project.RoleCoder, Preparation: "f" + strings.Repeat("c", 24),
			Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1",
			Assignment: project.FactoryPromptDigest(prompt), SourceCommit: strings.Repeat("e", 40),
			Connection:     "subscription",
			ExecutionPhase: phase,
		}
		return project.FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("f", 64)}
	}
	for name, launch := range map[string]project.FactoryLaunch{
		"implement at 320 KiB": launchFor(project.ExecutionImplement, project.MaxReassessPrompt),
		"reassess over bound":  launchFor(project.ExecutionReassess, project.MaxReassessPrompt+1),
	} {
		if _, err := c.FactoryLaunch(context.Background(), launch); err == nil {
			t.Fatalf("oversized launch %s sent to host", name)
		}
	}
}
