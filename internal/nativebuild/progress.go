package nativebuild

// Native monotonic progress. Legacy callers can share the same monotonic epoch
// and log until their media/producer paths are retired.
import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type BuildProgress struct {
	title, label, path string
	origin, started    time.Duration
	Now                func() time.Duration
	Stderr             io.Writer
	finished           bool
	phase              string
	phaseStarted       time.Duration
}

func NewBuildProgress(source, title string) (*BuildProgress, error) {
	p := &BuildProgress{title: title, Now: monotonic, Stderr: os.Stderr}
	p.origin = p.Now()
	if raw := os.Getenv("SODA_BUILD_START_NS"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 || time.Duration(n) > p.origin {
			return nil, errors.New("invalid inherited timing origin")
		}
		p.origin = time.Duration(n)
	} else {
		os.Setenv("SODA_BUILD_START_NS", strconv.FormatInt(int64(p.origin), 10))
	}
	return p, nil
}

func duration(d time.Duration) string {
	s := max(int64(d/time.Second), 0)
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s/60%60, s%60)
}

func (p *BuildProgress) emit(kind, text string) error {
	line := fmt.Sprintf("%-8s %s\n", kind, text)
	_, err := io.WriteString(p.Stderr, line)
	if p.path != "" {
		f, e := os.OpenFile(p.path, os.O_WRONLY|os.O_APPEND|syscall.O_NOFOLLOW, 0)
		if e != nil {
			return errors.Join(err, e)
		}
		_, e = io.WriteString(f, line)
		err = errors.Join(err, e, f.Close())
	}
	return err
}

func (p *BuildProgress) CreateLog(path string) error {
	if inherited := os.Getenv("SODA_BUILD_TIMING_LOG"); inherited != "" && os.Getenv("SODA_BUILD_CHILD") == "1" {
		p.path = inherited
		return nil
	}
	if err := WriteNew(path, nil, 0o600); err != nil {
		return err
	}
	p.path = path
	return p.emit("LOG", path)
}

func (p *BuildProgress) Phase(label string) error {
	if p.finished || strings.ContainsAny(label, "\r\n") {
		return errors.New("invalid progress transition")
	}
	if err := errors.Join(p.End(nil), p.EndPhase(nil)); err != nil {
		return err
	}
	p.phase, p.phaseStarted = label, p.Now()
	return p.emit("START", label)
}

func (p *BuildProgress) EndPhase(err error) error {
	if p.phase == "" {
		return nil
	}
	kind := "DONE"
	if err != nil {
		kind = "FAILED"
	}
	if code := BuildExitCode(err); code == 130 || code == 143 {
		kind = "CANCELLED"
	}
	label := p.phase
	p.phase = ""
	return p.emit(kind, label+" | phase "+duration(p.Now()-p.phaseStarted)+" | total "+duration(p.Now()-p.origin))
}

func (p *BuildProgress) Next(label string) error {
	if p.finished || strings.ContainsAny(label, "\r\n") {
		return errors.New("invalid progress transition")
	}
	if err := p.End(nil); err != nil {
		return err
	}
	p.label, p.started = label, p.Now()
	return p.emit("START", label)
}

func (p *BuildProgress) End(err error) error {
	if p.label == "" {
		return nil
	}
	kind := "DONE"
	if err != nil {
		kind = "FAILED"
	}
	if code := BuildExitCode(err); code == 130 || code == 143 {
		kind = "CANCELLED"
	}
	label := p.label
	p.label = ""
	now := p.Now()
	return p.emit(kind, label+" | section "+duration(now-p.started)+" | total "+duration(now-p.origin))
}

func (p *BuildProgress) Finish(err error) error {
	if p.finished {
		return nil
	}
	p.finished = true
	end := errors.Join(p.End(err), p.EndPhase(err))
	if os.Getenv("SODA_BUILD_CHILD") == "1" {
		return end
	}
	if p.path != "" {
		data, e := os.ReadFile(p.path)
		end = errors.Join(end, e)
		_, e = fmt.Fprintln(p.Stderr, "\nSECTION SUMMARY")
		end = errors.Join(end, e)
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "DONE ") || strings.HasPrefix(line, "FAILED ") || strings.HasPrefix(line, "CANCELLED ") {
				_, e = fmt.Fprintln(p.Stderr, "  "+line)
				end = errors.Join(end, e)
			}
		}
	}
	kind := "SUCCESS"
	code := BuildExitCode(errors.Join(err, end))
	if code != 0 {
		kind = "FAILED"
	}
	if code == 130 || code == 143 {
		kind = "CANCELLED"
	}
	return errors.Join(end, p.emit(kind, p.title+" | total "+duration(p.Now()-p.origin)+fmt.Sprintf(" | exit %d", code)))
}

func BuildExitCode(err error) int {
	if err == nil {
		return 0
	}
	var status interface{ ExitCode() int }
	if errors.As(err, &status) && status.ExitCode() > 0 {
		return status.ExitCode()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
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

// resolveBuildTool resolves a build tool at run time. The GOTOOLCHAIN pin in
// the command environment forces the exact compiler version; PATH decides
// which installation provides it. A build-time GOROOT would answer a run-time
// question with a stale path once the binary moves machines.
func resolveBuildTool(name string) string {
	if name == "go" {
		if path, err := exec.LookPath("go"); err == nil {
			return path
		}
	}
	return name
}

func (b BuildExecution) command(dir, name string, args ...string) *exec.Cmd {
	executable := resolveBuildTool(name)
	cmd := exec.CommandContext(b.Context, executable, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN="+runtime.Version(), "GOWORK=off", "GOFLAGS=-mod=readonly", "CGO_ENABLED=0")
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
