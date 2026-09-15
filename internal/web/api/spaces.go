package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type SpaceView struct {
	TailnetState         string               `json:"tailnet_state,omitempty"`
	Environment          EnvironmentView      `json:"environment"`
	Login                string               `json:"login"`
	Administrator        bool                 `json:"environment_administrator"`
	AuthorityUnavailable bool                 `json:"authority_unavailable"`
	NativeUnavailable    bool                 `json:"native_unavailable"`
	Observed             *project.Environment `json:"observed"`
	Terminals            []TerminalView       `json:"terminals"`
}

type SpacesView struct {
	Items    []SpaceView `json:"items"`
	Complete bool        `json:"complete"`
}

func (s *API) resolveSpaceAuthority(request *http.Request, v store.Session, p store.Project) (environmentReader, bool, bool) {
	reader, err := s.readEnvironmentAuthority(request, v, p)
	if err != nil {
		complete := errors.Is(err, errRepositoryDenied)
		return reader, false, complete
	}
	// Operator inspection is legitimate but never a terminal-authority bypass.
	if v.User.ID == s.Config.OperatorID {
		_, err = s.visibleRepository(request, v, p.RepositoryID)
		reader.repositoryVisible = err == nil
		reader.authorityUnavailable = err != nil
	}
	return reader, true, !reader.authorityUnavailable
}

func (s *API) inspectSpaceNative(ctx context.Context, p store.Project, row *SpaceView) bool {
	observed, err := s.Host.Inspect(ctx, p.ID)
	if err == nil && p.Profile != nil && (observed.Profile == nil || *observed.Profile != *p.Profile) {
		err = errors.New("creation profile mismatch")
	}
	row.NativeUnavailable = err != nil
	if err == nil {
		row.Observed = &observed
		return true
	}
	return false
}

func (s *API) inspectSpaceTerminals(
	request *http.Request,
	check context.Context,
	cookieValue string,
	v store.Session,
	p store.Project,
	reader environmentReader,
	row *SpaceView,
	terminalCount *int,
) bool {
	if !reader.repositoryVisible || reader.authorityUnavailable || reader.login == "" || !s.terminalCurrent(check, cookieValue, v, p, reader.login) {
		return true
	}
	items, err := s.terminalOperation(request, v, p, reader.login, cookieValue, host.TerminalRequest{Action: "list"})
	complete := err == nil
	for _, item := range items {
		if *terminalCount >= 64 {
			complete = false
			break
		}
		row.Terminals = append(row.Terminals, terminalDTO(item, v, p, reader.login))
		*terminalCount++
	}
	return complete
}

func (s *API) inspectSpaceTailnet(check context.Context, p store.Project, reader environmentReader, row *SpaceView) {
	if !p.Ready || reader.authorityUnavailable || (!reader.administrator && reader.login == "") {
		return
	}
	// Optional, bounded read. Networking cannot redefine project readiness or
	// consume the whole collection's existing time budget.
	networkCtx, done := context.WithTimeout(check, 300*time.Millisecond)
	network, err := s.Host.TailnetPolicy(networkCtx, p.ID)
	done()
	row.TailnetState = "unavailable"
	if err == nil {
		row.TailnetState = "off"
		if network.Enabled {
			row.TailnetState = "managed"
		}
	}
}

func (s *API) inspectSpaceRow(
	r *http.Request,
	ctx context.Context,
	cookieValue string,
	v store.Session,
	p store.Project,
	terminalCount *int,
) (SpaceView, bool, bool) {
	check, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request := r.WithContext(check)

	reader, allowed, complete := s.resolveSpaceAuthority(request, v, p)
	if !allowed {
		return SpaceView{}, false, complete
	}

	row := SpaceView{
		Environment:          EnvironmentDTO(p),
		Login:                reader.login,
		Administrator:        reader.administrator && !reader.authorityUnavailable,
		AuthorityUnavailable: reader.authorityUnavailable,
		Terminals:            []TerminalView{},
	}

	if !s.inspectSpaceNative(check, p, &row) {
		complete = false
	}
	if !s.inspectSpaceTerminals(request, check, cookieValue, v, p, reader, &row, terminalCount) {
		complete = false
	}
	s.inspectSpaceTailnet(check, p, reader, &row)
	return row, true, complete
}

func appendSpaceRow(response *SpacesView, row SpaceView) bool {
	response.Items = append(response.Items, row)
	body, err := json.Marshal(response)
	if err != nil || len(body) >= 65535 {
		response.Items = response.Items[:len(response.Items)-1]
		response.Complete = false
		return false
	}
	return true
}

func (s *API) inspectSpaces(
	r *http.Request,
	ctx context.Context,
	cookieValue string,
	v store.Session,
	projects []store.Project,
) SpacesView {
	response := SpacesView{Items: []SpaceView{}, Complete: len(projects) <= 128}
	if len(projects) > 128 {
		projects = projects[:128]
	}
	terminalCount := 0
	for _, p := range projects {
		if ctx.Err() != nil || len(response.Items) >= 32 {
			response.Complete = false
			break
		}
		row, keep, complete := s.inspectSpaceRow(r, ctx, cookieValue, v, p, &terminalCount)
		if !complete {
			response.Complete = false
		}
		if !keep {
			continue
		}
		if !appendSpaceRow(&response, row) {
			break
		}
	}
	return response
}

func (s *API) verifySpacesSession(w http.ResponseWriter, ctx context.Context, cookieValue string, v store.Session) bool {
	// Logout wins publication too; never hold the registry lock over inspections.
	s.terminalMu.Lock()
	current, err := s.Store.Session(ctx, cookieValue)
	s.terminalMu.Unlock()
	if ctx.Err() != nil {
		auth.JSONError(w, 503, "spaces_unavailable", "Spaces inspection exhausted its time budget; no complete result is available.")
		return false
	}
	if err != nil || current.ContextID != v.ContextID || current.User.ID != v.User.ID || current.CSRF != v.CSRF {
		auth.JSONError(w, 401, "unauthenticated", "Session ended.")
		return false
	}
	return true
}

// Fixed bounds: 4 requests, sequential provider/helper calls, 8 seconds total,
// 2 seconds per row, 128 scanned associations, 32 published rows, <=64 KiB JSON.
// No missing repository_id overload, copied permissions, unauthorized counts,
// unbounded fan-out or denial placeholders. Limits are incomplete, not empty truth.
func (s *API) apiSpaces(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	select {
	case s.SpacesSlots <- struct{}{}:
		defer func() { <-s.SpacesSlots }()
	default:
		auth.JSONError(w, 503, "spaces_unavailable", "Spaces inspection is busy; no complete result is available.")
		return
	}
	ctx, done := context.WithTimeout(r.Context(), 8*time.Second)
	defer done()
	projects, err := s.Store.SpaceProjects(ctx)
	if err != nil {
		auth.JSONError(w, 503, "spaces_unavailable", "Could not enumerate Soda associations.")
		return
	}
	cookie, err := auth.RequestCookie(r, auth.SessionCookie)
	if err != nil {
		auth.JSONError(w, 401, "unauthenticated", "Sign in again.")
		return
	}
	response := s.inspectSpaces(r, ctx, cookie.Value, v, projects)
	if !s.verifySpacesSession(w, ctx, cookie.Value, v) {
		return
	}
	auth.JSONResponse(w, 200, response)
}
