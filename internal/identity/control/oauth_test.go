package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

type codeSession struct {
	testSession
	calls int
}

func (s *codeSession) Complete(_ context.Context, state, code string) error {
	s.calls++
	if state != "synthetic-state" || code != "synthetic-code" {
		return identity.ErrDenied
	}
	return nil
}

func TestCodeEnrollmentOwner(t *testing.T) {
	c, _, _, _ := controllerFixture(t)
	session := &codeSession{}
	c.enrollments["code-enrollment"] = &enrollment{owner: 1, providerID: identity.Codex, session: session}
	if _, err := c.CompleteEnrollment(t.Context(), 2, "code-enrollment", "synthetic-state", "synthetic-code"); !errors.Is(err, identity.ErrDenied) || session.calls != 0 {
		t.Fatal("another owner reached native code exchange")
	}
	result, err := c.CompleteEnrollment(t.Context(), 1, "code-enrollment", "synthetic-state", "synthetic-code")
	if err != nil || session.calls != 1 || result.State != "completed" || result.Connection != nil {
		t.Fatal("owned callback did not await encrypted retention", err)
	}
}

func TestDeviceEnrollmentRejectsCodeCompletion(t *testing.T) {
	c, _, _, _ := controllerFixture(t)
	c.enrollments["device-enrollment"] = &enrollment{owner: 1, providerID: identity.Codex, session: &testSession{}}
	if _, err := c.CompleteEnrollment(t.Context(), 1, "device-enrollment", "synthetic-state", "synthetic-code"); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("device enrollment accepted authorization code")
	}
}
