// Package host is the privileged Unix client and thin Daemon facade: mux,
// admission and wiring of host/project, host/terminal and host/tailnet. It is
// not the SQLite owner, browser OAuth surface or build/release controller.
package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	projectexec "github.com/levitateos/sodaos/internal/host/project"
	tailnetexec "github.com/levitateos/sodaos/internal/host/tailnet"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/platform"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
)

var networkName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)

type Config struct {
	TailnetManagement bool   `json:"tailnet_management,omitempty"`
	TailnetImage      string `json:"tailnet_image,omitempty"`
	Image             string `json:"image"`
	Network           string `json:"network"`
	Subnet            string `json:"subnet"`
	Bridge            string `json:"bridge"`
}

func LoadConfig(path string) (Config, error) {
	return loadConfig(path, platform.Release)
}

func applyReleaseImages(c *Config, releasePath string) error {
	if releasePath == "" {
		return nil
	}
	p, err := deliver.Load(releasePath)
	if err != nil || build.RequireNative(p.Architecture) != nil {
		return errors.New("immutable appliance image defaults unavailable")
	}
	// Never silently replace an operator's saved image choice. New installs
	// omit these fields; a conflicting old install needs explicit migration.
	projectImage, companionImage := p.Images["project-os"].Config, p.Images["tailnet"].Config
	if (c.Image != "" && c.Image != projectImage) || (c.TailnetImage != "" && c.TailnetImage != companionImage) {
		return errors.New("saved image selection conflicts with appliance release; explicit migration required")
	}
	c.Image = projectImage
	if c.TailnetManagement {
		c.TailnetImage = companionImage
	}
	return nil
}

func validTailnetImage(c Config) bool {
	if c.TailnetImage == "" {
		return true
	}
	return c.TailnetManagement && strings.HasPrefix(c.TailnetImage, "sha256:") && project.ValidImageRef(c.TailnetImage)
}

func validNetworkNames(c Config) bool {
	return c.Image != "" && !strings.HasPrefix(c.Image, "-") && networkName.MatchString(c.Network) && networkName.MatchString(c.Bridge)
}

func validateRuntimeConfig(c Config) error {
	if _, err := netip.ParsePrefix(c.Subnet); err != nil {
		return err
	}
	if !validTailnetImage(c) {
		return errors.New("invalid immutable Tailnet companion configuration")
	}
	if !validNetworkNames(c) {
		return errors.New("invalid native runtime configuration")
	}
	return nil
}

func loadConfig(path, releasePath string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	if err = applyReleaseImages(&c, releasePath); err != nil {
		return c, err
	}
	if err = validateRuntimeConfig(c); err != nil {
		return c, err
	}
	return c, nil
}

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}
type Native struct{}

func (Native) HostNative() {}

func (Native) Run(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = bytes.NewReader(in)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", command, err, stderr.String())
	}
	return out, nil
}

type Daemon struct {
	Tailnet       *tailnet.Control
	Companion     *tailnetexec.Companion
	Terminal      *terminal.Service
	Project       *projectexec.Runtime
	Runners       *runners.Operations
	Config        Config
	Exec          Executor
	admissionOnce sync.Once
	admission     chan struct{}
}

// NewDaemon wires the thin host facade with project, terminal, runners and
// optional Tailnet companion runtimes. Callers outside package host should use
// this instead of assembling subsystem packages themselves.
func NewDaemon(c Config) *Daemon {
	native := Native{}
	runnerNative := runners.NewNative()
	d := &Daemon{
		Config:   c,
		Exec:     native,
		Project:  projectRuntime(native, c),
		Terminal: &terminal.Service{Exec: native},
		Runners:  &runners.Operations{Local: runnerNative, Lifecycle: runnerNative},
	}
	if !c.TailnetManagement {
		return d
	}
	if c.TailnetImage != "" {
		d.Tailnet = tailnet.NewProjectControl()
	} else {
		d.Tailnet = tailnet.NewControl()
	}
	d.Companion = &tailnetexec.Companion{Exec: native, Tailnet: d.Tailnet, Image: c.TailnetImage}
	return d
}

// NewCompanionDaemon wires only the companion path used by systemd run/stop phases.
func NewCompanionDaemon(c Config) *Daemon {
	native := Native{}
	control := tailnet.NewProjectControl()
	return &Daemon{
		Config:  c,
		Exec:    native,
		Project: projectRuntime(native, c),
		Tailnet: control,
		Companion: &tailnetexec.Companion{
			Exec:    native,
			Tailnet: control,
			Image:   c.TailnetImage,
		},
	}
}

// Keep mutations globally serialized, but let cancelled waiters leave.
// Lazy initialization keeps Daemon struct literals usable without a constructor.
// This gate is independent of terminal streams and runner file locks.
func (d *Daemon) acquireAdmission(ctx context.Context) error {
	d.admissionOnce.Do(func() { d.admission = make(chan struct{}, 1) })
	select {
	case d.admission <- struct{}{}:
		// Done and the gate can both be ready; select does not prioritize Done.
		if err := ctx.Err(); err != nil {
			<-d.admission
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var errNotFound = errors.New("not found")

func (d *Daemon) routeSubsystem(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/tailnet/") {
		d.tailnetHandler(w, r)
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/runners/") {
		d.runnerHandler(w, r)
		return true
	}
	if r.URL.Path == "/terminal" {
		d.terminalHandler(w, r)
		return true
	}
	return false
}

func hasNativeCleanPath(r *http.Request) bool {
	switch r.URL.Path {
	case "/lifecycle", "/access-keys", "/profile", "/create", "/os":
		return r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == ""
	default:
		return true
	}
}

func validateNativeOperationRequest(r *http.Request) int {
	if !hasNativeCleanPath(r) {
		return http.StatusBadRequest
	}
	if r.Method != "POST" {
		return http.StatusMethodNotAllowed
	}
	return 0
}

func nativeRequestErrorMessage(code int) string {
	if code == http.StatusBadRequest {
		return "invalid native operation path"
	}
	return "POST required"
}

func isAdmittedMutationPath(path string) bool {
	return path == "/lifecycle" || path == "/access-keys" || path == "/account"
}

func makeBodyDecoder(body io.Reader, ctx context.Context) func(any) error {
	return func(v any) error {
		if err := strictjson.Decode(body, v); err != nil {
			return err
		}
		// A body read can outlive admission. Refuse cancellation before dispatch;
		// already-running native commands retain their existing context handling.
		return ctx.Err()
	}
}

func (d *Daemon) dispatchProfile(ctx context.Context, decode func(any) error) (any, error) {
	var in struct{}
	if err := decode(&in); err != nil {
		return nil, err
	}
	return d.Project.ResolveProfile(ctx)
}

func (d *Daemon) dispatchCreate(ctx context.Context, decode func(any) error) (any, error) {
	var in project.Create
	if err := decode(&in); err != nil {
		return nil, err
	}
	if !project.ValidID(in.ID) || in.Owner <= 0 {
		return nil, errors.New("invalid project identity")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	// Create is not on the shared mutation gate; it owns writer admission here.
	if err := d.acquireAdmission(ctx); err != nil {
		return nil, err
	}
	defer func() { <-d.admission }()
	return d.Project.Create(ctx, in)
}

func (d *Daemon) dispatchCreateTargeted(ctx context.Context, path string, decode func(any) error) (any, error) {
	var in project.Create
	if err := decode(&in); err != nil {
		return nil, err
	}
	if !project.ValidID(in.ID) {
		return nil, errors.New("invalid project identity")
	}
	switch path {
	case "/inspect":
		out, _, err := d.Project.Inspect(ctx, in.ID)
		return out, err
	case "/os":
		return d.Project.ObserveOS(ctx, in.ID)
	case "/connection":
		return d.Project.Connection(ctx, in.ID)
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) dispatchMutation(ctx context.Context, path string, decode func(any) error) (any, error) {
	switch path {
	case "/lifecycle":
		var in project.Lifecycle
		if err := decode(&in); err != nil {
			return nil, err
		}
		return d.Project.Lifecycle(ctx, in)
	case "/access-keys":
		var in project.AccessKeys
		if err := decode(&in); err != nil {
			return nil, err
		}
		return d.Project.AccessKeys(ctx, in)
	case "/account":
		var in project.Account
		if err := decode(&in); err != nil {
			return nil, err
		}
		if err := d.Project.Account(ctx, in); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) dispatchOperation(ctx context.Context, path string, decode func(any) error) (any, error) {
	switch path {
	case "/profile":
		return d.dispatchProfile(ctx, decode)
	case "/create":
		return d.dispatchCreate(ctx, decode)
	case "/inspect", "/os", "/connection":
		return d.dispatchCreateTargeted(ctx, path, decode)
	case "/lifecycle", "/access-keys", "/account":
		return d.dispatchMutation(ctx, path, decode)
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.routeSubsystem(w, r) {
		return
	}
	if code := validateNativeOperationRequest(r); code != 0 {
		http.Error(w, nativeRequestErrorMessage(code), code)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	decode := makeBodyDecoder(r.Body, ctx)
	if ctx.Err() != nil {
		http.Error(w, "native operation cancelled before admission", http.StatusRequestTimeout)
		return
	}
	// Read-only observations may overlap mutations and report transitional state.
	// Create owns the same writer gate inside its native provisioning phase.
	if isAdmittedMutationPath(r.URL.Path) {
		if err := d.acquireAdmission(ctx); err != nil {
			http.Error(w, "native operation cancelled before admission", http.StatusRequestTimeout)
			return
		}
		defer func() { <-d.admission }()
	}
	out, err := d.dispatchOperation(ctx, r.URL.Path, decode)
	if err != nil {
		if errors.Is(err, errNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.Error("project native operation failed", "operation", r.URL.Path, "error", err)
		http.Error(w, "native operation failed; inspect operator journal", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
