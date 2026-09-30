package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/web/auth"
)

var TerminalID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func newTerminalID() string {
	var id [16]byte
	_, _ = rand.Read(id[:])
	return hex.EncodeToString(id[:])
}

// TerminalCreationScope binds identity launches to the current admitted context.
func TerminalCreationScope(v store.Session) string {
	sum := sha256.Sum256([]byte(v.ContextID + "\x00" + v.CSRF))
	return hex.EncodeToString(sum[:])
}

type TerminalView struct {
	host.TerminalState
	EnvironmentID string `json:"environment_id"`
	RepositoryID  string `json:"repository_id"`
	UserID        string `json:"user_id"`
	Login         string `json:"login"`
}

func terminalDTO(item host.TerminalState, v store.Session, p store.Project, login string) TerminalView {
	return TerminalView{item, p.ID, strconv.FormatInt(p.RepositoryID, 10), strconv.FormatInt(v.User.ID, 10), login}
}

func validTerminalGeometry(cols, rows int, name string) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300 && host.ValidTerminalName(name)
}

func reservedTerminal(items []host.TerminalState, id string, err error) bool {
	return err == nil && len(items) == 1 && items[0].ID == id
}

func terminalMetadata(w http.ResponseWriter, view *TerminalView) {
	auth.JSONResponse(w, http.StatusOK, struct {
		Terminal *TerminalView `json:"terminal"`
	}{view})
}

type terminalSessionMutation struct {
	Action string  `json:"action"`
	Name   *string `json:"name"`
}

func parseTerminalSessionAction(w http.ResponseWriter, r *http.Request, in *host.TerminalRequest) bool {
	if r.Method == http.MethodGet {
		return true
	}
	var action terminalSessionMutation
	if !auth.DecodeAPIObject(w, r, &action) {
		return false
	}
	if !validTerminalSessionMutation(action) {
		auth.JSONError(w, http.StatusBadRequest, "invalid_action", "Choose End or Rename for this exact terminal.")
		return false
	}
	in.Action = action.Action
	if action.Name != nil {
		in.Name = *action.Name
	}
	return true
}

func validTerminalSessionMutation(action terminalSessionMutation) bool {
	switch action.Action {
	case "end":
		return action.Name == nil
	case "rename":
		return action.Name != nil && host.ValidTerminalName(*action.Name)
	default:
		return false
	}
}

func validTerminalOrigin(r *http.Request, origin string) bool {
	origins := r.Header.Values("Origin")
	return r.URL.RawQuery == "" && !r.URL.ForceQuery && len(origins) == 1 &&
		strings.HasPrefix(origin, "https://") && origins[0] == origin && len(r.Header.Values("Sec-WebSocket-Protocol")) == 0
}

func validSecFetchHeaders(r *http.Request) bool {
	for name, expected := range map[string]string{
		"Sec-Fetch-Site": "same-origin",
		"Sec-Fetch-Mode": "websocket",
		"Sec-Fetch-Dest": "empty",
	} {
		values := r.Header.Values(name)
		if len(values) > 1 || (len(values) == 1 && values[0] != expected) {
			return false
		}
	}
	return true
}

func checkTerminalRequestHeaders(w http.ResponseWriter, r *http.Request, origin string) bool {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		auth.JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "WebSocket GET required.")
		return false
	}
	if !validTerminalOrigin(r, origin) || !validSecFetchHeaders(r) {
		auth.JSONError(w, http.StatusForbidden, "invalid_origin", "Same-origin terminal required.")
		return false
	}
	return true
}

type terminalHandshake struct {
	Action            string `json:"action"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	RepositoryID      string `json:"repository_id"`
	SessionGeneration string `json:"session_generation"`
	Cols              int    `json:"cols"`
	Rows              int    `json:"rows"`
}

func readTerminalHandshake(ctx context.Context, conn *websocket.Conn) (terminalHandshake, error) {
	var in terminalHandshake
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

func validHandshakeDimensions(cols, rows int) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300
}

func validHandshakeAction(id, action, name string) bool {
	if !TerminalID.MatchString(id) {
		return false
	}
	return (action == "create" && host.ValidTerminalName(name)) || (action == "attach" && name == "")
}

func validHandshakeRepository(in terminalHandshake, repositoryID int64) bool {
	id, valid := auth.PositiveID(in.RepositoryID)
	return valid && id == repositoryID
}

func refuseTerminal(conn *websocket.Conn) {
	_ = conn.Close(websocket.StatusPolicyViolation, "terminal unavailable or native authorization failed; use an exact ID or reload this page")
}

func pumpTerminalInput(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, controls chan<- host.TerminalFrame) {
	defer cancel()
	for {
		kind, body, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var frame host.TerminalFrame
		if kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &frame) != nil || (frame.Type != "input" && frame.Type != "resize") {
			return
		}
		select {
		case controls <- frame:
		case <-ctx.Done():
			return
		}
	}
}

func pumpNativeToExtension(ctx context.Context, conn *websocket.Conn, native *host.Terminal) {
	for {
		frame, err := native.Receive(ctx)
		if err != nil || frame.Type == "metadata" {
			return
		}
		body, _ := json.Marshal(frame)
		write, done := context.WithTimeout(ctx, 5*time.Second)
		err = conn.Write(write, websocket.MessageText, body)
		done()
		if err != nil || frame.Type == "closed" {
			return
		}
	}
}

// CloseTerminals shuts down native extension terminal peers before process exit.
func (s *API) CloseTerminals() {
	s.terminalMu.Lock()
	s.terminalClosed = true
	for _, peer := range s.TerminalPeers {
		peer.Cancel()
	}
	s.terminalMu.Unlock()
	s.terminalWG.Wait()
}

func (s *API) dropTerminalPeer(r *http.Request) {
	s.terminalMu.Lock()
	delete(s.TerminalPeers, r)
	s.terminalMu.Unlock()
}

func (s *API) unregisterTerminalPeer(r *http.Request) {
	s.dropTerminalPeer(r)
	s.terminalWG.Done()
}

func peerIDInUse(peers map[*http.Request]*TerminalPeer, current *TerminalPeer, projectID, id string) bool {
	for _, other := range peers {
		if other != current && other.Project == projectID && other.ID == id {
			return true
		}
	}
	return false
}
