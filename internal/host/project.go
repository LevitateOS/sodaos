package host

import (
	projectexec "github.com/levitateos/sodaos/internal/host/project"
)

// projectRuntime maps daemon network/image config onto the privileged executor.
func projectRuntime(exec Executor, c Config) *projectexec.Runtime {
	return &projectexec.Runtime{
		Exec: exec,
		Config: projectexec.Config{
			Image:   c.Image,
			Network: c.Network,
			Subnet:  c.Subnet,
			Bridge:  c.Bridge,
		},
	}
}
