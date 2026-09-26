package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// Executor is the native process boundary used by runtime contract checks.
type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}

type Native struct{}

type ExitError struct{ Code int }

func (e *ExitError) Error() string {
	return fmt.Sprintf("native workspace command failed (exit %d)", e.Code)
}

// Native retains bounded output in memory. It never logs command arguments,
// provider responses or credential-bearing native diagnostics.
func (Native) Run(ctx context.Context, input []byte, executable string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdin = bytes.NewReader(input)
	var output boundedOutput
	var diagnostics boundedOutput
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		var exited *exec.ExitError
		if errors.As(err, &exited) {
			return nil, &ExitError{Code: exited.ExitCode()}
		}
		return nil, errors.New("native workspace command could not start")
	}
	return output.Bytes(), nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4<<20 {
		return 0, errors.New("workspace output limit exceeded")
	}
	return b.Buffer.Write(p)
}
