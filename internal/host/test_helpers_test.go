package host

import (
	projectexec "github.com/levitateos/sodaos/internal/host/project"
)

// testProject builds the privileged project runtime over a fake executor.
func testProject(exec Executor, c Config) *projectexec.Runtime {
	return projectRuntime(exec, c)
}
