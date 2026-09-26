package runners

import "context"

type recordingCommandRunner struct{ commands []Command }

func (runner *recordingCommandRunner) Run(_ context.Context, command Command) (CommandResult, error) {
	runner.commands = append(runner.commands, command)
	if command.Name == "forgejo-runner" {
		return CommandResult{Stdout: "forgejo-runner fixture\n"}, nil
	}
	return CommandResult{Stdout: "LoadState=loaded\nActiveState=inactive\nSubState=dead\nUnitFileState=disabled\n"}, nil
}
