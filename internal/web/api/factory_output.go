package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func privateFactoryOutputRoute(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	return r.Method == http.MethodGet && len(parts) == 6 && parts[1] == "api" && parts[2] == "factory" && parts[3] == "runs" && parts[4] != "" && parts[5] == "output"
}

func checkFactoryOutputHeaders(w http.ResponseWriter, r *http.Request, origin string) bool {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		auth.JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "WebSocket GET required.")
		return false
	}
	if !validTerminalOrigin(r, origin) || !validSecFetchHeaders(r) {
		auth.JSONError(w, http.StatusForbidden, "invalid_origin", "Same-origin factory view required.")
		return false
	}
	return true
}

// factoryViewerIdentity binds one read-only output attachment to its native
// authority, run and project. Viewers need current code-write authority,
// like human terminal access; project membership is not required.
type factoryViewerIdentity struct {
	authority extensions.Authority
	project   store.Project
	runID     string
	actorID   int64
}

func (s *API) factoryViewerAccount(w http.ResponseWriter, r *http.Request) (factoryViewerIdentity, bool) {
	var identity factoryViewerIdentity
	authority, err := auth.ExtensionAuthority(r)
	if err != nil || !nativeTerminalContribution(authority) {
		auth.JSONError(w, 403, "native_authority_unavailable", "Current native factory-view authority is required.")
		return identity, false
	}
	actorID, ok := s.nativeTerminalActor(authority)
	if !ok {
		auth.JSONError(w, 403, "invalid_actor", "Native actor is invalid.")
		return identity, false
	}
	runID := r.PathValue("runID")
	if !factory.ValidID(runID) {
		auth.JSONError(w, 404, "not_found", "Factory run not found.")
		return identity, false
	}
	run, err := s.Store.FactoryRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Factory run not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read factory run.")
		}
		return identity, false
	}
	p, err := s.Store.Project(r.Context(), run.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Factory run not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read factory run.")
		}
		return identity, false
	}
	identity = factoryViewerIdentity{authority: authority, project: p, runID: runID, actorID: actorID}
	if !s.factoryViewerCurrent(r.Context(), r, identity) {
		auth.JSONError(w, 403, "repository_write_required", "Current repository write permission is required.")
		return factoryViewerIdentity{}, false
	}
	return identity, true
}

// factoryViewerCurrent revalidates the attachment: the same live native
// authority, actor and contribution, the same repository binding and current
// code-write permission. Logout, permission loss or account switch ends the
// view; stale private output is never served to a changed viewer.
func (s *API) factoryViewerCurrent(ctx context.Context, r *http.Request, identity factoryViewerIdentity) bool {
	if ctx.Err() != nil {
		return false
	}
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	current, err := auth.ExtensionAuthority(r.WithContext(check))
	if err != nil || !sameNativeTerminalAuthority(current, identity.authority) {
		return false
	}
	p, err := s.Store.Project(check, identity.project.ID)
	if err != nil || p.RepositoryID != identity.project.RepositoryID {
		return false
	}
	return s.extensionTerminalRepository(check, extensionTerminalIdentity{authority: current, project: p})
}

// factoryOutputHandshake opens one read-only attachment. The cursor resumes
// a detached view; reattachment only reads, never launches or resumes an
// agent.
type factoryOutputHandshake struct {
	RunID             string `json:"run_id"`
	RepositoryID      string `json:"repository_id"`
	SessionGeneration string `json:"session_generation"`
	Cursor            int64  `json:"cursor"`
}

func readFactoryOutputHandshake(ctx context.Context, conn *websocket.Conn) (factoryOutputHandshake, error) {
	var in factoryOutputHandshake
	first, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	kind, body, err := conn.Read(first)
	if err != nil {
		return in, err
	}
	if kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil {
		return in, errors.New("invalid handshake")
	}
	return in, nil
}

func validFactoryOutputHandshake(in factoryOutputHandshake, identity factoryViewerIdentity) bool {
	id, valid := auth.PositiveID(in.RepositoryID)
	return in.RunID == identity.runID && valid && id == identity.project.RepositoryID &&
		in.SessionGeneration == identity.authority.SessionGeneration &&
		in.Cursor >= 0 && in.Cursor <= project.MaxFactoryOutputOffset
}

// Factory output frames use fixed key sets, like the terminal transport:
// status carries run/process identity, output carries base64 bytes with
// server-chosen cursors, closed ends the attachment with a reason.
type factoryStatusFrame struct {
	Type       string `json:"type"`
	RunID      string `json:"run_id"`
	Phase      string `json:"phase"`
	Container  string `json:"container"`
	Unit       string `json:"unit"`
	Invocation string `json:"invocation"`
	Reason     string `json:"reason"`
	Live       bool   `json:"live"`
	Terminal   bool   `json:"terminal"`
	ExitCode   *int   `json:"exit_code"`
}

type factoryOutputFrame struct {
	Type      string `json:"type"`
	Data      string `json:"data"`
	Cursor    int64  `json:"cursor"`
	Next      int64  `json:"next"`
	Gap       bool   `json:"gap"`
	Truncated bool   `json:"truncated"`
}

type factoryClosedFrame struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func factoryOutputStatus(runID string, out project.FactoryOutputState) factoryStatusFrame {
	return factoryStatusFrame{
		Type: "status", RunID: runID, Phase: out.Phase, Container: out.Container,
		Unit: out.Unit, Invocation: out.Invocation, Reason: out.Reason,
		Live: out.Live, Terminal: out.Terminal, ExitCode: out.ExitCode,
	}
}

func writeFactoryFrame(ctx context.Context, conn *websocket.Conn, frame any) error {
	body, _ := json.Marshal(frame)
	write, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	return conn.Write(write, websocket.MessageText, body)
}

func refuseFactoryOutput(conn *websocket.Conn) {
	_ = conn.Close(websocket.StatusPolicyViolation, "factory view unavailable or native authorization failed; use the exact run or reload this page")
}

func (s *API) factoryOutputStream(w http.ResponseWriter, r *http.Request) {
	if !checkFactoryOutputHeaders(w, r, s.Config.ForgejoURL) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "host_unavailable", "Run observation is unavailable.")
		return
	}
	identity, ok := s.factoryViewerAccount(w, r)
	if !ok {
		return
	}
	s.factoryOutputAttach(w, r, identity)
}

func (s *API) factoryOutputAttach(w http.ResponseWriter, r *http.Request, identity factoryViewerIdentity) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	if _, ok := s.registerExtensionTerminalPeer(r, extensionTerminalIdentity{authority: identity.authority, project: identity.project}, cancel); !ok {
		auth.JSONError(w, 409, "terminal_unavailable", "Factory view transport unavailable.")
		return
	}
	defer s.unregisterTerminalPeer(r)
	// The private proxy uses an internal Host; exact public Origin was checked
	// before this upgrade, so the library's Host comparison does not apply here.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(32768)
	closePeer := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer closePeer()
	in, err := readFactoryOutputHandshake(ctx, conn)
	if err != nil || !validFactoryOutputHandshake(in, identity) {
		refuseFactoryOutput(conn)
		return
	}
	if !s.factoryViewerCurrent(ctx, r, identity) {
		refuseFactoryOutput(conn)
		return
	}
	go factoryRejectViewerInput(ctx, cancel, conn)
	s.pumpFactoryOutput(ctx, conn, r, identity, in.Cursor)
}

// factoryRejectViewerInput ends a read-only attachment the moment the viewer
// sends anything past the handshake: input, resize, signals and End frames
// cannot reach the factory process.
func factoryRejectViewerInput(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn) {
	defer cancel()
	if _, _, err := conn.Read(ctx); err == nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "read-only factory stream; input, resize and End are rejected")
	}
}

func (s *API) pumpFactoryOutput(ctx context.Context, conn *websocket.Conn, r *http.Request, identity factoryViewerIdentity, cursor int64) {
	poll := time.NewTicker(500 * time.Millisecond)
	defer poll.Stop()
	heart := time.NewTicker(15 * time.Second)
	defer heart.Stop()
	var last factoryStatusFrame
	sent := false
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-heart.C:
			if !s.factoryOutputHeartbeat(ctx, conn, r, identity) {
				return
			}
		case <-poll.C:
			next, done, ok := s.pumpFactorySlice(ctx, conn, identity, cursor, &last, &sent)
			cursor = next
			if done {
				return
			}
			if !ok {
				failures++
				if failures > 10 {
					_ = writeFactoryFrame(ctx, conn, factoryClosedFrame{Type: "closed", Reason: "output_unavailable"})
					return
				}
				continue
			}
			failures = 0
		}
	}
}

func (s *API) pumpFactorySlice(ctx context.Context, conn *websocket.Conn, identity factoryViewerIdentity, cursor int64, last *factoryStatusFrame, sent *bool) (int64, bool, bool) {
	out, err := s.Host.FactoryOutput(ctx, project.FactoryOutput{Project: identity.project.ID, ID: identity.runID, Offset: cursor, Limit: project.MaxFactoryOutputRead})
	if err != nil {
		return cursor, s.closeFactorySlice(ctx, conn, err), false
	}
	status := factoryOutputStatus(identity.runID, out)
	if !*sent || !sameFactoryStatus(status, *last) {
		if writeFactoryFrame(ctx, conn, status) != nil {
			return cursor, true, false
		}
		*last, *sent = status, true
	}
	if out.Data != "" || out.Gap || out.Truncated {
		frame := factoryOutputFrame{Type: "output", Data: out.Data, Cursor: out.Offset, Next: out.Next, Gap: out.Gap, Truncated: out.Truncated}
		if writeFactoryFrame(ctx, conn, frame) != nil {
			return cursor, true, false
		}
	}
	if out.Terminal {
		_ = writeFactoryFrame(ctx, conn, factoryClosedFrame{Type: "closed", Reason: "eof"})
		return out.Next, true, true
	}
	return out.Next, false, true
}

// sameFactoryStatus compares two status frames by value, including the
// optional exit code.
func sameFactoryStatus(a, b factoryStatusFrame) bool {
	if a.ExitCode == nil || b.ExitCode == nil {
		return a.ExitCode == b.ExitCode && statusWithoutExit(a) == statusWithoutExit(b)
	}
	return *a.ExitCode == *b.ExitCode && statusWithoutExit(a) == statusWithoutExit(b)
}

func statusWithoutExit(s factoryStatusFrame) factoryStatusFrame {
	s.ExitCode = nil
	return s
}

// closeFactorySlice ends the attachment for a host read failure. Unknown
// runs and changed incarnations close with their reason; anything else is
// a transient miss the pump retries within its failure budget.
func (s *API) closeFactorySlice(ctx context.Context, conn *websocket.Conn, err error) bool {
	switch {
	case errors.Is(err, host.ErrRunNotFound):
		_ = writeFactoryFrame(ctx, conn, factoryClosedFrame{Type: "closed", Reason: "run_unknown"})
		return true
	case errors.Is(err, host.ErrRunStale):
		_ = writeFactoryFrame(ctx, conn, factoryClosedFrame{Type: "closed", Reason: "incarnation_changed"})
		return true
	default:
		return false
	}
}

func (s *API) factoryOutputHeartbeat(ctx context.Context, conn *websocket.Conn, r *http.Request, identity factoryViewerIdentity) bool {
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	if !s.factoryViewerCurrent(check, r, identity) {
		_ = writeFactoryFrame(ctx, conn, factoryClosedFrame{Type: "closed", Reason: "authority_lost"})
		return false
	}
	return conn.Ping(check) == nil
}
