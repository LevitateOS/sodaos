// display renders the admitted controller's stderr event protocol as a
// step-by-step table with wall times. It never drives the build itself.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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
}

// parseEvent splits a controller line into START, DONE, FAILED, CANCELLED,
// CANDIDATE, MEDIA, FINAL or LOG. Anything else is child log traffic.
func parseEvent(line string) (event, bool) {
	kind, rest, _ := strings.Cut(line, " ")
	switch kind {
	case "START":
		if rest == "" {
			return event{}, false
		}
		return event{kind: kind, label: rest}, true
	case "DONE", "FAILED", "CANCELLED":
		parts := strings.Split(rest, " | ")
		if len(parts) < 1 || parts[0] == "" {
			return event{}, false
		}
		e := event{kind: kind, label: parts[0]}
		for _, p := range parts[1:] {
			if d, ok := strings.CutPrefix(p, "phase "); ok {
				e.phaseDur = d
			} else if d, ok := strings.CutPrefix(p, "section "); ok {
				e.phaseDur = d
			} else if d, ok := strings.CutPrefix(p, "total "); ok {
				e.totalDur = d
			}
		}
		return e, true
	case "CANDIDATE", "MEDIA", "FINAL", "LOG":
		if rest == "" {
			return event{}, false
		}
		return event{kind: kind, path: rest}, true
	default:
		return event{}, false
	}
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
	mu     sync.Mutex
	w      io.Writer
	tty    bool
	width  int
	start  time.Time
	phases []phase
	log    []string
	arts   []string
	drawn  int
	ticker *time.Ticker
	stop   chan struct{}
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
	case "CANDIDATE", "MEDIA", "FINAL", "LOG":
		r.arts = append(r.arts, e.kind+" "+e.path)
	}
	if !r.tty {
		_, err := fmt.Fprintf(r.w, "[%s] %s\n", wallSince(r.start), line)
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
	now := time.Now()
	lines := []string{fmt.Sprintf("soda-iso | elapsed %s", wallDur(now.Sub(r.start)))}
	for _, l := range r.log {
		lines = append(lines, "  "+l)
	}
	if len(r.phases) > 0 {
		lines = append(lines, "  steps:")
	}
	for _, p := range r.phases {
		var line string
		if p.state == "run" {
			frame := spinner[int(now.Sub(p.started)/(250*time.Millisecond))%len(spinner)]
			line = fmt.Sprintf("  [%c] %s  (live %s)", frame, p.label, wallDur(now.Sub(p.started)))
		} else {
			line = fmt.Sprintf("  %s %s", phaseMark(p.state), p.label)
			if p.dur != "" {
				line += "  (" + p.dur + ")"
			}
		}
		lines = append(lines, line)
	}
	if len(r.arts) > 0 {
		lines = append(lines, "  outputs:")
	}
	for _, a := range r.arts {
		lines = append(lines, "  "+a)
	}
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
	_, err := fmt.Fprintf(r.w, "soda-iso: finished in %s with exit %d%s\n", wallSince(r.start), code, exitMeaning(code))
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
