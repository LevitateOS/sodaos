package terminal

import (
	"context"
	"time"
)

// FactoryGitCaller comes only from the broker's current base factory lease
// and an exact native OCI peer match, never from the remote-helper request.
type FactoryGitCaller struct {
	Container, Project, ExecutionID string
	Actor, RepositoryID             int64
	HostPID                         int
	Deadline                        time.Time
}

// FactoryGitRuntime mediates upload-pack from disposable worker namespaces.
// The worker receives a public launch socket, never a Forgejo credential.
type FactoryGitRuntime struct {
	GitRuntime
	Resolve func(context.Context, MusePeer) (FactoryGitCaller, error)
}
