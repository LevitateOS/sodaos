package api

import (
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiFactoryActivity(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	validPart := func(part string) bool {
		return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, "/\\%?#") && !strings.ContainsFunc(part, func(c rune) bool { return unicode.IsSpace(c) || unicode.IsControl(c) })
	}
	owner, name := access.repository.Owner.Login, access.repository.Name
	if !validPart(owner) || !validPart(name) {
		auth.JSONError(w, 503, "invalid_native_metadata", "Repository metadata cannot produce a local link.")
		return
	}
	if r.Method == http.MethodPut {
		if !s.factoryOwner(w, r, v, repository) || !s.checkLifecycleSession(w, r, v) {
			return
		}
		var in struct {
			ExpectedRevision *int64 `json:"expected_revision"`
			Settings         *struct {
				IssueCreated  *bool `json:"issue_created"`
				PRCreated     *bool `json:"pr_created"`
				VerifiedMerge *bool `json:"verified_merge"`
			} `json:"settings"`
		}
		if !auth.DecodeAPIObject(w, r, &in) {
			return
		}
		if in.ExpectedRevision == nil || *in.ExpectedRevision < 0 || in.Settings == nil || in.Settings.IssueCreated == nil || in.Settings.PRCreated == nil || in.Settings.VerifiedMerge == nil {
			auth.JSONError(w, 400, "invalid_activity_settings", "A current revision and all three activity settings are required.")
			return
		}
		settings := factory.ActivitySettings{IssueCreated: *in.Settings.IssueCreated, PRCreated: *in.Settings.PRCreated, VerifiedMerge: *in.Settings.VerifiedMerge}
		err = s.Store.SetActivitySettings(r.Context(), repository, v.User.ID, *in.ExpectedRevision, settings)
		if errors.Is(err, store.ErrStaleRevision) {
			auth.JSONError(w, 412, "stale_revision", "Activity settings changed; refresh and retry.")
			return
		}
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Activity settings are unavailable.")
			return
		}
	}
	activity, err := s.Store.RepositoryActivity(r.Context(), repository, "/"+owner+"/"+name)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Repository activity is unavailable.")
		return
	}
	auth.JSONResponse(w, 200, activity)
}
