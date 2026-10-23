package api

import (
	"errors"
	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"time"
)

type applicationRestoreInput struct {
	Action            string `json:"action"`
	ID                string `json:"id"`
	Snapshot          string `json:"snapshot"`
	Revision          int64  `json:"revision"`
	ExpectedContainer string `json:"expected_container"`
	Confirm           bool   `json:"confirm"`
}

func (s *API) noApplicationRestore(w http.ResponseWriter, r *http.Request, app string) bool {
	pending, err := s.Store.PendingApplicationRestore(r.Context(), app)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Application recovery state is unavailable.")
		return false
	}
	if pending != nil {
		auth.JSONError(w, 409, "application_restore_pending", "A Restore operation is pending. Reconcile or abort it before acting on the application.")
		return false
	}
	return true
}

func (s *API) apiApplicationRestore(w http.ResponseWriter, r *http.Request, session store.Session) {
	if !applicationNoQuery(w, r) || !s.authorizeOperator(w, r, session) {
		return
	}
	record, ok := s.loadApplication(w, r, session)
	if !ok {
		return
	}
	var input applicationRestoreInput
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	if !input.Confirm || (input.Action != "restore" && input.Action != "abort") {
		auth.JSONError(w, 400, "restore_confirmation_required", "Explicitly confirm Restore or abort.")
		return
	}
	release, acquired := s.acquireApplicationMutation(record.Application.ID)
	if !acquired {
		auth.JSONError(w, 409, "application_pending", "An application operation is in progress.")
		return
	}
	defer release()
	if !s.checkLifecycleSession(w, r, session) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "native_unavailable", "Application host is unavailable.")
		return
	}
	op, err := s.Store.ApplicationRestore(r.Context(), record.Application.ID, input.ID)
	if errors.Is(err, store.ErrNotFound) {
		if input.Action == "abort" {
			auth.JSONError(w, 404, "restore_not_found", "Restore operation was not found.")
			return
		}
		var admitted bool
		op, admitted = s.beginApplicationRestore(w, r, record, input)
		if !admitted {
			return
		}
	} else if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Restore records are unavailable.")
		return
	}
	if op.Snapshot != input.Snapshot || op.Previous.Revision != input.Revision || op.ExpectedContainer != input.ExpectedContainer {
		auth.JSONError(w, 409, "restore_intent_changed", "Use the original Restore operation and its recorded inputs.")
		return
	}
	if op.State != "pending" {
		if (op.State == "complete" && input.Action == "abort") || (op.State == "aborted" && input.Action == "restore") {
			auth.JSONError(w, 409, "restore_already_settled", "This Restore operation is already settled. Inspect the current application; a new Restore requires a fresh operation.")
			return
		}
		current, visible := s.loadApplication(w, r, session)
		if visible {
			auth.JSONResponse(w, 200, s.applicationView(r, session, current, true))
		}
		return
	}
	request := application.Request{Application: op.Next, Action: "restore", ExpectedContainer: op.ExpectedContainer, Restore: &op}
	abort := input.Action == "abort"
	if abort {
		request.Action, request.Application = "restore_abort", op.Previous
	}
	ctx, cancel := operationContext(r.Context(), 4*time.Minute)
	defer cancel()
	observed, err := s.Host.Application(ctx, request)
	if err != nil {
		auth.JSONError(w, 502, "restore_unconfirmed", "Restore outcome is unconfirmed. Both data identities are retained; refresh and reconcile this same operation before starting the app.")
		return
	}
	if err := s.Store.SettleApplicationRestore(ctx, record.AdministratorID, record.Application.ID, op.ID, observed, abort); err != nil {
		auth.JSONError(w, 502, "restore_unconfirmed", "Native Restore was observed but its admission could not be recorded. Refresh and reconcile this same operation.")
		return
	}
	current, visible := s.loadApplication(w, r, session)
	if visible {
		auth.JSONResponse(w, 200, s.applicationView(r, session, current, true))
	}
}

func (s *API) beginApplicationRestore(w http.ResponseWriter, r *http.Request, record application.Record, input applicationRestoreInput) (application.RestoreOperation, bool) {
	var empty application.RestoreOperation
	if !s.noApplicationRestore(w, r, record.Application.ID) {
		return empty, false
	}
	if input.Revision != record.Application.Revision || record.Observation == nil || record.Observation.Running || record.Observation.Container != input.ExpectedContainer || input.ExpectedContainer == "" {
		auth.JSONError(w, 409, "application_restore_conflict", "Stop and inspect the current application before restoring.")
		return empty, false
	}
	data, err := application.NewIdentity()
	if err != nil {
		auth.JSONError(w, 503, "restore_unavailable", "A fresh data identity could not be reserved.")
		return empty, false
	}
	op := application.RestoreOperation{ID: input.ID, Snapshot: input.Snapshot, Previous: record.Application, Next: record.Application, ExpectedContainer: input.ExpectedContainer, State: "pending"}
	op.Next.Revision++
	op.Next.Data = data
	if op.Validate() != nil {
		auth.JSONError(w, 400, "invalid_restore", "Use opaque Restore and snapshot identities with the current application.")
		return empty, false
	}
	ctx, cancel := operationContext(r.Context(), 30*time.Second)
	defer cancel()
	inventory, err := s.Host.ApplicationSnapshots(ctx, application.SnapshotRequest{Application: record.Application, Action: "inspect", ID: input.Snapshot})
	if err != nil || len(inventory.Snapshots) != 1 || !inventory.Snapshots[0].Matches(record.Application) {
		auth.JSONError(w, 409, "snapshot_unavailable", "Choose a completed backup of this exact application bundle, owner and address.")
		return empty, false
	}
	if err := s.Store.BeginApplicationRestore(ctx, record.AdministratorID, op); err != nil {
		auth.JSONError(w, 409, "application_restore_conflict", "Application state changed. Refresh before reserving Restore.")
		return empty, false
	}
	return op, true
}
