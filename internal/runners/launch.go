package runners

import "path/filepath"

type LaunchCommand struct {
	Path      string
	Arguments []string
	Directory string
	Home      string
}

func (native *Native) Launch(id string) (LaunchCommand, error) {
	if _, err := native.readDescriptor(id); err != nil {
		return LaunchCommand{}, err
	}
	state := native.statePath(id)
	return LaunchCommand{
		Path:      "/usr/bin/forgejo-runner",
		Arguments: []string{"forgejo-runner", "daemon", "--config", filepath.Join(state, "forgejo-runner.yml")},
		Directory: state,
		Home:      state,
	}, nil
}
