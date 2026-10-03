package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
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

// SpaceFactoryRun maps one recorded run to its display row: identity,
// display binding and recorded outcome. Liveness and process identity stay
// on the run status route; the collection performs no host calls.
type SpaceFactoryRun struct {
	Outcome    factory.Outcome `json:"outcome,omitempty"`
	ID         string          `json:"id"`
	Role       string          `json:"role"`
	Issue      string          `json:"issue,omitempty"`
	Attempt    string          `json:"attempt,omitempty"`
	Reconciled bool            `json:"reconciled"`
}

// spaceAuthorityView exposes the effective factory verdict for the row's
// repository: whether dispatch may proceed and which authorization is
// missing or withdrawn. It carries reason codes only, never credentials.
type spaceAuthorityView struct {
	Missing      []string `json:"missing"`
	Effective    bool     `json:"effective"`
	DispatchOpen bool     `json:"dispatch_open"`
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

func (s *API) resolveSpaceAuthority(request *http.Request, v store.Session, p store.Project) (environmentReader, bool, bool) {
	reader, err := s.readEnvironmentAuthority(request, v, p)
	if err != nil {
		complete := errors.Is(err, errRepositoryDenied)
		return reader, false, complete
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
	v store.Session,
	p store.Project,
	reader environmentReader,
	row *SpaceView,
	terminalCount *int,
) bool {
	if !reader.executionAllowed || !reader.repositoryVisible || reader.authorityUnavailable || reader.login == "" {
		return true
	}
	var items []host.TerminalState
	var err error
	items, err = s.extensionTerminalStates(request, check, v, p, reader.login, "list")
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
	v store.Session,
	p store.Project,
	runs []factory.Run,
	views map[string]factory.RunView,
	runsKnown bool,
	terminalCount *int,
	usage *spacesRunUsage,
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
		ExecutionAllowed:     reader.executionAllowed,
		Administrator:        reader.administrator && !reader.authorityUnavailable,
		AuthorityUnavailable: reader.authorityUnavailable,
		Terminals:            []TerminalView{},
		FactoryRuns:          []SpaceFactoryRun{},
	}

	if !s.inspectSpaceNative(check, p, &row) {
		complete = false
	}
	if !s.inspectSpaceTerminals(request, check, v, p, reader, &row, terminalCount) {
		complete = false
	}
	s.inspectSpaceTailnet(check, p, reader, &row)
	// Preparation readiness is a durable store read, never a native call.
	row.Preparation = s.inspectSpacePreparation(check, p.ID)
	row.FactoryAuthority = s.inspectSpaceAuthority(check, p.RepositoryID)
	row.FactoryControl = s.inspectSpaceControl(check, runs, runsKnown, p.ID, p.RepositoryID)
	row.FactoryRuns, complete = inspectSpaceFactoryRuns(row.FactoryRuns, runs, views, p.ID, usage, complete)
	return row, true, complete
}

// inspectSpaceFactoryRuns maps the row's recorded runs to display rows.
// Like terminals, the collection caps published runs globally; overflow
// marks the collection incomplete rather than silently dropping truth.
// Live rows admit before settled history, so a cap filled by history
// never hides unsettled work in a later row.
func inspectSpaceFactoryRuns(
	out []SpaceFactoryRun,
	runs []factory.Run,
	views map[string]factory.RunView,
	projectID string,
	usage *spacesRunUsage,
	complete bool,
) ([]SpaceFactoryRun, bool) {
	for _, run := range runs {
		if run.ProjectID != projectID {
			continue
		}
		if run.Reconciled {
			if usage.history >= usage.historyBudget {
				complete = false
				continue
			}
			usage.history++
		} else {
			if usage.unsettled >= spacesRunCap {
				complete = false
				continue
			}
			usage.unsettled++
		}
		row := SpaceFactoryRun{ID: run.ID, Role: run.Role, Outcome: run.Outcome, Reconciled: run.Reconciled}
		if view, ok := views[run.ID]; ok {
			if view.Issue > 0 {
				row.Issue = strconv.FormatInt(view.Issue, 10)
			}
			row.Attempt = view.Attempt
		}
		out = append(out, row)
	}
	return out, complete
}

func (s *API) inspectSpaceAuthority(ctx context.Context, repository int64) *spaceAuthorityView {
	if s.Coordinator == nil {
		return nil
	}
	effective, err := s.Coordinator.EffectiveAuthority(ctx, repository)
	if err != nil {
		return nil
	}
	return &spaceAuthorityView{Missing: effective.Missing, Effective: effective.Effective, DispatchOpen: effective.DispatchOpen}
}

// spaceControlView exposes the intervention state for the row's factory:
// whether dispatch is open and why it closed, whether the owner paused,
// and how many recorded runs are unsettled. It carries reason codes and
// counts only, never credentials or run contents.
type spaceControlView struct {
	WithdrawalCause string `json:"withdrawal_cause,omitempty"`
	UnsettledRuns   int    `json:"unsettled_runs"`
	Paused          bool   `json:"paused"`
	DispatchOpen    bool   `json:"dispatch_open"`
}

func (s *API) inspectSpaceControl(ctx context.Context, runs []factory.Run, runsKnown bool, projectID string, repository int64) *spaceControlView {
	// An unreadable run listing leaves the unsettled count unknown: no
	// control rather than an authoritative zero.
	if !runsKnown {
		return nil
	}
	view := &spaceControlView{}
	if policy, err := s.Store.RepositoryPolicy(ctx, repository); err == nil {
		view.Paused = policy.Paused
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil
	}
	open, _, withdrawal, err := s.Store.DispatchState(ctx, repository)
	if err != nil {
		return nil
	}
	view.DispatchOpen = open
	if !open {
		view.WithdrawalCause = withdrawal.Cause
	}
	for _, run := range runs {
		if run.ProjectID == projectID && !run.Reconciled {
			view.UnsettledRuns++
		}
	}
	return view
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
