// Adapted from soda-os bc1d3e0: private exclusive evidence and redacted errors.
package acceptance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

// Evidence holds an open directory capability. It never follows a path outside
// its new private root and never overwrites an earlier observation.
type Evidence struct {
	root    *os.Root
	secrets [][]byte
}

func CreateEvidence(path string, secrets [][]byte) (*Evidence, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("absolute fresh evidence directory required")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if parent != filepath.Clean(filepath.Dir(path)) {
		return nil, errors.New("evidence parent must not contain symlinks")
	}
	if err = os.Mkdir(path, 0700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	e := &Evidence{root: root}
	for _, secret := range secrets {
		if len(secret) > 0 {
			e.secrets = append(e.secrets, bytes.Clone(secret))
			// Logs may contain JSON-escaped credentials rather than raw values.
			encoded, _ := json.Marshal(string(secret))
			if !bytes.Equal(encoded[1:len(encoded)-1], secret) {
				e.secrets = append(e.secrets, bytes.Clone(encoded[1:len(encoded)-1]))
			}
		}
	}
	sort.Slice(e.secrets, func(i, j int) bool { return len(e.secrets[i]) > len(e.secrets[j]) })
	return e, nil
}
func (e *Evidence) Close() error { return e.root.Close() }
func (e *Evidence) Path() string { return e.root.Name() }
func (e *Evidence) Writer(name string) (io.WriteCloser, error) {
	f, err := e.open(name)
	if err != nil {
		return nil, err
	}
	return &redactingWriter{out: f, secrets: e.secrets}, nil
}

func (e *Evidence) open(name string) (*os.File, error) {
	if !filepath.IsLocal(name) || filepath.Clean(name) != name || name == "." {
		return nil, errors.New("invalid evidence name")
	}
	dir := filepath.Dir(name)
	if dir != "." {
		current := ""
		for _, part := range strings.Split(dir, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			if err := e.root.Mkdir(current, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, err
			}
			st, err := e.root.Lstat(current)
			if err != nil {
				return nil, err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("unsafe evidence parent")
			}
		}
	}
	f, err := e.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	return f, nil
}
func (e *Evidence) Write(name string, data []byte) error {
	w, err := e.Writer(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return errors.Join(err, w.Close())
}

// WriteJSON sanitizes string values before encoding. Raw log redaction must
// never rewrite encoded JSON (URL escaping can otherwise destroy its syntax).
func (e *Evidence) WriteJSON(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > evidenceLimit {
		return errors.New("structured evidence limit exceeded")
	}
	var decoded any
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err = d.Decode(&decoded); err != nil {
		return err
	}
	var scrub func(any) (any, error)
	scrub = func(v any) (any, error) {
		switch v := v.(type) {
		case string:
			return e.RedactString(v), nil
		case []any:
			for i := range v {
				v[i], err = scrub(v[i])
				if err != nil {
					return nil, err
				}
			}
		case map[string]any:
			out := make(map[string]any, len(v))
			for k, item := range v {
				key := e.RedactString(k)
				if _, exists := out[key]; exists {
					return nil, errors.New("redacted JSON key collision")
				}
				out[key], err = scrub(item)
				if err != nil {
					return nil, err
				}
			}
			return out, nil
		}
		return v, nil
	}
	decoded, err = scrub(decoded)
	if err != nil {
		return err
	}
	data, err = json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > evidenceLimit {
		return errors.New("structured evidence limit exceeded")
	}
	for _, secret := range e.secrets {
		if bytes.Contains(data, secret) {
			return errors.New("secret reached structured evidence")
		}
	}
	f, err := e.open(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return errors.Join(err, f.Sync(), f.Close())
}

// PublishObservation leaves incomplete attempts private and unpublishable. The
// final name is linked exclusively only after write/close/leak checks succeed.
func (e *Evidence) PublishObservation(o Observation) error {
	if err := e.WriteJSON("observation.pending.json", o); err != nil {
		return err
	}
	if err := e.CheckSecrets(); err != nil {
		return err
	}
	return e.root.Link("observation.pending.json", "observation.json")
}

// CheckSecrets is defense in depth, not a claim that unknown secrets are absent.
func (e *Evidence) CheckSecrets() error {
	return fs.WalkDir(e.root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("unexpected non-regular evidence entry")
		}
		f, err := e.root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() {
			return errors.New("evidence entry changed to a special file")
		}
		max := 1
		for _, s := range e.secrets {
			if len(s) > max {
				max = len(s)
			}
		}
		tail := []byte{}
		buf := make([]byte, 32768)
		for {
			n, rerr := f.Read(buf)
			block := append(tail, buf[:n]...)
			for _, s := range e.secrets {
				if bytes.Contains(block, s) {
					return errors.New("secret reached evidence")
				}
			}
			if len(block) >= max {
				tail = bytes.Clone(block[len(block)-max+1:])
			} else {
				tail = bytes.Clone(block)
			}
			if rerr == io.EOF {
				return nil
			}
			if rerr != nil {
				return rerr
			}
		}
	})
}

type safeError struct {
	message string
	cause   error
}

func (e safeError) Error() string { return e.message }
func (e safeError) Unwrap() error { return e.cause }
func (e *Evidence) RedactString(s string) string {
	for _, secret := range e.secrets {
		s = strings.ReplaceAll(s, string(secret), "[REDACTED]")
	}
	return redactURLs(s)
}

var urlPattern = regexp.MustCompile(`https?://[^\s"'<>\\]+`)

func redactURLs(s string) string {
	return urlPattern.ReplaceAllStringFunc(s, func(raw string) string {
		u, err := url.Parse(raw)
		if err != nil {
			return "[URL OMITTED]"
		}
		u.RawQuery = ""
		u.ForceQuery = false
		u.Fragment = ""
		u.User = nil
		return u.String()
	})
}
func (e *Evidence) RedactError(err error) error {
	if err == nil {
		return nil
	}
	return safeError{e.RedactString(err.Error()), err}
}

// Retain the longest possible match across Write calls. Longest matching secret
// wins; partial credentials never reach the file even when a command fails.
type redactingWriter struct {
	out        io.WriteCloser
	secrets    [][]byte
	pending    []byte
	urlPending []byte
	count      int64
	closed     bool
	err        error
}

const evidenceLimit = 16 << 20

func (w *redactingWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, os.ErrClosed
	}
	if w.err != nil {
		return 0, w.err
	}
	if w.count+int64(len(p)) > evidenceLimit {
		w.err = errors.New("evidence output limit exceeded")
		return 0, w.err
	}
	w.count += int64(len(p))
	w.pending = append(w.pending, p...)
	w.err = w.flush(false)
	return len(p), w.err
}
func (w *redactingWriter) flush(final bool) error {
	max := 1
	for _, s := range w.secrets {
		if len(s) > max {
			max = len(s)
		}
	}
	var out bytes.Buffer
	for len(w.pending) > 0 && (final || len(w.pending) >= max) {
		match := 0
		for _, s := range w.secrets {
			if len(s) > match && bytes.HasPrefix(w.pending, s) {
				match = len(s)
			}
		}
		if match > 0 {
			out.WriteString("[REDACTED]")
			w.pending = w.pending[match:]
		} else {
			out.WriteByte(w.pending[0])
			w.pending = w.pending[1:]
		}
	}
	w.urlPending = append(w.urlPending, out.Bytes()...)
	end := bytes.LastIndexByte(w.urlPending, '\n') + 1
	if final {
		end = len(w.urlPending)
	}
	if end > 0 {
		safe := redactURLs(string(w.urlPending[:end]))
		w.urlPending = bytes.Clone(w.urlPending[end:])
		_, err := io.WriteString(w.out, safe)
		return err
	}
	return nil
}
func (w *redactingWriter) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	w.err = errors.Join(w.err, w.flush(true), w.out.Close())
	return w.err
}

func PrivateFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("absolute private input required")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() > 1<<20 {
		return nil, fmt.Errorf("restricted regular input required: %s", filepath.Base(path))
	}
	return os.ReadFile(path)
}
