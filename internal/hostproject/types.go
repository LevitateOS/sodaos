package hostproject

import (
	"regexp"

	"github.com/levitateos/sodaos/internal/projectos"
)

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
