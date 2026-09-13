package nativebuild

// Go build callers use the existing Python timing/reporting owner. Do not create
// a second clock, log format, summary implementation or monitoring service.
import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type BuildProgress struct {
	helper, title, label, started string
	Stderr                        io.Writer
}

func NewBuildProgress(source, title string) (*BuildProgress, error) {
	p := &BuildProgress{helper: filepath.Join(source, "scripts/build_progress.py"), title: title, Stderr: os.Stderr}
	if os.Getenv("SODA_BUILD_START_NS") == "" {
		now, e := p.clock()
		if e != nil {
			return nil, e
		}
		if e = os.Setenv("SODA_BUILD_START_NS", now); e != nil {
			return nil, e
		}
	}
	return p, nil
}
func (p *BuildProgress) clock() (string, error) {
	b, e := exec.Command("python3", p.helper, "clock").Output()
	if e != nil {
		return "", e
	}
	s := strings.TrimSpace(string(b))
	if _, e = strconv.ParseInt(s, 10, 64); e != nil {
		return "", e
	}
	return s, nil
}
func (p *BuildProgress) helperRun(args ...string) error {
	cmd := exec.Command("python3", append([]string{p.helper}, args...)...)
	cmd.Stdout = p.Stderr
	cmd.Stderr = p.Stderr
	return cmd.Run()
}
func (p *BuildProgress) CreateLog(path string) error {
	if err := p.helperRun("create-log", path); err != nil {
		return err
	}
	// A helper subprocess cannot export its environment back to this caller.
	if os.Getenv("SODA_BUILD_TIMING_LOG") == "" {
		return os.Setenv("SODA_BUILD_TIMING_LOG", path)
	}
	return nil
}
func (p *BuildProgress) Next(label string) error {
	if e := p.End(nil); e != nil {
		return e
	}
	started, e := p.clock()
	if e != nil {
		return e
	}
	p.label, p.started = label, started
	return p.helperRun("emit", "START", label)
}
func (p *BuildProgress) End(err error) error {
	if p.label == "" {
		return nil
	}
	outcome := "DONE"
	if err != nil {
		outcome = "FAILED"
	}
	code := BuildExitCode(err)
	if code == 130 || code == 143 {
		outcome = "CANCELLED"
	}
	label, started := p.label, p.started
	p.label, p.started = "", ""
	return p.helperRun("emit", outcome, label, started)
}
func (p *BuildProgress) Finish(err error) error {
	end := p.End(err)
	return errors.Join(end, p.helperRun("finish", p.title, strconv.Itoa(BuildExitCode(err))))
}
func BuildExitCode(err error) int {
	if err == nil {
		return 0
	}
	var status interface{ ExitCode() int }
	if errors.As(err, &status) && status.ExitCode() > 0 {
		return status.ExitCode()
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}

// BuildExecution owns only local command IO/environment; it is not a job runner.
// The entrypoint's existing build_progress.py supervisor owns process-group
// cancellation. No command arguments are copied into timing records.
type BuildExecution struct {
	Context context.Context
	Log     io.Writer
	Output  io.Writer // optional public command output; captures remain isolated
}

func (b BuildExecution) command(dir, name string, args ...string) *exec.Cmd {
	executable := name
	if name == "go" {
		executable = filepath.Join(runtime.GOROOT(), "bin", "go")
	}
	cmd := exec.CommandContext(b.Context, executable, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=-mod=readonly", "CGO_ENABLED=0", "PATH="+filepath.Join(runtime.GOROOT(), "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Stderr = b.Log
	// Build arguments are public; raw provider/provisioning outputs do not enter
	// this producer. Keep tool logs separate from the progress-only timing log.
	fmt.Fprintf(b.Log, "\n$ %s %s\n", name, strings.Join(args, " "))
	return cmd
}
func (b BuildExecution) Execute(dir, name string, args ...string) error {
	cmd := b.command(dir, name, args...)
	cmd.Stdout = b.Log
	if b.Output != nil {
		cmd.Stdout = io.MultiWriter(b.Log, b.Output)
	}
	if e := cmd.Run(); e != nil {
		return fmt.Errorf("%s failed; retain attempt and inspect build.log: %w", name, errors.Join(b.Context.Err(), e))
	}
	return nil
}
func (b BuildExecution) Capture(dir, name string, args ...string) (string, error) {
	data, e := b.command(dir, name, args...).Output()
	if e != nil {
		return "", fmt.Errorf("%s observation failed; retain attempt and inspect build.log: %w", name, errors.Join(b.Context.Err(), e))
	}
	return strings.TrimSpace(string(data)), nil
}
