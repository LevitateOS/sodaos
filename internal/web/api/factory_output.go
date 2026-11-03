package api

import (
	"context"

	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/host"

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
	output    *factoryservice.OutputAttachment
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
	if s.FountainIdentities == nil {
		auth.JSONError(w, 503, "identity_unavailable", "Factory output identity is unavailable.")
		return factoryViewerIdentity{}, false
	}
	actor, err := s.FountainIdentities.LocalActor(r.Context(), authority.Actor.ID)
	if err != nil {
		auth.JSONError(w, 503, "identity_unavailable", "Factory output identity is unavailable.")
		return factoryViewerIdentity{}, false
	}
	identity.output, err = s.factoryOutputReader(r, identity, actor).Open(r.Context(), actor, runID)
	if err != nil {
		switch {
		case errors.Is(err, factoryservice.ErrRunOutputAccess):
			auth.JSONError(w, 403, "repository_write_required", "Current repository write permission is required.")
		case errors.Is(err, factoryservice.ErrRunOutputChanged):
			auth.JSONError(w, 409, "view_conflict", "Run output scope changed; refresh the inventory.")
		default:
			auth.JSONError(w, 503, "output_unavailable", "Factory output admission is unavailable.")
		}
		return factoryViewerIdentity{}, false
	}
	return identity, true
}

func (s *API) factoryOutputReader(r *http.Request, identity factoryViewerIdentity, admitted collaboration.ActorID) factoryservice.RunOutput {
	return factoryservice.RunOutput{Store: s.Store,
		Scope: func(ctx context.Context, actor collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
			if actor != admitted || run.ID != identity.runID || run.ProjectID != identity.project.ID {
				return 0, factoryservice.ErrRunOutputChanged
			}
			ref, err := s.Store.CollaborationRepositoryForNativeProject(ctx, run.ProjectID)
			return ref.ID, err
		},
		Access: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, run factory.Run) error {
			if actor != admitted || !s.factoryViewerCurrent(ctx, r, identity) {
				return factoryservice.ErrRunOutputAccess
			}
			return nil
		},
		Source: func(ctx context.Context, actor collaboration.ActorID, run factory.Run, cursor int64, limit int) (project.FactoryOutputState, error) {
			if s.Host == nil {
				return project.FactoryOutputState{}, factoryservice.ErrRunOutputUnavailable
			}
			out, err := s.Host.FactoryOutput(ctx, project.FactoryOutput{Project: run.ProjectID, ID: run.ID, Offset: cursor, Limit: limit})
			if errors.Is(err, host.ErrRunNotFound) {
				return project.FactoryOutputState{}, factoryservice.ErrRunOutputMissing
			}
			if errors.Is(err, host.ErrRunStale) {
				return project.FactoryOutputState{}, factoryservice.ErrRunOutputChanged
			}
			return out, err
		},
	}
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

func validNativeOutputHandshake(in factoryhttpapi.OutputHandshake, identity factoryViewerIdentity) bool {
	return in.RepositoryID == strconv.FormatInt(identity.project.RepositoryID, 10) &&
		in.SessionGeneration == identity.authority.SessionGeneration
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
	transport := factoryhttpapi.Output{Attachment: identity.output,
		Handshake: func(in factoryhttpapi.OutputHandshake) bool { return validNativeOutputHandshake(in, identity) },
		Current:   func(ctx context.Context) bool { return s.factoryViewerCurrent(ctx, r, identity) },
	}
	transport.Serve(ctx, conn)
}
