package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type SpaceView struct {
	TailnetState         string                `json:"tailnet_state,omitempty"`
	Environment          EnvironmentView       `json:"environment"`
	Login                string                `json:"login"`
	ExecutionAllowed     bool                  `json:"execution_allowed"`
	Administrator        bool                  `json:"environment_administrator"`
	AuthorityUnavailable bool                  `json:"authority_unavailable"`
	NativeUnavailable    bool                  `json:"native_unavailable"`
	Observed             *project.Environment  `json:"observed"`
	Terminals            []TerminalView        `json:"terminals"`
	Preparation          *spacePreparationView `json:"preparation,omitempty"`
	FactoryAuthority     *spaceAuthorityView   `json:"factory_authority,omitempty"`
	FactoryControl       *spaceControlView     `json:"factory_control,omitempty"`
	FactoryRuns          []SpaceFactoryRun     `json:"factory_runs"`
}

type SpacesView struct {
	Items             []SpaceView `json:"items"`
	Complete          bool        `json:"complete"`
	Actor             SpacesActor `json:"actor"`
	NextAfter         string      `json:"next_after,omitempty"`
	FactoryIncomplete bool        `json:"factory_incomplete,omitempty"`
}

type SpacesActor struct {
	ID    string `json:"id"`
	Login string `json:"login"`
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
	v store.Session,
	inv spacesInventory,
	after string,
) SpacesView {
	actor := SpacesActor{ID: strconv.FormatInt(v.User.ID, 10), Login: v.User.Login}
	if authority, ok := requestExtensionAuthority(r); ok {
		actor = SpacesActor{ID: authority.Actor.ID, Login: authority.Actor.Username}
	}
	response := SpacesView{Items: []SpaceView{}, Complete: true, Actor: actor}
	if !inv.factoryComplete() {
		response.Complete = false
		response.FactoryIncomplete = true
	}
	projects := inv.projects
	if len(projects) > 128 {
		projects = projects[:128]
	}
	terminalCount := 0
	usage := newSpacesRunUsage(inv.runs)
	scanned := 0
	progress := ""
	for _, p := range projects {
		if ctx.Err() != nil || len(response.Items) >= 32 {
			response.Complete = false
			break
		}
		scanned++
		row, keep, complete := s.inspectSpaceRow(r, ctx, v, p, inv.runs, inv.views, inv.runsKnown(), &terminalCount, usage)
		if !complete {
			response.Complete = false
		}
		if !keep {
			progress = p.ID
			continue
		}
		if !appendSpaceRow(&response, row) {
			// The JSON cap dropped this row: resume before it so
			// the next page still covers it under a fresh budget.
			response.Complete = false
			break
		}
		progress = p.ID
	}
	// A full fetch proves more associations remain; a short scan of a
	// partial fetch means the page stopped early. Either way the next
	// page resumes after the last covered association.
	if len(inv.projects) == spacesAssociationPage || scanned < len(inv.projects) {
		response.Complete = false
		response.NextAfter = progress
		if response.NextAfter == "" {
			response.NextAfter = after
		}
	}
	return response
}

func (s *API) verifySpacesSession(w http.ResponseWriter, ctx context.Context, r *http.Request, v store.Session) bool {
	if ctx.Err() != nil {
		auth.JSONError(w, 503, "spaces_unavailable", "Spaces inspection exhausted its time budget; no complete result is available.")
		return false
	}
	if !s.extensionSessionCurrent(ctx, r, v) {
		auth.JSONError(w, 401, "unauthenticated", "Native Forgejo session ended.")
		return false
	}
	return true
}

// Fixed bounds: 4 requests, sequential native/helper calls, 8 seconds total,
// 2 seconds per row, 128 scanned associations, 32 published rows, <=64 KiB JSON.
// No missing repository_id overload, copied permissions, unauthorized counts,
// unbounded fan-out or denial placeholders. Limits are incomplete, not empty truth.
// One optional cursor continues the bounded association scan: after names the
// last covered project identity and the page resumes past it.
func (s *API) apiSpaces(w http.ResponseWriter, r *http.Request, v store.Session) {
	after, ok := parseSpacesCursor(r)
	if !ok {
		auth.JSONError(w, 400, "invalid_request", "Only a project cursor is accepted.")
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
	inv, err := s.loadSpacesInventory(ctx, after)
	if err != nil {
		auth.JSONError(w, 503, "spaces_unavailable", "Could not enumerate Soda associations.")
		return
	}
	response := s.inspectSpaces(r, ctx, v, inv, after)
	if !s.verifySpacesSession(w, ctx, r, v) {
		return
	}
	auth.JSONResponse(w, 200, response)
}

// parseSpacesCursor admits an empty query or exactly one project cursor.
// Anything else stays a rejected alias: cursors paginate one inventory,
// they never select repositories, actors or terminals.
func parseSpacesCursor(r *http.Request) (string, bool) {
	if r.URL.RawQuery == "" && !r.URL.ForceQuery {
		return "", true
	}
	query := r.URL.Query()
	if len(query) != 1 {
		return "", false
	}
	after := query["after"]
	if len(after) != 1 || !project.ValidID(after[0]) {
		return "", false
	}
	return after[0], true
}
