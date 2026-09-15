package host

// testDaemon builds a Daemon with Project wired to the same Exec/Config.
func testDaemon(exec Executor, c Config) Daemon {
	return Daemon{Exec: exec, Config: c, Project: NewProject(exec, c)}
}

func testDaemonPtr(exec Executor, c Config) *Daemon {
	d := testDaemon(exec, c)
	return &d
}
