package api

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Admitted operation lifetimes across the Fountain→Soda→host boundary.
//
// Fountain bounds ordinary extension responses at 30 seconds and the
// browser may disconnect at any time; neither bound is the operation's
// lifetime. Once Soda admits an operation it runs detached from the
// request under one of the bounds below, so response cancellation cannot
// strand result recording. The host daemon bounds one native operation at
// three minutes and the Unix client at four; the Soda-side bounds stay
// outside the daemon bound with margin for result recording, and outside
// the coordinator's admitted bounds for stop and start. There is no
// asynchronous job service: the handler runs the admitted operation to
// its recorded outcome, and refresh/inspection observes the truth after.
const (
	// createOperationTimeout covers native provisioning (daemon 3m) plus
	// transport and result-recording margin.
	createOperationTimeout = 3*time.Minute + 30*time.Second
	// stopOperationTimeout covers coordinator withdrawal/run-stop (10m),
	// the maintenance hold and the native stop (daemon 3m).
	stopOperationTimeout = 14 * time.Minute
	// startOperationTimeout covers the native start (daemon 3m) and start
	// verification (5m) with margin.
	startOperationTimeout = 9 * time.Minute
	// reconcileRecordTimeout bounds the local result write that finishes
	// a retained reservation from host evidence.
	reconcileRecordTimeout = 10 * time.Second
)

// operationContext detaches one admitted operation from request
// cancellation (Fountain bound, browser disconnect) while keeping it
// bounded. Admission checks (authority, session, visibility) always run
// on the live request context before this detachment.
func operationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}

// provisioningConfirmed reports whether host evidence already proves the
// admitted creation: a running endpoint carrying the reserved profile.
// This is the same bar as a successful Create call, which the host
// client confirms by running state, valid address and profile match.
func provisioningConfirmed(p store.Project, env project.Environment) bool {
	return !p.Ready && env.Running && env.IP != "" && !environmentProfileMismatch(p, env)
}

// reconcileProvisioning finishes a retained reservation from host
// evidence. It records through a bounded detached context so the finish
// survives response cancellation, and leaves anything unconfirmed
// untouched for inspection. It never deletes or recreates native roots.
func (s *API) reconcileProvisioning(ctx context.Context, p store.Project, env project.Environment) (store.Project, bool) {
	if !provisioningConfirmed(p, env) {
		return p, false
	}
	record, cancel := context.WithTimeout(context.WithoutCancel(ctx), reconcileRecordTimeout)
	defer cancel()
	if err := s.Store.MarkReady(record, p.ID, env.IP); err != nil {
		return p, false
	}
	p.Ready, p.IP = true, env.IP
	return p, true
}
