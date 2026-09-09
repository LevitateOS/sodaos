package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type spaceView struct {
	Environment          environmentView   `json:"environment"`
	Login                string            `json:"login"`
	Administrator        bool              `json:"environment_administrator"`
	AuthorityUnavailable bool              `json:"authority_unavailable"`
	NativeUnavailable    bool              `json:"native_unavailable"`
	Observed             *host.Environment `json:"observed"`
	Terminals            []terminalView    `json:"terminals"`
}
type spacesView struct {
	Items    []spaceView `json:"items"`
	Complete bool        `json:"complete"`
}

// Fixed bounds: 4 requests, sequential provider/helper calls, 8 seconds total,
// 2 seconds per row, 128 scanned associations, 32 published rows, <=64 KiB JSON.
// No missing repository_id overload, copied permissions, unauthorized counts,
// unbounded fan-out or denial placeholders. Limits are incomplete, not empty truth.
func (s *Server) apiSpaces(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	select {
	case s.spacesSlots <- struct{}{}:
		defer func() { <-s.spacesSlots }()
	default:
		jsonError(w, 503, "spaces_unavailable", "Spaces inspection is busy; no complete result is available.")
		return
	}
	ctx, done := context.WithTimeout(r.Context(), 8*time.Second)
	defer done()
	projects, err := s.Store.SpaceProjects(ctx)
	if err != nil {
		jsonError(w, 503, "spaces_unavailable", "Could not enumerate Soda associations.")
		return
	}
	response := spacesView{Items: []spaceView{}, Complete: len(projects) <= 128}
	if len(projects) > 128 {
		projects = projects[:128]
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		jsonError(w, 401, "unauthenticated", "Sign in again.")
		return
	}
	for _, p := range projects {
		if ctx.Err() != nil || len(response.Items) >= 32 {
			response.Complete = false
			break
		}
		check, cancel := context.WithTimeout(ctx, 2*time.Second)
		request := r.WithContext(check)
		reader, err := s.readEnvironmentAuthority(request, v, p)
		if err != nil {
			if !errors.Is(err, errRepositoryDenied) {
				response.Complete = false
			}
			cancel()
			continue
		}
		// Operator inspection is legitimate but never a terminal-authority bypass.
		if v.User.ID == s.Config.OperatorID {
			_, err = s.visibleRepository(request, v, p.RepositoryID)
			reader.repositoryVisible = err == nil
			reader.authorityUnavailable = err != nil
		}
		row := spaceView{Environment: environmentDTO(p), Login: reader.login, Administrator: reader.administrator && !reader.authorityUnavailable, AuthorityUnavailable: reader.authorityUnavailable, Terminals: []terminalView{}}
		if reader.authorityUnavailable {
			response.Complete = false
		}
		observed, err := s.Host.Inspect(check, p.ID)
		row.NativeUnavailable = err != nil
		if err == nil {
			row.Observed = &observed
		} else {
			response.Complete = false
		}
		if reader.repositoryVisible && !reader.authorityUnavailable && reader.login != "" && s.terminalCurrent(check, cookie.Value, v, p, reader.login) {
			s.terminalMu.Lock()
			for _, entry := range s.terminals {
				if entry.matches(v, p, reader.login, cookie.Value) {
					if !entry.live() {
						entry.cancel()
					}
					row.Terminals = append(row.Terminals, entry.view())
				}
			}
			s.terminalMu.Unlock()
			sort.Slice(row.Terminals, func(i, j int) bool { return row.Terminals[i].ID < row.Terminals[j].ID })
		}
		cancel()
		response.Items = append(response.Items, row)
		body, err := json.Marshal(response)
		if err != nil || len(body) >= 65535 {
			response.Items = response.Items[:len(response.Items)-1]
			response.Complete = false
			break
		}
	}
	// Logout wins publication too; never hold the registry lock over inspections.
	s.terminalMu.Lock()
	current, err := s.Store.Session(ctx, cookie.Value)
	s.terminalMu.Unlock()
	if ctx.Err() != nil {
		jsonError(w, 503, "spaces_unavailable", "Spaces inspection exhausted its time budget; no complete result is available.")
		return
	}
	if err != nil || current.ContextID != v.ContextID || current.User.ID != v.User.ID || current.CSRF != v.CSRF {
		jsonError(w, 401, "unauthenticated", "Session ended.")
		return
	}
	jsonResponse(w, 200, response)
}
