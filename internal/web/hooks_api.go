package web

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) hookRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/hooks", s.apiProvider(s.apiHooks, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/hooks/{hook}", s.apiProvider(s.apiHook, "write:repository", "GET", "PATCH"))
}

type hookView struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Active           bool     `json:"active"`
	Events           []string `json:"events"`
	BranchFilter     string   `json:"branch_filter"`
	TargetConfigured bool     `json:"target_configured"`
	HasAuthorization bool     `json:"has_authorization"`
}

func hookDTO(h forgejo.Hook) hookView {
	events := append([]string{}, h.Events...)
	return hookView{strconv.FormatInt(h.ID, 10), h.Type, h.Active, events, h.BranchFilter, h.URL != "", h.Authorization != ""}
}
func validHookTarget(target string) bool {
	u, err := url.Parse(target)
	return err == nil && len(target) <= 2048 && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(target, "\r\n\x00")
}
func validHookEvents(events []string) bool {
	if len(events) == 0 || len(events) > 32 {
		return false
	}
	for _, event := range events {
		switch event {
		case "create", "delete", "fork", "push", "issues", "issues_only", "issue_assign", "issue_label", "issue_milestone", "issue_comment", "pull_request", "pull_request_only", "pull_request_assign", "pull_request_label", "pull_request_milestone", "pull_request_comment", "pull_request_review", "pull_request_review_request", "pull_request_sync", "wiki", "repository", "release", "package", "action_run_failure", "action_run_recover", "action_run_success":
		default:
			return false
		}
	}
	return true
}
func (s *Server) apiHooks(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input struct {
			URL           string   `json:"url"`
			Secret        string   `json:"secret"`
			Authorization string   `json:"authorization"`
			Events        []string `json:"events"`
			Active        bool     `json:"active"`
			BranchFilter  string   `json:"branch_filter"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validHookTarget(input.URL) || !validHookEvents(input.Events) || len(input.Secret) > 4096 || len(input.Authorization) > 4096 || strings.ContainsAny(input.Authorization, "\r\n\x00") || len(input.BranchFilter) > 1024 {
			jsonError(w, 422, "invalid_hook", "Provide an HTTP(S) target without URL credentials/fragment, supported events and bounded secret/header/filter inputs.")
			return
		}
		result, err := s.Forgejo.CreateHook(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), forgejo.HookInput{URL: input.URL, Secret: input.Secret, Authorization: input.Authorization, Events: input.Events, Active: input.Active, BranchFilter: input.BranchFilter})
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, hookDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	hooks, metadata, err := s.Forgejo.Hooks(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]hookView, 0, len(hooks))
	for _, hook := range hooks {
		items = append(items, hookDTO(hook))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiHook(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("hook"))
	if !ok {
		return
	}
	var input struct {
		URL           *string   `json:"url"`
		Authorization *string   `json:"authorization"`
		Events        *[]string `json:"events"`
		Active        *bool     `json:"active"`
		BranchFilter  *string   `json:"branch_filter"`
	}
	if r.Method == "PATCH" {
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if (input.URL != nil && !validHookTarget(*input.URL)) || (input.Events != nil && !validHookEvents(*input.Events)) || (input.Authorization != nil && (len(*input.Authorization) > 4096 || strings.ContainsAny(*input.Authorization, "\r\n\x00"))) || (input.BranchFilter != nil && len(*input.BranchFilter) > 1024) {
			jsonError(w, 422, "invalid_hook", "Invalid bounded hook fields.")
			return
		}
	}
	current, err := s.Forgejo.Hook(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id)
	if err != nil {
		providerError(w, err)
		return
	}
	if r.Method == "GET" {
		jsonResponse(w, 200, hookDTO(current))
		return
	}
	// The selected PATCH unconditionally replaces these native fields. Retain
	// omitted values only in this request, never by exposing or persisting secrets.
	update := forgejo.HookInput{URL: current.URL, Authorization: current.Authorization, Events: current.Events, Active: current.Active, BranchFilter: current.BranchFilter}
	if input.URL != nil {
		update.URL = *input.URL
	}
	if input.Authorization != nil {
		update.Authorization = *input.Authorization
	}
	if input.Active != nil {
		update.Active = *input.Active
	}
	if input.BranchFilter != nil {
		update.BranchFilter = *input.BranchFilter
	}
	if input.Events != nil {
		for _, event := range []string{"package", "action_run_failure", "action_run_recover", "action_run_success"} {
			if slices.Contains(*input.Events, event) != slices.Contains(current.Events, event) {
				jsonError(w, 422, "native_hook_edit_gap", "This Forgejo API cannot change package/action event flags. Use native hook settings for those flags.")
				return
			}
		}
		update.Events = *input.Events
	}
	// Empty events are replaced with push by upstream PATCH. Do not silently add
	// a delivery subscription to an existing eventless hook.
	if len(update.Events) == 0 {
		jsonError(w, 422, "native_hook_edit_gap", "Native PATCH defaults empty events to push. Select explicit events or use native hook settings.")
		return
	}
	result, err := s.Forgejo.EditHook(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, update)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, hookDTO(result))
}
