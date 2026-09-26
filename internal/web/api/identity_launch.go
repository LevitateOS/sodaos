package api

import (
	"context"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiIdentityLaunch(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	input, ok := identityLaunchInput(w, r)
	if !ok {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok || !s.identityProjectMember(w, r, v, p) {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil {
		identityError(w, err)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	lease, err := s.Host.IdentityLaunch(r.Context(), identity.TerminalStart{ConnectionID: input.ConnectionID, ProjectID: p.ID, ActorID: v.User.ID, Login: login, Scope: TerminalCreationScope(v), Cols: input.Cols, Rows: input.Rows})
	if err != nil {
		auth.JSONError(w, 409, "identity_unavailable", "Codex did not start. Check connection status and whether it is already in use.")
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		_ = s.Identity.EndLease(ctx, v.User.ID, lease.ID)
		return
	}
	auth.JSONResponse(w, 200, struct {
		TerminalID string `json:"terminal_id"`
		LeaseID    string `json:"lease_id"`
	}{lease.ExecutionID, lease.ID})
}

type codexLaunchInput struct {
	ConnectionID string `json:"connection_id"`
	Cols         int    `json:"cols"`
	Rows         int    `json:"rows"`
}

func identityLaunchInput(w http.ResponseWriter, r *http.Request) (codexLaunchInput, bool) {
	var input codexLaunchInput
	if !auth.DecodeAPIObject(w, r, &input) {
		return input, false
	}
	if input.ConnectionID == "" || !validTerminalGeometry(input.Cols, input.Rows, "Codex") {
		auth.JSONError(w, 400, "invalid_request", "Choose a connection and valid terminal dimensions.")
		return input, false
	}
	return input, true
}
