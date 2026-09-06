package web

import (
	"encoding/base64"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) forgejoRoutes() {
	s.mux.HandleFunc("/api/forgejo/me", s.apiProvider(s.apiForgejoMe, "read:user", "GET"))
	s.mux.HandleFunc("/api/forgejo/me/settings", s.apiProvider(s.apiForgejoSettings, "write:user", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/me/git-keys", s.apiProvider(s.apiGitKeys, "write:user", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/admin/users", s.apiProvider(s.apiPeople, "write:admin", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repositories", s.apiProvider(s.apiRepositories, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}", s.apiProvider(s.apiRepository, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/contents", s.apiProvider(s.apiContents, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/download", s.apiProvider(s.apiDownload, "read:repository", "GET"))
	s.historyRoutes()
	s.issueRoutes()
	s.pullRoutes()
	s.hookRoutes()
	s.repositorySettingsRoutes()
	s.protectionRoutes()
	s.organizationRoutes()
	s.workRoutes()
}

type providerUserView struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"full_name"`
	Admin bool   `json:"is_admin"`
}

func providerUser(u forgejo.User) providerUserView {
	return providerUserView{strconv.FormatInt(u.ID, 10), u.Login, u.Name, u.Admin}
}
func (s *Server) apiForgejoMe(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	user, err := s.Forgejo.Current(r.Context(), token)
	if err != nil {
		providerError(w, err)
		return
	}
	if user.ID != v.User.ID {
		jsonError(w, 401, "provider_identity_mismatch", "Sign in again.")
		return
	}
	jsonResponse(w, 200, providerUser(user))
}
func (s *Server) apiForgejoSettings(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	var result forgejo.UserSettings
	var err error
	if r.Method == "GET" {
		result, err = s.Forgejo.Settings(r.Context(), token)
	} else {
		if !decodeAPIObject(w, r, &result) {
			return
		}
		if len(result.Name) > 200 || len(result.Description) > 4096 || len(result.Location) > 255 || len(result.Pronouns) > 255 {
			jsonError(w, 400, "invalid_fields", "Profile fields exceed supported lengths.")
			return
		}
		result, err = s.Forgejo.UpdateSettings(r.Context(), token, result)
	}
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, result)
}

func apiPage(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := r.URL.Query().Get("page")
	if value == "" {
		return 1, true
	}
	page, err := strconv.Atoi(value)
	if err != nil || page < 1 || page > 1000000 {
		jsonError(w, 400, "invalid_page", "Page must be between 1 and 1000000.")
		return 0, false
	}
	return page, true
}

type pageView[T any] struct {
	Items []T `json:"items"`
	forgejo.Pagination
}

func pageResult[T any](items []T, metadata forgejo.Pagination) pageView[T] {
	if items == nil {
		items = []T{}
	}
	return pageView[T]{Items: items, Pagination: metadata}
}
func (s *Server) apiGitKeys(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if r.Method == "POST" {
		var input struct {
			Title string `json:"title"`
			Key   string `json:"key"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Title) == "" || len(input.Title) > 255 {
			jsonError(w, 400, "invalid_title", "Provide a key title of at most 255 bytes.")
			return
		}
		key, _, err := normalizeDevelopmentKey(input.Key)
		if err != nil {
			jsonError(w, 400, "invalid_public_key", "Provide one public SSH key without options, never a private key.")
			return
		}
		result, err := s.Forgejo.AddGitKey(r.Context(), token, input.Title, key)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, gitKeyView{strconv.FormatInt(result.ID, 10), result.Title, result.Key, result.Fingerprint})
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	keys, metadata, err := s.Forgejo.GitKeys(r.Context(), token, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]gitKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, gitKeyView{strconv.FormatInt(key.ID, 10), key.Title, key.Key, key.Fingerprint})
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}

type gitKeyView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
}

func (s *Server) apiPeople(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	// Native /admin/users enforces current administrator authority on every call.
	// Soda operator status is neither required nor sufficient.
	if r.Method == "POST" {
		var input struct {
			Login    string `json:"login"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if input.Login == "" || len(input.Login) > 255 || input.Email == "" || len(input.Email) > 320 || input.Password == "" || len(input.Password) > 4096 {
			jsonError(w, 400, "invalid_fields", "Provide a username, email and initial password within supported lengths.")
			return
		}
		user, err := s.Forgejo.CreateUser(r.Context(), token, input.Login, input.Email, input.Password)
		input.Password = ""
		if err != nil {
			providerError(w, err)
			return
		}
		// Do not manufacture a local identity/session: native onboarding and OAuth
		// create the user's extension record when they actually sign in.
		jsonResponse(w, 201, providerUser(user))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	users, metadata, err := s.Forgejo.People(r.Context(), token, page)
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

type repositoryView struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	FullName      string           `json:"full_name"`
	Owner         providerUserView `json:"owner"`
	Description   string           `json:"description"`
	Private       bool             `json:"private"`
	DefaultBranch string           `json:"default_branch"`
	CloneURL      string           `json:"clone_url"`
	SSHURL        string           `json:"ssh_url"`
	Empty         bool             `json:"empty"`
}

func repositoryDTO(repo forgejo.Repository) repositoryView {
	return repositoryView{strconv.FormatInt(repo.ID, 10), repo.Name, repo.FullName, providerUser(repo.Owner), repo.Description, repo.Private, repo.DefaultBranch, repo.CloneURL, repo.SSHURL, repo.Empty}
}
func (s *Server) apiRepositories(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if r.Method == "POST" {
		var input forgejo.CreateRepository
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Name) == "" || len(input.Name) > 100 || len(input.Description) > 2048 {
			jsonError(w, 400, "invalid_repository", "Provide a repository name and bounded description.")
			return
		}
		repo, err := s.Forgejo.CreateRepository(r.Context(), token, input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, repositoryDTO(repo))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	repos, metadata, err := s.Forgejo.MyRepositories(r.Context(), token, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]repositoryView, 0, len(repos))
	for _, repo := range repos {
		items = append(items, repositoryDTO(repo))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiRepository(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	repo, err := s.Forgejo.Repository(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"))
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, repositoryDTO(repo))
}
func repositoryContentQuery(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if !validRepositoryRoute(w, r) {
		return "", "", false
	}
	ref, file := r.URL.Query().Get("ref"), r.URL.Query().Get("path")
	if len(ref) > 1024 || len(file) > 4096 || strings.ContainsAny(ref+file, "\x00\r\n\\") || strings.HasPrefix(file, "/") {
		jsonError(w, 400, "invalid_path", "Invalid repository ref or relative path.")
		return "", "", false
	}
	for _, part := range strings.Split(file, "/") {
		if part == "." || part == ".." || (file != "" && part == "") {
			jsonError(w, 400, "invalid_path", "Empty and dot path segments are not supported.")
			return "", "", false
		}
	}
	return ref, file, true
}
func (s *Server) apiDownload(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ref, file, ok := repositoryContentQuery(w, r)
	if !ok {
		return
	}
	if file == "" {
		jsonError(w, 400, "invalid_path", "Select a file to download.")
		return
	}
	body, err := s.Forgejo.RawFile(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), ref, file)
	if err != nil {
		providerError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(file)}))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
func (s *Server) apiContents(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ref, file, ok := repositoryContentQuery(w, r)
	if !ok {
		return
	}
	contents, err := s.Forgejo.Contents(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), ref, file)
	if err != nil {
		providerError(w, err)
		return
	}
	type contentView struct {
		Name        string  `json:"name"`
		Path        string  `json:"path"`
		Type        string  `json:"type"`
		SHA         string  `json:"sha"`
		Size        string  `json:"size"`
		Text        *string `json:"text"`
		Unavailable bool    `json:"unavailable"`
	}
	items := make([]contentView, 0, len(contents))
	for _, item := range contents {
		view := contentView{Name: item.Name, Path: item.Path, Type: item.Type, SHA: item.SHA, Size: strconv.FormatInt(item.Size, 10)}
		if item.Type == "file" {
			view.Unavailable = true
			if item.Size >= 0 && item.Size <= 256*1024 && item.Encoding == "base64" {
				text, err := base64.StdEncoding.DecodeString(item.Content)
				if err == nil && len(text) <= 256*1024 && utf8.Valid(text) && !strings.ContainsRune(string(text), 0) {
					value := string(text)
					view.Text = &value
					view.Unavailable = false
				}
			}
		}
		items = append(items, view)
	}
	jsonResponse(w, 200, struct {
		Items []contentView `json:"items"`
	}{items})
}
