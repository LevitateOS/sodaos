package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Controller authority stays on Fountain. This adapter maps the selected
// canonical repository to its native locator and forwards live human admission;
// it creates no local placement, background PAT authority or inference funding.
func (s *API) apiFactoryController(w http.ResponseWriter, r *http.Request, session store.Session) {
	repository, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "Choose a repository without query parameters.")
		return
	}
	access, err := s.factoryVisibleRepository(r, session, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok || !s.extensionSessionCurrent(r.Context(), r, session) {
		auth.JSONError(w, 403, "session_changed", "Refresh the current account before requesting runtime control.")
		return
	}
	native := strconv.FormatInt(access.repository.ID, 10)
	if r.Method == http.MethodGet {
		view, err := authority.RuntimeController(r.Context(), native)
		if err != nil {
			auth.JSONError(w, 503, "controller_unavailable", "Runtime control is unavailable for this repository.")
			return
		}
		if !s.extensionSessionCurrent(r.Context(), r, session) {
			auth.JSONError(w, 409, "session_changed", "Refresh the current account.")
			return
		}
		auth.JSONResponse(w, 200, view)
		return
	}
	if !s.factoryOwner(w, r, session, repository) {
		return
	}
	if r.Pattern == "POST /api/repositories/{repositoryID}/factory/controller/operations" {
		var request struct {
			sdk.ApplianceOperationAuthorization
			PolicyRevision *int64 `json:"policy_revision,omitempty"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		var trailing any
		if decoder.Decode(&request) != nil || decoder.Decode(&trailing) != io.EOF || !sdk.ValidApplianceOperationAuthorization(request.ApplianceOperationAuthorization) || request.PolicyRevision != nil && *request.PolicyRevision < 1 {
			auth.JSONError(w, 400, "invalid_controller_command", "Choose an enrolled actor and finite operation scope.")
			return
		}
		grant, err := authority.AuthorizeRuntimeOperations(r.Context(), native, request.ApplianceOperationAuthorization)
		if err != nil {
			auth.JSONError(w, 503, "controller_unconfirmed", "Authorization was not confirmed. Keep the original command and check again.")
			return
		}
		if !s.extensionSessionCurrent(r.Context(), r, session) {
			auth.JSONError(w, 409, "session_changed", "Keep the original authorization command and check its receipt.")
			return
		}
		if request.PolicyRevision != nil {
			outcome, err := s.selectRuntimeGrant(r.Context(), r, session, repository, grant, *request.PolicyRevision)
			if err != nil {
				auth.JSONError(w, 503, "policy_selection_unconfirmed", "The host authorization is retained. Keep the original command to reconcile policy selection.")
				return
			}
			auth.JSONResponse(w, 200, struct {
				sdk.ApplianceOperationGrant
				SelectionOutcome string `json:"selection_outcome"`
			}{grant, outcome})
			return
		}
		auth.JSONResponse(w, 200, grant)
		return
	}
	var request sdk.ApplianceRuntimeSelection
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	var trailing any
	if decoder.Decode(&request) != nil || decoder.Decode(&trailing) != io.EOF || !sdk.ValidApplianceRuntimeSelection(request) {
		auth.JSONError(w, 400, "invalid_controller_command", "Use the captured runtime, installation, revision and finite command.")
		return
	}
	receipt, err := authority.SelectRuntimeController(r.Context(), native, request)
	if err != nil {
		auth.JSONError(w, 503, "controller_unconfirmed", "Runtime selection was not confirmed. Keep its original command and check again.")
		return
	}
	if !s.extensionSessionCurrent(r.Context(), r, session) {
		auth.JSONError(w, 409, "session_changed", "The session changed. Keep the original command and check its receipt.")
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

// Selecting an immutable grant for new work is separate from funding. The
// captured policy revision prevents a replay from overwriting a concurrent
// human policy change. Already selected references prove adoption without
// another policy write; old run admissions retain their original references.
func (s *API) selectRuntimeGrant(ctx context.Context, r *http.Request, session store.Session, repository int64, g sdk.ApplianceOperationGrant, expected int64) (string, error) {
	policy, err := s.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		return "", err
	}
	actor, err := s.FountainIdentities.LocalActor(ctx, g.ActorID)
	if err != nil {
		return "", err
	}
	token, err := strconv.ParseInt(g.TokenID, 10, 64)
	if err != nil {
		return "", err
	}
	selected, applicable := true, false
	if slices.Contains(g.Operations, "execution.launch") {
		applicable = true
		selected = policy.Execution != nil && policy.Execution.ControllerGrantID == g.ID
	}
	for _, ref := range []factory.ActorBindingRef{policy.Publish, policy.Create, policy.Review, policy.Resolve, policy.Merge} {
		if ref.ActorID == int64(actor) && ref.TokenID == token && slices.Contains(g.Operations, ref.Kind) {
			applicable = true
			if ref.ControllerGrantID != g.ID {
				selected = false
			}
		}
	}
	if !applicable {
		return "refused", nil
	}
	if selected {
		return "applied", nil
	}
	if !g.Active || policy.Revision != expected {
		return "refused", nil
	}
	if s.Coordinator == nil {
		return "", errors.New("policy controller unavailable")
	}
	human, err := s.FountainIdentities.LocalActor(ctx, strconv.FormatInt(session.User.ID, 10))
	if err != nil {
		return "", err
	}
	policy.GrantedBy = int64(human)
	if slices.Contains(g.Operations, "execution.launch") {
		policy.Execution = &factory.ActorBindingRef{ControllerGrantID: g.ID, TokenID: token, ActorID: int64(actor), Kind: "execution.launch"}
	}
	for _, ref := range []*factory.ActorBindingRef{&policy.Publish, &policy.Create, &policy.Review, &policy.Resolve, &policy.Merge} {
		if ref.ActorID == int64(actor) && ref.TokenID == token && slices.Contains(g.Operations, ref.Kind) {
			ref.ControllerGrantID = g.ID
		}
	}
	if policy.AutomationReviewer != nil && policy.AutomationReviewer.ActorID == int64(actor) && policy.AutomationReviewer.TokenID == token && slices.Contains(g.Operations, policy.AutomationReviewer.Kind) {
		copy := *policy.AutomationReviewer
		copy.ControllerGrantID = g.ID
		policy.AutomationReviewer = &copy
	}
	if !s.extensionSessionCurrent(ctx, r, session) {
		return "refused", nil
	}
	_, err = s.Coordinator.ApplyPolicy(ctx, factoryPrincipal(session), policy)
	if errors.Is(err, store.ErrStaleRevision) {
		return "refused", nil
	}
	if err != nil {
		return "", err
	}
	return "applied", nil
}
