package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

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
