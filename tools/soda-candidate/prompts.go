// prompts asks the operator for every build choice the flags did not
// pre-seed. Flags stay as scriptable overrides; the TUI is the primary
// interface. IO is injected so scripted tests can drive the flow.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newPrompter(stdin io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewReader(stdin), out: out}
}

func (p *prompter) line(prompt, def string) (string, error) {
	if def != "" {
		if _, err := fmt.Fprintf(p.out, "%s [%s]: ", prompt, def); err != nil {
			return "", err
		}
	} else if _, err := fmt.Fprintf(p.out, "%s: ", prompt); err != nil {
		return "", err
	}
	s, err := p.in.ReadString('\n')
	if err != nil {
		return "", errors.New("input ended; rerun with flags or --non-interactive")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	return s, nil
}

// choice shows a numbered menu and returns the selected index.
func (p *prompter) choice(prompt string, options []string, def int) (int, error) {
	for i, o := range options {
		if _, err := fmt.Fprintf(p.out, "  %d) %s\n", i+1, o); err != nil {
			return 0, err
		}
	}
	for {
		s, err := p.line(prompt, strconv.Itoa(def+1))
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 1 || n > len(options) {
			if _, err := fmt.Fprintf(p.out, "Enter a number 1-%d.\n", len(options)); err != nil {
				return 0, err
			}
			continue
		}
		return n - 1, nil
	}
}

func modeIndex(mode string) int {
	for i, m := range []string{"candidate", "media", "production"} {
		if mode == m {
			return i
		}
	}
	return 1
}

// ovField is one editable row on the overview screen.
type ovField struct {
	label  string
	show   func() string
	status func() string
	edit   func() error
}

func outStatus(out string) string {
	switch {
	case !filepath.IsAbs(out):
		return "✗ not absolute"
	case !parentDirExists(out):
		return "✗ parent missing"
	case !pathAbsent(out):
		return "✗ already exists"
	default:
		return "✓ fresh"
	}
}

// overviewFields lists every choice for the current mode. Arch is shown
// pinned, never edited: the controller only runs natively.
func overviewFields(p *prompter, o *options) []ovField {
	fields := []ovField{
		{"mode", func() string { return o.mode }, nil, func() error {
			sel, err := p.choice("Build mode", modeOptions, modeIndex(o.mode))
			if err != nil {
				return err
			}
			o.mode = []string{"candidate", "media", "production"}[sel]
			return nil
		}},
		{"output", func() string { return o.out }, func() string { return outStatus(o.out) }, func() error {
			s, err := p.askOut("Fresh output directory", o.out)
			if err != nil {
				return err
			}
			o.out = s
			return nil
		}},
		{"controller", func() string { return o.controller }, nil, func() error {
			s, err := p.askAbsolute("Admitted soda-build executable", o.controller)
			if err != nil {
				return err
			}
			o.controller = s
			return nil
		}},
		{"worker config", func() string { return o.workerConfig }, nil, func() error {
			s, err := p.askAbsolute("Restricted worker config", o.workerConfig)
			if err != nil {
				return err
			}
			o.workerConfig = s
			return nil
		}},
	}
	switch o.mode {
	case "media":
		fields = append(fields,
			ovField{"rootfs URL", func() string { return o.rootfsURL }, nil, func() error {
				s, err := p.askNonEmpty("Public rootfs base URL", o.rootfsURL)
				if err != nil {
					return err
				}
				o.rootfsURL = s
				return nil
			}},
			ovField{"fast compress", func() string {
				if o.compression == "fast" {
					return "yes"
				}
				return "no"
			}, nil, func() error {
				def := "n"
				if o.compression == "fast" {
					def = "y"
				}
				s, err := p.askFast(def)
				if err != nil {
					return err
				}
				o.compression = s
				return nil
			}},
		)
	case "production":
		fields = append(fields,
			ovField{"rootfs URL", func() string { return o.rootfsURL }, nil, func() error {
				s, err := p.askNonEmpty("Public rootfs base URL", o.rootfsURL)
				if err != nil {
					return err
				}
				o.rootfsURL = s
				return nil
			}},
			ovField{"qualification", func() string { return o.qualConfig }, nil, func() error {
				s, err := p.askAbsolute("Qualification config", o.qualConfig)
				if err != nil {
					return err
				}
				o.qualConfig = s
				return nil
			}},
			ovField{"signing", func() string {
				if o.signConfig == "" {
					return "(empty: ends incomplete)"
				}
				return o.signConfig
			}, nil, func() error {
				for {
					s, err := p.line("Signing config (empty ends incomplete, no qualified release)", o.signConfig)
					if err != nil {
						return err
					}
					if s != "" && !filepath.IsAbs(s) {
						if _, err := fmt.Fprintln(p.out, "Absolute path required."); err != nil {
							return err
						}
						o.signConfig = s
						continue
					}
					o.signConfig = s
					return nil
				}
			}},
		)
	}
	fields = append(fields, ovField{"repo prefix", func() string { return o.repoPrefix }, nil, func() error {
		s, err := p.askNonEmpty("Image repository prefix (intent only; no publication)", o.repoPrefix)
		if err != nil {
			return err
		}
		o.repoPrefix = s
		return nil
	}})
	return fields
}

// overview shows every choice on one screen. The operator edits fields by
// number and starts with 'go'. Nothing runs before an explicit start, and a
// blocked start explains itself without losing any answers.
func (p *prompter) overview(o *options, suggestOut func() string) error {
	if o.mode == "" {
		o.mode = "media"
	}
	if o.out == "" {
		o.out = suggestOut()
	}
	for {
		fields := overviewFields(p, o)
		if _, err := fmt.Fprintf(p.out, "\nsoda-candidate | %s | arch %s (this host)\n", modeLabel(o.mode), o.arch); err != nil {
			return err
		}
		for i, f := range fields {
			extra := ""
			if f.status != nil {
				extra = "  [" + f.status() + "]"
			}
			if _, err := fmt.Fprintf(p.out, "  %d) %-13s %s%s\n", i+1, f.label, f.show(), extra); err != nil {
				return err
			}
		}
		s, err := p.line("Number to edit, 'go' to start, 'quit' to abort", "go")
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "go", "run", "":
			if err := validateResolved(o); err != nil {
				if _, werr := fmt.Fprintf(p.out, "Cannot start: %s\n", err); werr != nil {
					return werr
				}
				continue
			}
			return nil
		case "quit", "q", "abort":
			return errors.New("aborted by operator")
		}
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil || n < 1 || n > len(fields) {
			if _, err := fmt.Fprintln(p.out, "Type a field number, 'go', or 'quit'."); err != nil {
				return err
			}
			continue
		}
		if err := fields[n-1].edit(); err != nil {
			return err
		}
	}
}

var modeOptions = []string{
	"Development candidate (no media, no signing)",
	"Development media (installer ISO, fixture rootfs URL)",
	"Production (protected qualification, optional final signing)",
}

func modeLabel(mode string) string {
	switch mode {
	case "candidate":
		return modeOptions[0]
	case "production":
		return modeOptions[2]
	default:
		return modeOptions[1]
	}
}

// askAbsolute re-asks until the answer is an absolute path.
func (p *prompter) askAbsolute(prompt, def string) (string, error) {
	for {
		s, err := p.line(prompt, def)
		if err != nil {
			return "", err
		}
		def = s
		if !filepath.IsAbs(s) {
			if _, err := fmt.Fprintln(p.out, "Absolute path required."); err != nil {
				return "", err
			}
			continue
		}
		return s, nil
	}
}

// askNonEmpty re-asks until the answer is not blank.
func (p *prompter) askNonEmpty(prompt, def string) (string, error) {
	for {
		s, err := p.line(prompt, def)
		if err != nil {
			return "", err
		}
		def = s
		if strings.TrimSpace(s) == "" {
			if _, err := fmt.Fprintln(p.out, "A value is required here."); err != nil {
				return "", err
			}
			continue
		}
		return s, nil
	}
}

func parentDirExists(path string) bool {
	st, err := os.Stat(filepath.Dir(path))
	return err == nil && st.IsDir()
}

func pathAbsent(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// askOut re-asks until the answer is a fresh absolute directory whose
// parent already exists, mirroring the controller's output rules.
func (p *prompter) askOut(prompt, def string) (string, error) {
	for {
		s, err := p.line(prompt, def)
		if err != nil {
			return "", err
		}
		def = s
		var reason string
		switch {
		case !filepath.IsAbs(s):
			reason = "Absolute path required."
		case !parentDirExists(s):
			reason = "Parent " + filepath.Dir(s) + " must already exist."
		case !pathAbsent(s):
			reason = "That output exists; each attempt needs a fresh directory."
		default:
			return s, nil
		}
		if _, err := fmt.Fprintln(p.out, reason); err != nil {
			return "", err
		}
	}
}

// askFast offers iteration compression, defaulting from the flags.
func (p *prompter) askFast(def string) (string, error) {
	for {
		s, err := p.line("Fast iteration compression (dev ISO only) [y/n]", def)
		if err != nil {
			return "", err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "y", "yes":
			return "fast", nil
		case "n", "no":
			return "", nil
		default:
			def = s
			if _, err := fmt.Fprintln(p.out, "Answer y or n."); err != nil {
				return "", err
			}
		}
	}
}
