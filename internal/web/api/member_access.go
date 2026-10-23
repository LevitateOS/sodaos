package api

import (
	"errors"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"strconv"
)

func (s *API) memberAccessAuthority(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) bool {
	if v.User.ID == s.Config.OperatorID {
		return s.authorizeOperator(w, r, v)
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if !allowed {
		auth.JSONError(w, 403, "owner_required", "Only the current repository owner, organization owner or Soda operator can revoke or restore Project access.")
		return false
	}
	return s.checkLifecycleSession(w, r, v)
}

func (s *API) apiMemberAccess(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if !s.memberAccessAuthority(w, r, v, p) {
		return
	}
	uid, ok := auth.PositiveID(r.PathValue("member"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_member", "Choose one Project member.")
		return
	}
	login, err := s.Store.MemberIdentity(r.Context(), p.ID, uid)
	if errors.Is(err, store.ErrNotFound) {
		auth.JSONError(w, 404, "not_found", "Project member not found.")
		return
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read Project membership.")
		return
	}
	record, err := s.Store.MemberAccess(r.Context(), p.ID, uid)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not inspect Project access.")
		return
	}
	nativeRequest := project.MemberAccess{Project: p.ID, Login: login, Identity: uid, Action: "inspect", Revision: record.Revision}
	if r.Method == http.MethodGet {
		native, nativeErr := s.Host.MemberAccess(r.Context(), nativeRequest)
		fingerprints := []string{}
		if nativeErr == nil {
			record.Native = &native
			var valid bool
			fingerprints, valid = nativeAccessFingerprints(w, native.Keys)
			if !valid {
				return
			}
		}
		auth.JSONResponse(w, 200, struct {
			UserID                string             `json:"user_id"`
			Login                 string             `json:"login"`
			Access                store.MemberAccess `json:"access"`
			NativeUnavailable     bool               `json:"native_unavailable"`
			InstalledFingerprints []string           `json:"installed_fingerprints"`
		}{strconv.FormatInt(uid, 10), login, record, nativeErr != nil, fingerprints})
		return
	}
	var input struct {
		Action   string `json:"action"`
		Revision int64  `json:"revision"`
		Confirm  bool   `json:"confirm"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	if !input.Confirm || input.Revision < 0 || (input.Action != "revoke" && input.Action != "restore") {
		auth.JSONError(w, 400, "confirmation_required", "Preview this member and confirm Revoke Project access or Restore Project access.")
		return
	}
	opCtx, cancel := operationContext(r.Context(), stopOperationTimeout)
	defer cancel()
	// Serialize the complete fence lifetime. Otherwise an earlier confirmed
	// operation could remove a newer request's startup fence after its DB hold.
	s.memberAccessMu.Lock()
	defer s.memberAccessMu.Unlock()
	// Recheck current native owner and session after preview I/O and waiting.
	if !s.memberAccessAuthority(w, r, v, p) {
		return
	}
	if s.MemberAccessFence == nil || s.MemberAccessFence(p.ID, uid, true) != nil {
		auth.JSONError(w, 503, "access_fence_unavailable", "Could not protect Project restart. No access change was dispatched; ask the operator to inspect.")
		return
	}
	record, err = s.Store.BeginMemberAccess(opCtx, p.ID, uid, input.Revision, input.Action)
	if errors.Is(err, store.ErrStaleRevision) {
		if current, readErr := s.Store.MemberAccess(opCtx, p.ID, uid); readErr == nil && (!current.Held || current.Complete) {
			_ = s.MemberAccessFence(p.ID, uid, false)
		}
		auth.JSONError(w, 409, "access_changed", "Project access changed or revocation is unfinished. Refresh its state before acting.")
		return
	}
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not record the Project access hold. No native operation was dispatched.")
		return
	}
	nativeRequest.Action, nativeRequest.Revision = input.Action, record.Revision
	native, nativeErr := s.Host.MemberAccess(opCtx, nativeRequest)
	if nativeErr == nil {
		err = s.Store.FinishMemberAccess(opCtx, p.ID, uid, record, native)
		if err == nil {
			record, err = s.Store.MemberAccess(opCtx, p.ID, uid)
			if err == nil {
				err = s.MemberAccessFence(p.ID, uid, false)
			}
		}
	}
	status := 200
	if nativeErr != nil || err != nil {
		status = 202
	}
	auth.JSONResponse(w, status, struct {
		UserID            string             `json:"user_id"`
		Login             string             `json:"login"`
		Access            store.MemberAccess `json:"access"`
		NativeUnavailable bool               `json:"native_unavailable"`
		Pending           bool               `json:"pending"`
	}{strconv.FormatInt(uid, 10), login, record, nativeErr != nil, status == 202})
}
