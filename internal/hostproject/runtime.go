// Package hostproject runs privileged project environment operations. It does
// not own HTTP admission, Tailnet policy, terminal attach or runner protocols.
package hostproject

import (
	"context"
	"regexp"

	"github.com/levitateos/sodaos/internal/projectos"
)

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}

// Config holds the project-network image and bridge fields needed to create and
// inspect environments. Tailnet companion settings stay on the host Daemon.
type Config struct {
	Image   string
	Network string
	Subnet  string
	Bridge  string
}

type Runtime struct {
	Exec   Executor
	Config Config
}

var (
	projectID   = regexp.MustCompile(`^p[0-9a-f]{24}$`)
	loginName   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)
	imageID     = regexp.MustCompile(`^(?:sha256:)?[0-9a-f]{64}$`)
	containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Create struct {
	Profile *projectos.Profile
	ID      string
	Owner   int64
}

type Account struct {
	Project  string
	Login    string
	Identity int64
	Keys     []string
}

type Environment struct {
	Image   string
	Profile *projectos.Profile
	ID      string
	IP      string
	Running bool
}

type Connection struct {
	Environment Environment
	HostKey     string
	Fingerprint string
}

type Lifecycle struct {
	Project string
	Action  string
}

type LifecycleState struct {
	Environment Environment
	BootEnabled bool
}

type AccessKeys struct {
	Project  string
	Login    string
	Identity int64
	Revision string
	Keys     []string
	Apply    bool
}

type AccessKeyState struct {
	Revision string
	Keys     []string
}

type OSRelease struct {
	ID      string
	Version string
	Name    string
}

type OSObservation struct {
	Environment Environment
	Release     *OSRelease
	Unavailable bool
}

func (r *Runtime) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return r.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}
