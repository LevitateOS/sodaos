package api

import (
	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"time"
)

func (s *API) apiApplicationSnapshots(w http.ResponseWriter, r *http.Request, session store.Session) {
	if !applicationNoQuery(w, r) || !s.authorizeOperator(w, r, session) {
		return
	}
	record, ok := s.loadApplication(w, r, session)
	if !ok {
		return
	}
	request := application.SnapshotRequest{Application: record.Application, Action: "list"}
	if r.Method == http.MethodPost {
		var in struct {
			ID                string `json:"id"`
			Revision          int64  `json:"revision"`
			ExpectedContainer string `json:"expected_container"`
		}
		if !auth.DecodeAPIObject(w, r, &in) {
			return
		}
		if in.Revision != record.Application.Revision || record.Observation == nil || record.Observation.Running || record.Observation.Container == "" || record.Observation.Container != in.ExpectedContainer {
			auth.JSONError(w, 409, "application_backup_conflict", "Stop and inspect the current application before making a backup.")
			return
		}
		request.Action, request.ID, request.ExpectedContainer = "backup", in.ID, in.ExpectedContainer
	}
	if request.Validate() != nil {
		auth.JSONError(w, 400, "invalid_snapshot", "Use an application snapshot identity and the current stopped generation.")
		return
	}
	if !s.checkLifecycleSession(w, r, session) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "native_unavailable", "Application host is unavailable.")
		return
	}
	if request.Action == "backup" {
		release, acquired := s.acquireApplicationMutation(record.Application.ID)
		if !acquired {
			auth.JSONError(w, 409, "application_pending", "An application operation is in progress.")
			return
		}
		defer release()
		if !s.noApplicationRestore(w, r, record.Application.ID) {
			return
		}
	}
	ctx, cancel := operationContext(r.Context(), 4*time.Minute)
	defer cancel()
	inventory, err := s.Host.ApplicationSnapshots(ctx, request)
	if err != nil {
		auth.JSONError(w, 502, "snapshot_unconfirmed", "Snapshot operation could not be confirmed. Refresh the inventory before retrying; existing data is retained.")
		return
	}
	auth.JSONResponse(w, 200, inventory)
}
