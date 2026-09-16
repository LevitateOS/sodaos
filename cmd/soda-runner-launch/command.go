package main

import (
	"errors"
	"fmt"
	"os/user"
	"strings"

	"github.com/levitateos/sodaos/internal/runners"
)

var errUsage = errors.New("usage: soda-runner-launch <runner-id>")

// currentUsername is a stub point for tests; production resolves the
// kernel-reported caller.
var currentUsername = func() (string, error) {
	account, err := user.Current()
	if err != nil {
		return "", err
	}
	return account.Username, nil
}

// verifyRunnerCaller binds the launcher to its systemd identity: the unit
// runs as soda-runner-<id>, so only that account may launch the runner. This
// keeps one runner's caller (or a manual root invocation) from executing
// another runner's daemon state.
func verifyRunnerCaller(id string) error {
	expected, err := runners.AccountName(id)
	if err != nil {
		return err
	}
	username, err := currentUsername()
	if err != nil {
		return fmt.Errorf("identify runner caller: %w", err)
	}
	if username != expected {
		return fmt.Errorf("runner %q must be launched by %q", id, expected)
	}
	return nil
}

func execute(args, environment []string, launch func(string) (runners.LaunchCommand, error), replace func(runners.LaunchCommand, []string) error) error {
	if len(args) != 1 {
		return errUsage
	}
	if err := verifyRunnerCaller(args[0]); err != nil {
		return err
	}
	command, err := launch(args[0])
	if err != nil {
		return err
	}
	return replace(command, runnerEnvironment(environment, command.Home))
}

func runnerEnvironment(inherited []string, home string) []string {
	// syscall.Exec, unlike os/exec.Cmd, does not deduplicate environment entries.
	environment := make([]string, 0, len(inherited)+1)
	for _, entry := range inherited {
		if !strings.HasPrefix(entry, "HOME=") {
			environment = append(environment, entry)
		}
	}
	return append(environment, "HOME="+home)
}
