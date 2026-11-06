package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	extensions "forgejo.org/extension-sdk"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type extensionAuthorityKey struct{}

func requestExtensionAuthority(r *http.Request) (extensions.Authority, bool) {
	authority, ok := r.Context().Value(extensionAuthorityKey{}).(extensions.Authority)
	return authority, ok
}

func (s *API) extensionProtected(next func(http.ResponseWriter, *http.Request, store.Session), methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !auth.AllowAPIMethod(w, r, methods) {
			return
		}
		authority, ok := extensionProductAuthority(w, r)
		if !ok {
			return
		}
		if !extensionGenerationCurrent(w, r, authority) {
			return
		}
		actorID, ok := s.extensionProductActor(w, authority)
		if !ok {
			return
		}
		if !s.extensionProductMutation(w, r) {
			return
		}
		user, ok := s.extensionProductUser(w, r, actorID, authority.Actor.Username)
		if !ok {
			return
		}
		ctx := context.WithValue(r.Context(), extensionAuthorityKey{}, authority)
		session := store.Session{User: user, ContextID: authority.SessionGeneration}
		next(w, r.WithContext(ctx), session)
	}
}

func extensionProductAuthority(w http.ResponseWriter, r *http.Request) (extensions.Authority, bool) {
	authority, err := auth.ExtensionAuthority(r)
	contributionOK := err == nil && auth.ExtensionContribution(authority.ExtensionID, authority.Contribution)
	productOK := contributionOK && extensionProductContribution(r.URL.Path, authority.Contribution)
	if err != nil || !contributionOK || !productOK {
		denial := "authority"
		if err == nil {
			denial = "contribution"
			if contributionOK {
				denial = "product"
			}
		}
		logExtensionDenial(r, denial, err, authority.Contribution)
		auth.JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
		return extensions.Authority{}, false
	}
	return authority, true
}

// logExtensionDenial records which authority sub-check refused a request.
// The response stays a bounded 403; contribution fields are routing
// metadata, never credentials.
func logExtensionDenial(r *http.Request, denial string, err error, contribution extensions.Contribution) {
	slog.Warn("extension authority denied",
		"path", r.URL.Path,
		"denial", denial,
		"authority_error", errorString(err),
		"contribution_kind", contribution.Kind,
		"contribution_scope", contribution.Scope,
		"contribution_id", contribution.ID,
		"contribution_action", contribution.Action,
	)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func extensionGenerationCurrent(w http.ResponseWriter, r *http.Request, authority extensions.Authority) bool {
	generations := r.Header.Values(extensions.SessionGenerationHeader)
	if len(generations) != 1 || generations[0] != authority.SessionGeneration {
		auth.JSONError(w, http.StatusConflict, "extension_session_changed", "The workspace page is no longer current.")
		return false
	}
	return true
}

func (s *API) extensionProductActor(w http.ResponseWriter, authority extensions.Authority) (int64, bool) {
	actorID, valid := auth.PositiveID(authority.Actor.ID)
	if !valid || s.Store == nil {
		auth.JSONError(w, http.StatusForbidden, "invalid_actor", "Native actor is invalid.")
		return 0, false
	}
	return actorID, true
}

func (s *API) extensionProductMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if !s.extensionMutationOrigin(r) {
			auth.JSONError(w, http.StatusForbidden, "invalid_origin", "Same-origin Soda mutation required.")
			return false
		}
		limit := int64(auth.APIBodyLimit)
		if factoryhttpapi.IsHumanCommentSubmission(r) {
			limit = factoryhttpapi.HumanCommentRequestLimit
		}
		if factoryhttpapi.IsHumanEditSubmission(r) {
			limit = factoryhttpapi.HumanEditRequestLimit
		}
		if factoryhttpapi.IsHumanInlineSubmission(r) {
			limit = factoryhttpapi.HumanInlineRequestLimit
		}
		if factoryhttpapi.IsHumanReviewSubmission(r) {
			limit = factoryhttpapi.HumanReviewRequestLimit
		}
		if factoryhttpapi.IsHumanMergeSubmission(r) {
			limit = factoryhttpapi.HumanMergeRequestLimit
		}
		if factoryhttpapi.IsHumanLockSubmission(r) {
			limit = factoryhttpapi.HumanLockRequestLimit
		}
		if factoryhttpapi.IsHumanPullCreateSubmission(r) {
			limit = factoryhttpapi.HumanPullCreateRequestLimit
		}
		if factoryhttpapi.IsHumanIssueCreateSubmission(r) {
			limit = factoryhttpapi.HumanIssueCreateRequestLimit
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}
	return true
}

func (s *API) extensionProductUser(w http.ResponseWriter, r *http.Request, actorID int64, username string) (store.User, bool) {
	if err := s.Store.UpsertUser(r.Context(), store.User{ID: actorID, Login: username}); err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	user, err := s.Store.User(r.Context(), actorID)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	return user, true
}

func extensionProductContribution(path string, contribution extensions.Contribution) bool {
	if path == "/api/settings/tailnet" || strings.HasPrefix(path, "/api/settings/tailnet/") {
		return auth.ExtensionContribution("tailnet", contribution)
	}
	// The AI accounts panel mounts on every page, so it reaches only the
	// person's own provider connections and sign-ins.
	if extensionProvidersPanelContribution(contribution) {
		for _, family := range []string{"/api/identity/connections", "/api/identity/enrollments", "/api/identity/jev"} {
			if path == family || strings.HasPrefix(path, family+"/") {
				return true
			}
		}
		return false
	}
	return extensionPageContribution(contribution, "spaces") || extensionPageContribution(contribution, "feed") || extensionPageContribution(contribution, "applications") ||
		contribution.Kind == "page" && (contribution.ID == "repository-feed" || contribution.ID == "repository-settings") && contribution.Scope == "repository" || extensionWorkspacePanelContribution(contribution)
}

func extensionPageContribution(contribution extensions.Contribution, id string) bool {
	return contribution.Kind == "page" && contribution.Scope == "global" && contribution.ID == id
}

func extensionProvidersPanelContribution(contribution extensions.Contribution) bool {
	return contribution.Kind == "panel" && contribution.ID == "providers" && contribution.Scope == "panel"
}

func extensionWorkspacePanelContribution(contribution extensions.Contribution) bool {
	return contribution.Kind == "panel" && contribution.ID == "workspace" && contribution.Scope == "panel"
}

func (s *API) extensionMutationOrigin(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if s.Config.ForgejoURL == "" || len(origins) != 1 || origins[0] != s.Config.ForgejoURL {
		return false
	}
	sites := r.Header.Values("Sec-Fetch-Site")
	return len(sites) <= 1 && (len(sites) == 0 || sites[0] == "same-origin")
}

func (s *API) extensionSessionCurrent(ctx context.Context, r *http.Request, session store.Session) bool {
	initial, ok := requestExtensionAuthority(r)
	if !ok {
		return false
	}
	current, err := auth.ExtensionAuthority(r.WithContext(ctx))
	return err == nil && extensionAuthorityCurrent(current, initial) &&
		current.Actor.ID == strconv.FormatInt(session.User.ID, 10) && session.ContextID == initial.SessionGeneration
}

func extensionAuthorityCurrent(current, initial extensions.Authority) bool {
	return current.ExtensionID == initial.ExtensionID && current.InstanceID == initial.InstanceID &&
		current.SessionGeneration == initial.SessionGeneration && current.Contribution == initial.Contribution &&
		current.Actor == initial.Actor
}

func (s *API) registerExtensionProductRoutes(mux *http.ServeMux) {
	protect := s.extensionProtected
	mux.HandleFunc("GET /api/applications", protect(s.apiApplications, http.MethodGet))
	mux.HandleFunc("POST /api/applications", protect(s.apiApplications, http.MethodPost))
	mux.HandleFunc("GET /api/applications/setups", protect(s.apiApplicationSetups, http.MethodGet))
	mux.HandleFunc("POST /api/applications/setups", protect(s.apiApplicationSetups, http.MethodPost))
	mux.HandleFunc("POST /api/applications/setups/{setup}/actions", protect(s.apiApplicationSetupAction, http.MethodPost))
	mux.HandleFunc("GET /api/applications/{id}", protect(s.apiApplication, http.MethodGet))
	mux.HandleFunc("POST /api/applications/{id}/actions", protect(s.apiApplicationAction, http.MethodPost))
	mux.HandleFunc("POST /api/applications/{id}/project", protect(s.apiApplicationProject, http.MethodPost))
	mux.HandleFunc("GET /api/applications/{id}/snapshots", protect(s.apiApplicationSnapshots, http.MethodGet))
	mux.HandleFunc("POST /api/applications/{id}/snapshots", protect(s.apiApplicationSnapshots, http.MethodPost))
	mux.HandleFunc("POST /api/applications/{id}/restores", protect(s.apiApplicationRestore, http.MethodPost))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/application-builds", protect(s.apiApplicationBuilds, http.MethodGet))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/application-builds", protect(s.apiApplicationBuilds, http.MethodPost))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/application-builds/{build}/setup", protect(s.apiApplicationBuildSetup, http.MethodPost))
	mux.HandleFunc("GET /api/factory/departments", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/native-repositories/resolve", protect(s.apiFactoryResolveNativeRepository, http.MethodPost))
	mux.HandleFunc("GET /api/factory/identity", protect(s.apiFactoryIdentity, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/issues", protect(s.apiFactoryCollaborationIssues, http.MethodGet))
	mergeBase := "/api/factory/collaboration/repositories/{repository}/pulls/{number}/merges"
	mux.HandleFunc("POST "+mergeBase, protect(s.apiFactoryHumanMerges, http.MethodPost))
	mux.HandleFunc("GET "+mergeBase+"/commands/{command}", protect(s.apiFactoryHumanMerges, http.MethodGet))
	mux.HandleFunc("POST "+mergeBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanMerges, http.MethodPost))
	reviewerBase := "/api/factory/collaboration/repositories/{repository}/pulls/{number}/review-requests"
	mux.HandleFunc("POST "+reviewerBase, protect(s.apiFactoryHumanReviewRequests, http.MethodPost))
	mux.HandleFunc("GET "+reviewerBase+"/commands/{command}", protect(s.apiFactoryHumanReviewRequests, http.MethodGet))
	mux.HandleFunc("POST "+reviewerBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanReviewRequests, http.MethodPost))
	threadBase := "/api/factory/collaboration/repositories/{repository}/pulls/{number}/thread-resolutions"
	mux.HandleFunc("POST "+threadBase, protect(s.apiFactoryHumanThreads, http.MethodPost))
	mux.HandleFunc("GET "+threadBase+"/commands/{command}", protect(s.apiFactoryHumanThreads, http.MethodGet))
	mux.HandleFunc("POST "+threadBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanThreads, http.MethodPost))
	inlineBase := "/api/factory/collaboration/repositories/{repository}/pulls/{number}/inline-comments"
	mux.HandleFunc("POST "+inlineBase, protect(s.apiFactoryHumanInlines, http.MethodPost))
	mux.HandleFunc("GET "+inlineBase+"/commands/{command}", protect(s.apiFactoryHumanInlines, http.MethodGet))
	mux.HandleFunc("POST "+inlineBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanInlines, http.MethodPost))
	reviewBase := "/api/factory/collaboration/repositories/{repository}/pulls/{number}/reviews"
	mux.HandleFunc("POST "+reviewBase, protect(s.apiFactoryHumanReviews, http.MethodPost))
	mux.HandleFunc("GET "+reviewBase+"/commands/{command}", protect(s.apiFactoryHumanReviews, http.MethodGet))
	mux.HandleFunc("POST "+reviewBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanReviews, http.MethodPost))
	pullCreateBase := "/api/factory/collaboration/repositories/{repository}/pulls"
	pullCreateCommand := "/api/factory/collaboration/repositories/{repository}/pull-creation/commands/{command}"
	mux.HandleFunc("POST "+pullCreateBase, protect(s.apiFactoryHumanPullCreates, http.MethodPost))
	mux.HandleFunc("GET "+pullCreateCommand, protect(s.apiFactoryHumanPullCreates, http.MethodGet))
	mux.HandleFunc("POST "+pullCreateCommand+"/cancel", protect(s.apiFactoryHumanPullCreates, http.MethodPost))
	createBase := "/api/factory/collaboration/repositories/{repository}/issues"
	createCommand := "/api/factory/collaboration/repositories/{repository}/issue-creation/commands/{command}"
	mux.HandleFunc("POST "+createBase, protect(s.apiFactoryHumanIssueCreates, http.MethodPost))
	mux.HandleFunc("GET "+createCommand, protect(s.apiFactoryHumanIssueCreates, http.MethodGet))
	mux.HandleFunc("POST "+createCommand+"/cancel", protect(s.apiFactoryHumanIssueCreates, http.MethodPost))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/issues/{number}", protect(s.apiFactoryCollaborationIssues, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/issues/{number}/comments", protect(s.apiFactoryCollaborationDiscussions, http.MethodGet))
	for _, kind := range []string{"issues", "pulls"} {
		base := "/api/factory/collaboration/repositories/{repository}/" + kind + "/{number}/comments"
		mux.HandleFunc("POST "+base, protect(s.apiFactoryHumanComments, http.MethodPost))
		mux.HandleFunc("GET "+base+"/commands/{command}", protect(s.apiFactoryHumanComments, http.MethodGet))
		mux.HandleFunc("POST "+base+"/commands/{command}/cancel", protect(s.apiFactoryHumanComments, http.MethodPost))
		editBase := "/api/factory/collaboration/repositories/{repository}/" + kind + "/{number}/edits"
		mux.HandleFunc("POST "+editBase, protect(s.apiFactoryHumanEdits, http.MethodPost))
		mux.HandleFunc("GET "+editBase+"/commands/{command}", protect(s.apiFactoryHumanEdits, http.MethodGet))
		mux.HandleFunc("POST "+editBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanEdits, http.MethodPost))
		lockBase := "/api/factory/collaboration/repositories/{repository}/" + kind + "/{number}/locks"
		mux.HandleFunc("POST "+lockBase, protect(s.apiFactoryHumanLocks, http.MethodPost))
		mux.HandleFunc("GET "+lockBase+"/commands/{command}", protect(s.apiFactoryHumanLocks, http.MethodGet))
		mux.HandleFunc("POST "+lockBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanLocks, http.MethodPost))
		stateBase := "/api/factory/collaboration/repositories/{repository}/" + kind + "/{number}/state"
		mux.HandleFunc("POST "+stateBase, protect(s.apiFactoryHumanStates, http.MethodPost))
		mux.HandleFunc("GET "+stateBase+"/commands/{command}", protect(s.apiFactoryHumanStates, http.MethodGet))
		mux.HandleFunc("POST "+stateBase+"/commands/{command}/cancel", protect(s.apiFactoryHumanStates, http.MethodPost))
	}
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/threads", protect(s.apiFactoryCollaborationThreads, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/threads/{thread}", protect(s.apiFactoryCollaborationThreads, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/threads/{thread}/selection", protect(s.apiFactoryHumanThreads, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/threads/{thread}/comments", protect(s.apiFactoryCollaborationThreads, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/reviews", protect(s.apiFactoryCollaborationReviews, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/reviews/{review}", protect(s.apiFactoryCollaborationReviews, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories", protect(s.apiFactoryCollaborationRepositoryCatalog, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/capabilities", protect(s.apiFactoryCollaborationCapabilities, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/issues/{number}/capabilities", protect(s.apiFactoryCollaborationCapabilities, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/capabilities", protect(s.apiFactoryCollaborationCapabilities, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/checks", protect(s.apiFactoryCollaborationChecks, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/checks/jobs", protect(s.apiFactoryCollaborationChecks, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/files", protect(s.apiFactoryCollaborationPullFiles, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/commits", protect(s.apiFactoryCollaborationPullCommits, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls", protect(s.apiFactoryCollaborationPulls, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}", protect(s.apiFactoryCollaborationPulls, http.MethodGet))
	mux.HandleFunc("GET /api/factory/collaboration/repositories/{repository}/pulls/{number}/comments", protect(s.apiFactoryCollaborationDiscussions, http.MethodGet))
	mux.HandleFunc("GET /api/factory/repositories", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/repositories/{repositoryID}/identity", protect(s.apiFactoryRepositoryIdentity, http.MethodGet))
	mux.HandleFunc("POST /api/factory/departments", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("POST /api/factory/intents", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/intents", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/revisions/{revisionID}", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/revisions/{revisionID}/generation", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/revisions/{revisionID}/plan", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/revisions/{revisionID}/exploration", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/generation", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/history", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/verification-allowance", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/intents/{intentID}/verification-allowance", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/review", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/overview", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/revisions", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/intents/{intentID}/generate", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("POST /api/factory/intents/{intentID}/plan", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/intents/{intentID}/plan", protect(s.apiFactoryRequestsHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/intents/{intentID}/plan/review", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("POST /api/factory/intents/{intentID}/explore", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/repositories", protect(s.apiRepositories, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/profiles", protect(s.apiProjectProfiles, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/tailnet-options", protect(s.apiTailnetOptions, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory", protect(s.apiFactoryRepositoryHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/activity", protect(s.apiFactoryRepositoryHTTP, http.MethodGet))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/activity", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/policy", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/operator-grant", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/repair", protect(s.apiFactoryRepositoryHTTP, http.MethodGet))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/repair", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/environment-grant", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/access", protect(s.apiFactoryRepositoryHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/issues/{issueID}/plan", protect(s.apiFactoryWorkHTTP, http.MethodGet))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/sponsorships/{connection}", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("PUT /api/factory/capacity", protect(s.apiFactoryRepositoryHTTP, http.MethodPut))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/factory/actions", protect(s.apiFactoryRunHTTP, http.MethodPost))
	mux.HandleFunc("POST /api/factory/runs/{runID}/actions", protect(s.apiFactoryRunHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/runs/{runID}", protect(s.apiFactoryRunHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/issues/{issueID}", protect(s.apiFactoryWorkHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/issues/{issueID}/acceptance-preview", protect(s.apiFactoryAcceptanceHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/factory/issues/{issueID}/acceptances", protect(s.apiFactoryAcceptanceHTTP, http.MethodPost))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/factory/issues/{issueID}/withdrawal", protect(s.apiFactoryAcceptanceHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory/issues/{issueID}/assignment", protect(s.apiFactoryWorkHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/artifacts/{id}", protect(s.apiFactoryRecordsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/library", protect(s.apiFactoryRecordsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/library/item", protect(s.apiFactoryRecordsHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/feed", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/feed/subscription", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("PUT /api/factory/feed/subscription", protect(s.apiFactoryFeedHTTP, http.MethodPut))
	mux.HandleFunc("POST /api/factory/feed/posts", protect(s.apiFactoryFeedHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/feed/posts/{id}/replies", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("GET /api/factory/feed/posts/{id}", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/feed/posts/{id}/retry", protect(s.apiFactoryFeedHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/factory/feed/posts/{id}/retry/{command}", protect(s.apiFactoryFeedHTTP, http.MethodGet))
	mux.HandleFunc("POST /api/factory/feed/posts/{id}/review", protect(s.apiFactoryRequestsHTTP, http.MethodPost))
	mux.HandleFunc("GET /api/spaces", protect(s.apiSpaces, http.MethodGet))
	mux.HandleFunc("GET /api/environments", protect(s.apiEnvironments, http.MethodGet))
	mux.HandleFunc("POST /api/environments", protect(s.apiEnvironments, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}", protect(s.apiEnvironment, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/join", protect(s.apiJoinEnvironment, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/members", protect(s.apiEnvironmentMembers, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/members/{member}/access", protect(s.apiMemberAccess, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/members/{member}/access", protect(s.apiMemberAccess, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/connection", protect(s.apiConnection, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/os", protect(s.apiEnvironmentOS, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/lifecycle", protect(s.apiLifecycle, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/lifecycle", protect(s.apiLifecycle, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/access-keys", protect(s.apiAccessKeys, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/access-keys", protect(s.apiAccessKeys, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/preparation", protect(s.apiPreparation, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/preparation", protect(s.apiPreparation, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/preparation/acceptances", protect(s.apiPreparationAcceptances, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/preparation/actions", protect(s.apiPreparationActions, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/identity/launch", protect(s.apiIdentityLaunch, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/identity/connections", protect(s.apiIdentityAvailable, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/identity/grants", protect(s.apiIdentityCreateGrant, http.MethodPost))
	mux.HandleFunc("GET /api/identity/connections", protect(s.apiIdentityConnections, http.MethodGet))
	mux.HandleFunc("GET /api/identity/jev", protect(s.apiJEVStatus, http.MethodGet))
	mux.HandleFunc("POST /api/identity/jev", protect(s.apiJEVConnect, http.MethodPost))
	mux.HandleFunc("POST /api/identity/jev/disconnect", protect(s.apiJEVDisconnect, http.MethodPost))
	mux.HandleFunc("GET /api/identity/enrollments", protect(s.apiIdentityEnrollments, http.MethodGet))
	mux.HandleFunc("POST /api/identity/enrollments", protect(s.apiIdentityStartEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/identity/enrollments/{identityID}", protect(s.apiIdentityEnrollment, http.MethodGet))
	mux.HandleFunc("POST /api/identity/enrollments/{identityID}/cancel", protect(s.apiIdentityCancelEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/identity/connections/{identityID}/grants", protect(s.apiIdentityGrants, http.MethodGet))
	mux.HandleFunc("GET /api/identity/connections/{identityID}/leases", protect(s.apiIdentityLeases, http.MethodGet))
	mux.HandleFunc("POST /api/identity/connections/{identityID}/revoke", protect(s.apiIdentityRevoke, http.MethodPost))
	mux.HandleFunc("POST /api/identity/grants/{identityID}/revoke", protect(s.apiIdentityRevokeGrant, http.MethodPost))
	mux.HandleFunc("POST /api/identity/leases/{identityID}/end", protect(s.apiIdentityEndLease, http.MethodPost))
	mux.HandleFunc("GET /api/settings/tailnet", protect(s.apiTailnetSettings, http.MethodGet))
	mux.HandleFunc("POST /api/settings/tailnet/host", protect(s.apiTailnetHost, http.MethodPost))
	mux.HandleFunc("POST /api/settings/tailnet/enrollment", protect(s.apiTailnetEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/tailnet", protect(s.apiProjectTailnet, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/tailnet", protect(s.apiProjectTailnet, http.MethodPost))
}

func (s *API) extensionActor(r *http.Request, session store.Session) (extensions.Authority, bool) {
	authority, ok := requestExtensionAuthority(r)
	return authority, ok && authority.Actor.ID == strconv.FormatInt(session.User.ID, 10) && strings.EqualFold(authority.Actor.Username, session.User.Login)
}
