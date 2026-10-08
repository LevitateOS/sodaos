package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/acceptance"
)

func TestDispatch(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}); err == nil {
		t.Error("empty argv accepted")
	} else {
		var usage *acceptance.UsageError
		if !errors.As(err, &usage) || exitCode(err) != 2 {
			t.Errorf("empty argv error = %v", err)
		}
	}
	if err := run([]string{"bogus"}, &bytes.Buffer{}); err == nil {
		t.Error("unknown subcommand accepted")
	} else if exitCode(err) != 2 {
		t.Errorf("unknown subcommand exit = %d", exitCode(err))
	}
	// Every probe rejects empty args without side effects.
	for _, probe := range []string{"developer-access", "personal-git", "workload-access", "workload-exec"} {
		if err := run([]string{probe}, &bytes.Buffer{}); err == nil {
			t.Errorf("%s accepted empty args", probe)
		} else if exitCode(err) != 1 {
			t.Errorf("%s exit = %d", probe, exitCode(err))
		}
	}
	if err := run([]string{"service-https"}, &bytes.Buffer{}); err == nil {
		t.Error("service-https accepted empty args")
	} else if exitCode(err) != 2 {
		t.Errorf("service-https exit = %d", exitCode(err))
	}
}

func TestServiceHTTPSProbeFailure(t *testing.T) {
	err := run([]string{"service-https", "https://example.test/", "/absent/ca"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("missing CA accepted")
	}
	if !strings.HasPrefix(err.Error(), "HTTPS substrate check failed (") {
		t.Errorf("failure shape = %q", err.Error())
	}
	if exitCode(err) != 1 {
		t.Errorf("probe failure exit = %d", exitCode(err))
	}
}
