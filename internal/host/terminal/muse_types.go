package terminal

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

// MuseRuntime owns native caller resolution and one independently supervised
// execution per launch. Broker authority remains in its existing custody service.
type MuseRuntime struct {
	Exec            Executor
	BinaryVersion   string
	BinarySHA256    string
	FactoryRoot     string
	FactoryResolve  func(context.Context, MusePeer) (MuseFactoryCaller, error)
	Acquire         func(context.Context, identity.AcquireRequest) (identity.Lease, error)
	Attach          func(context.Context, string, identity.Binding) (identity.Delivery, error)
	End             func(context.Context, int64, string) error
	Authorize       func(context.Context, int64, string) error
	NestedAuthorize func(context.Context, int64, string) error
	Select          func(context.Context, int64, string, string) (string, error)
	mu              sync.Mutex
	nested          map[string]museNested
}

type (
	museCaller struct {
		Project, Container, Login, Home, Namespace, Child, Registration string
		Actor                                                           int64
		UID, GID, ProjectPID, NestedPID                                 int
		MuseAllowed                                                     bool
	}
	museNested struct {
		Parent, Project, Child, Namespace, Registration string
		Actor                                           int64
		PID                                             int
		Muse                                            bool
	}
)

// MusePeer is kernel-attested socket identity. PIDFD pins the original caller.
type MusePeer struct {
	PID   int
	UID   uint32
	GID   uint32
	PIDFD int
}

// MuseInvocation is prepared only after current authorization and native binding.
// Finish must confirm complete unit termination before returning the broker lease.
type MuseInvocation struct {
	Command *exec.Cmd
	Prepare func(context.Context) error
	Control func(context.Context, identity.LaunchControl) error
	Finish  func(context.Context) error
}

// MuseLaunch serves only the launch socket. Start must derive authority from Peer;
// request fields do not carry trusted identity, project, native binding or auth.
type MuseLaunch struct {
	Start    func(context.Context, MusePeer, identity.LaunchRequest, [3]*os.File) (MuseInvocation, error)
	Register func(context.Context, MusePeer, identity.NestedRegistration) error
}

// MuseFactoryCaller is produced by trusted broker lease/native OCI attestation.
// No launch request can supply this record.
type MuseFactoryCaller struct {
	Container, Project, Login, CredentialRoot string
	Actor                                     int64
	Deadline                                  time.Time
	UID, GID, HostPID                         int
}
