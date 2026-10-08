// Outside support only: runs the installed client probes against an
// authorized target; no product scenarios.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/levitateos/sodaos/internal/acceptance"
)

// toolUsage names every installed client probe.
const toolUsage = "developer-access|personal-git|service-https|workload-access|workload-exec"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

// exitCode preserves argparse's exit 2 for usage rejections.
func exitCode(err error) int {
	var usage *acceptance.UsageError
	if errors.As(err, &usage) {
		return 2
	}
	return 1
}

func run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return dispatchUsage("subcommand required")
	}
	switch args[0] {
	case "developer-access":
		return acceptance.RunDeveloperAccess(args[1:], stdout)
	case "personal-git":
		return acceptance.RunPersonalGit(args[1:], stdout)
	case "service-https":
		return acceptance.RunServiceHTTPS(args[1:], stdout)
	case "workload-access":
		return acceptance.RunWorkloadAccess(args[1:], stdout)
	case "workload-exec":
		return acceptance.RunWorkloadExec(args[1:], stdout)
	default:
		return dispatchUsage("unknown subcommand")
	}
}

func dispatchUsage(problem string) error {
	return &acceptance.UsageError{Probe: "soda-installed-probes", Message: problem + "; want " + toolUsage}
}
