package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) repositorySettingsRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/settings", s.apiProvider(s.apiRepositorySettings, "write:repository", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/collaborators", s.apiProvider(s.apiCollaborators, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/collaborators/{login}", s.apiProvider(s.apiCollaborator, "write:repository", "GET", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/deploy-keys", s.apiProvider(s.apiDeployKeys, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/deploy-keys/{key}", s.apiProvider(s.apiDeleteDeployKey, "write:repository", "DELETE"))
}

type repositorySettingsView struct {
	repositoryView
	Website             string `json:"website"`
	HasIssues           bool   `json:"has_issues"`
	HasPulls            bool   `json:"has_pull_requests"`
	HasWiki             bool   `json:"has_wiki"`
	HasActions          bool   `json:"has_actions"`
	HasReleases         bool   `json:"has_releases"`
	HasPackages         bool   `json:"has_packages"`
	AllowMerge          bool   `json:"allow_merge_commits"`
	AllowSquash         bool   `json:"allow_squash_merge"`
	AllowRebase         bool   `json:"allow_rebase"`
	AllowRebaseExplicit bool   `json:"allow_rebase_explicit"`
	AllowFastForward    bool   `json:"allow_fast_forward_only_merge"`
}

func repositorySettingsDTO(repo forgejo.RepositorySettings) repositorySettingsView {
	return repositorySettingsView{repositoryDTO(repo.Repository), repo.Website, repo.HasIssues, repo.HasPulls, repo.HasWiki, repo.HasActions, repo.HasReleases, repo.HasPackages, repo.AllowMerge, repo.AllowSquash, repo.AllowRebase, repo.AllowRebaseExplicit, repo.AllowFastForward}
}
func (s *Server) apiRepositorySettings(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "PATCH" {
		var input forgejo.EditRepositorySettings
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if (input.Description != nil && len(*input.Description) > 4096) || (input.DefaultBranch != nil && !validRef(*input.DefaultBranch)) {
			jsonError(w, 422, "invalid_repository_settings", "Provide bounded description and native default branch.")
			return
		}
		if input.Website != nil && *input.Website != "" {
			u, err := url.Parse(*input.Website)
			if err != nil || len(*input.Website) > 2048 || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
				jsonError(w, 422, "invalid_repository_website", "Use an HTTP(S) website without embedded credentials.")
				return
			}
		}
		result, err := s.Forgejo.EditRepositorySettings(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 200, repositorySettingsDTO(result))
		return
	}
	result, err := s.Forgejo.RepositorySettings(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"))
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, repositorySettingsDTO(result))
}
func (s *Server) apiCollaborators(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	users, metadata, err := s.Forgejo.Collaborators(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]providerUserView, 0, len(users))
	for _, user := range users {
		items = append(items, providerUser(user))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiCollaborator(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	login := r.PathValue("login")
	if !validRepositoryPart(login) {
		jsonError(w, 400, "invalid_user", "Select a native username.")
		return
	}
	if r.Method != "GET" {
		var input struct {
			Permission string `json:"permission"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if r.Method == "PUT" && input.Permission != "read" && input.Permission != "write" && input.Permission != "admin" {
			jsonError(w, 422, "invalid_native_permission", "Choose read, write or admin repository permission.")
			return
		}
		if err := s.Forgejo.SetCollaborator(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), login, input.Permission, r.Method == "DELETE"); err != nil {
			providerError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	result, err := s.Forgejo.CollaboratorPermission(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), login)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, struct {
		Permission string           `json:"permission"`
		Role       string           `json:"role_name"`
		User       providerUserView `json:"user"`
	}{result.Permission, result.Role, providerUser(result.User)})
}

type deployKeyView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	ReadOnly    bool   `json:"read_only"`
}

func deployKeyDTO(key forgejo.DeployKey) deployKeyView {
	return deployKeyView{strconv.FormatInt(key.ID, 10), key.Title, key.Key, key.Fingerprint, key.ReadOnly}
}
func (s *Server) apiDeployKeys(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Title    string `json:"title"`
			Key      string `json:"key"`
			ReadOnly *bool  `json:"read_only"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		key, _, err := normalizeDevelopmentKey(input.Key)
		if err != nil || strings.TrimSpace(input.Title) == "" || len(input.Title) > 255 || input.ReadOnly == nil {
			jsonError(w, 422, "invalid_deploy_key", "Provide a title, public SSH key and explicit read-only choice. Never send a private key.")
			return
		}
		result, err := s.Forgejo.CreateDeployKey(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input.Title, key, *input.ReadOnly)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, deployKeyDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	keys, metadata, err := s.Forgejo.DeployKeys(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]deployKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, deployKeyDTO(key))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiDeleteDeployKey(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("key"))
	if !ok {
		return
	}
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	if err := s.Forgejo.DeleteDeployKey(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
