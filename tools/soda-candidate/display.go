// display renders the admitted controller's stderr event protocol as a
// step-by-step table with wall times. It never drives the build itself.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// event is one parsed controller stderr line. Labels carry no newlines by
// controller contract, so plain line splitting is the whole protocol.
type event struct {
	kind     string
	label    string
	phaseDur string
	totalDur string
	path     string
	reason   string
}

// parseEvent splits a controller line into START, DONE, FAILED, CANCELLED,
// CANDIDATE, MEDIA, FINAL or LOG. Anything else is child log traffic.
func parseEvent(line string) (event, bool) {
	kind, rest, _ := strings.Cut(line, " ")
	switch kind {
	case "START":
		return parseStartEvent(kind, rest)
	case "DONE", "FAILED", "CANCELLED":
		return parseDoneEvent(kind, rest)
	case "CANDIDATE", "MEDIA", "FINAL", "LOG":
		return parseArtifactEvent(kind, rest)
	default:
		return event{}, false
	}
}

func parseStartEvent(kind, rest string) (event, bool) {
	if rest == "" {
		return event{}, false
	}
	return event{kind: kind, label: rest}, true
}

func parseDoneEvent(kind, rest string) (event, bool) {
	parts := strings.Split(rest, " | ")
	if len(parts) < 1 || parts[0] == "" {
		return event{}, false
	}
	e := event{kind: kind, label: parts[0]}
	for _, p := range parts[1:] {
		applyDurationPart(&e, p)
	}
	return e, true
}

func applyDurationPart(e *event, p string) {
	if d, ok := strings.CutPrefix(p, "phase "); ok {
		e.phaseDur = d
	} else if d, ok := strings.CutPrefix(p, "section "); ok {
		e.phaseDur = d
	} else if d, ok := strings.CutPrefix(p, "total "); ok {
		e.totalDur = d
	} else if r, ok := strings.CutPrefix(p, "reason "); ok {
		e.reason = r
	}
}

// hostArtifactPath maps a worker sandbox path back onto this run's host
// output directory by matching the run directory name. The worker never
// knows host paths, so the wrapper translates: everything from the run
// directory name onward is re-rooted onto the host output directory.
// Anything else passes through unchanged.
func hostArtifactPath(outDir, sandbox string) string {
	base := filepath.Base(outDir)
	if outDir == "" || base == "" || base == "/" || base == "." {
		return sandbox
	}
	i := strings.LastIndex(sandbox, base)
	if i < 0 {
		return sandbox
	}
	return filepath.Join(outDir, strings.TrimPrefix(sandbox[i+len(base):], "/"))
}

func parseArtifactEvent(kind, rest string) (event, bool) {
	if rest == "" {
		return event{}, false
	}
	return event{kind: kind, path: rest}, true
}

// phase tracks one STARTed controller phase until its DONE/FAILED line.
type phase struct {
	label   string
	state   string
	dur     string
	started time.Time
}

// spinner frames animate the running row on each ticker redraw.
var spinner = []rune{'|', '/', '-', '\\'}

// renderer shows phases and recent log traffic on a terminal, or
// timestamped passthrough when output is piped. Redraws truncate to the
// terminal width so cursor math stays exact.
type renderer struct {
	mu           sync.Mutex
	w            io.Writer
	tty          bool
	width        int
	start        time.Time
	phases       []phase
	log          []string
	arts         []string
	drawn        int
	ticker       *time.Ticker
	stop         chan struct{}
	outDir       string
	failedLabel  string
	failedReason string
}

// SetOutDir tells the renderer where this run writes on the host, so
// sandbox paths translate and the failure panel can name the build log.
func (r *renderer) SetOutDir(outDir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outDir = outDir
}

const logViewport = 8

func newRenderer(w io.Writer, tty bool, width int) *renderer {
	if width < 20 {
		width = 80
	}
	return &renderer{w: w, tty: tty, width: width, start: time.Now()}
}

// feed consumes one child stderr line.
func (r *renderer) feed(line string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := parseEvent(line)
	if !ok {
		return r.noteLocked(line)
	}
	switch e.kind {
	case "START":
		r.phases = append(r.phases, phase{label: e.label, state: "run", started: time.Now()})
	case "DONE", "FAILED", "CANCELLED":
		state := map[string]string{"DONE": "ok", "FAILED": "fail", "CANCELLED": "stop"}[e.kind]
		r.closePhase(e.label, state, e.phaseDur)
		// The first FAILED line is the most specific one: inner steps
		// fail before their parents, so it names the root cause.
		if e.kind == "FAILED" && r.failedLabel == "" {
			r.failedLabel, r.failedReason = e.label, e.reason
		}
	case "CANDIDATE", "MEDIA", "FINAL", "LOG":
		r.arts = append(r.arts, e.kind+" "+hostArtifactPath(r.outDir, e.path))
	}
	if !r.tty {
		shown := line
		switch e.kind {
		case "CANDIDATE", "MEDIA", "FINAL", "LOG":
			// Piped output is a working log: host paths only.
			if t := hostArtifactPath(r.outDir, e.path); t != e.path {
				shown = e.kind + " " + t
			}
		}
		_, err := fmt.Fprintf(r.w, "[%s] %s\n", wallSince(r.start), shown)
		return err
	}
	return r.draw()
}

// note records child log traffic: viewport on a terminal, passthrough piped.
func (r *renderer) note(line string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.noteLocked(line)
}

func (r *renderer) noteLocked(line string) error {
	if !r.tty {
		_, err := fmt.Fprintf(r.w, "[%s] %s\n", wallSince(r.start), line)
		return err
	}
	r.log = append(r.log, line)
	if len(r.log) > logViewport {
		r.log = r.log[len(r.log)-logViewport:]
	}
	return r.draw()
}

func (r *renderer) closePhase(label, state, dur string) {
	for i := len(r.phases) - 1; i >= 0; i-- {
		if r.phases[i].label == label && r.phases[i].state == "run" {
			r.phases[i].state, r.phases[i].dur = state, dur
			return
		}
	}
	r.phases = append(r.phases, phase{label: label, state: state, dur: dur})
}

func phaseMark(state string) string {
	switch state {
	case "ok":
		return "[ok]  "
	case "fail":
		return "[FAIL]"
	case "stop":
		return "[stop]"
	default:
		return "[..]  "
	}
}

// draw redraws the whole table in place. Callers hold r.mu.
func (r *renderer) draw() error {
	lines := r.buildLines(time.Now())
	var errs []error
	if r.drawn > 0 {
		_, err := fmt.Fprintf(r.w, "\x1b[%dA", r.drawn)
		errs = append(errs, err)
	}
	for _, l := range lines {
		_, err := fmt.Fprintf(r.w, "\x1b[K%s\n", truncate(l, r.width))
		errs = append(errs, err)
	}
	r.drawn = len(lines)
	return errors.Join(errs...)
}

func (r *renderer) buildLines(now time.Time) []string {
	lines := []string{fmt.Sprintf("soda-candidate | elapsed %s", wallDur(now.Sub(r.start)))}
	for _, l := range r.log {
		lines = append(lines, "  "+l)
	}
	if len(r.phases) > 0 {
		lines = append(lines, "  steps:")
	}
	for _, p := range r.phases {
		lines = append(lines, phaseLine(p, now))
	}
	if len(r.arts) > 0 {
		lines = append(lines, "  outputs:")
	}
	for _, a := range r.arts {
		lines = append(lines, "  "+a)
	}
	return lines
}

func phaseLine(p phase, now time.Time) string {
	if p.state == "run" {
		frame := spinner[int(now.Sub(p.started)/(250*time.Millisecond))%len(spinner)]
		return fmt.Sprintf("  [%c] %s  (live %s)", frame, p.label, wallDur(now.Sub(p.started)))
	}
	line := fmt.Sprintf("  %s %s", phaseMark(p.state), p.label)
	if p.dur != "" {
		line += "  (" + p.dur + ")"
	}
	return line
}

// startTicker refreshes the elapsed header while the controller runs.
func (r *renderer) startTicker() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.tty || r.ticker != nil {
		return
	}
	r.ticker = time.NewTicker(250 * time.Millisecond)
	r.stop = make(chan struct{})
	go func() {
		for {
			select {
			case <-r.ticker.C:
				r.mu.Lock()
				_ = r.draw()
				r.mu.Unlock()
			case <-r.stop:
				return
			}
		}
	}()
}

// stopTicker halts refresh before the closing summary.
func (r *renderer) stopTicker() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ticker == nil {
		return
	}
	r.ticker.Stop()
	close(r.stop)
	r.ticker = nil
}

// finish prints the closing summary and the controller exit code meaning.
func (r *renderer) finish(code int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tty {
		if err := r.draw(); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(r.w, "soda-candidate: finished in %s with exit %d%s\n", wallSince(r.start), code, exitMeaning(code))
	if err == nil && code != 0 && r.failedLabel != "" {
		err = r.printWhyPanelLocked()
	}
	return err
}

// printWhyPanelLocked names the root-cause step, its reason, and the host
// build log. It runs for terminals and pipes alike, after the closing
// summary, so the cause is the last thing the operator sees.
func (r *renderer) printWhyPanelLocked() error {
	cause := r.failedReason
	if cause == "" {
		cause = "see the build log"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  why: %s\n  cause: %s\n", r.failedLabel, cause)
	if r.outDir != "" {
		fmt.Fprintf(&b, "  log: %s\n", filepath.Join(r.outDir, "logs/build.log"))
	}
	_, err := io.WriteString(r.w, b.String())
	return err
}

func exitMeaning(code int) string {
	switch code {
	case 0:
		return " (development output ready; not release-qualified)"
	case 2:
		return " (incomplete: qualification passed but final signing is not connected; no qualified release)"
	default:
		return ""
	}
}

func wallDur(d time.Duration) string {
	s := int64(d / time.Second)
	if s < 0 {
		s = 0
	}
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s/60%60, s%60)
}

func wallSince(start time.Time) string {
	return wallDur(time.Since(start))
}

func truncate(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

func isTerminal(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}

func termWidth(f *os.File) int {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 {
		return 80
	}
	return int(ws.Col)
}
