package control

import (
	"context"

	"github.com/levitateos/sodaos/internal/identity"
)

// CompleteEnrollment sends the native callback only to its owner's pending
// session. The normal enrollment read retains the result in encrypted custody.
func (c *Controller) CompleteEnrollment(ctx context.Context, owner int64, id, state, code string) (identity.Enrollment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.enrollments[id]
	if e == nil || e.owner != owner || e.session == nil || !codeCallbackValid(state, code) {
		return identity.Enrollment{}, identity.ErrDenied
	}
	session, ok := e.session.(identity.CodeEnrollmentSession)
	if !ok {
		return identity.Enrollment{}, identity.ErrDenied
	}
	if err := session.Complete(ctx, state, code); err != nil {
		return identity.Enrollment{}, err
	}
	result := session.Snapshot()
	result.ProviderID = e.providerID
	return result, nil
}

func codeCallbackValid(state, code string) bool {
	return state != "" && len(state) <= 256 && code != "" && len(code) <= 4096
}
