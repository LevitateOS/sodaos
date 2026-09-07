package web

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

var commitHash = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

func (s *Server) historyRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/commits", s.apiProvider(s.apiCommits, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/commits/{sha}", s.apiProvider(s.apiCommit, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/commits/{sha}/diff", s.apiProvider(s.apiCommitDiff, "read:repository", "GET"))
	s.mux.HandleFunc("GET /api/forgejo/repos/{owner}/{repo}/branches", s.apiProvider(s.apiBranches, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/branches", s.apiProvider(s.apiBranches, "write:repository", "POST"))
	s.mux.HandleFunc("GET /api/forgejo/repos/{owner}/{repo}/tags", s.apiProvider(s.apiTags, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/tags", s.apiProvider(s.apiTags, "write:repository", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/compare", s.apiProvider(s.apiCompare, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/files", s.apiProvider(s.apiWriteFile, "write:repository", "POST", "PUT"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/fork", s.apiProvider(s.apiFork, "write:repository", "POST"))
	s.mux.HandleFunc("/api/forgejo/repositories/import", s.apiProvider(s.apiImportRepository, "write:repository", "POST"))
}
func validRepositoryRoute(w http.ResponseWriter, r *http.Request) bool {
	if !validRepositoryPart(r.PathValue("owner")) || !validRepositoryPart(r.PathValue("repo")) {
		jsonError(w, 400, "invalid_repository", "Invalid repository identity.")
		return false
	}
	return true
}
func validRef(ref string) bool {
	return ref != "" && len(ref) <= 1024 && !strings.ContainsAny(ref, "\x00\r\n\\")
}
func (s *Server) apiCommits(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ref, file, ok := repositoryContentQuery(w, r)
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := s.Forgejo.Commits(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), ref, file, page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiCommit(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if !commitHash.MatchString(r.PathValue("sha")) {
		jsonError(w, 400, "invalid_commit", "Select a full commit SHA.")
		return
	}
	result, err := s.Forgejo.Commit(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), r.PathValue("sha"))
	if err != nil {
		providerError(w, err)
		return
	}
	if !strings.EqualFold(result.SHA, r.PathValue("sha")) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 200, result)
}
func (s *Server) apiCommitDiff(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if !commitHash.MatchString(r.PathValue("sha")) {
		jsonError(w, 400, "invalid_commit", "Select a full commit SHA.")
		return
	}
	diff, err := s.Forgejo.CommitDiff(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), r.PathValue("sha"))
	if err != nil {
		providerError(w, err)
		return
	}
	if !utf8.Valid(diff) {
		jsonError(w, 422, "unsupported_diff", "Diff encoding is not supported. Use native Git.")
		return
	}
	jsonResponse(w, 200, struct {
		Text string `json:"text"`
	}{string(diff)})
}
func (s *Server) apiBranches(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Name string `json:"name"`
			Ref  string `json:"ref"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validRef(input.Name) || !validRef(input.Ref) {
			jsonError(w, 400, "invalid_ref", "Provide branch and source ref.")
			return
		}
		result, err := s.Forgejo.CreateBranch(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input.Name, input.Ref)
		if err != nil {
			providerError(w, err)
			return
		}
		if !validRef(result.Name) {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
		jsonResponse(w, 201, result)
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := s.Forgejo.Branches(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiTags(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Name    string `json:"name"`
			Target  string `json:"target"`
			Message string `json:"message"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validRef(input.Name) || !validRef(input.Target) || len(input.Message) > 8192 {
			jsonError(w, 400, "invalid_tag", "Provide a tag name, target and bounded message.")
			return
		}
		result, err := s.Forgejo.CreateTag(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input.Name, input.Target, input.Message)
		if err != nil {
			providerError(w, err)
			return
		}
		if !validRef(result.Name) || !commitHash.MatchString(result.Commit.SHA) {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
		jsonResponse(w, 201, result)
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := s.Forgejo.Tags(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiCompare(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	base, head := r.URL.Query().Get("base"), r.URL.Query().Get("head")
	if !validRef(base) || !validRef(head) {
		jsonError(w, 400, "invalid_ref", "Provide base and head refs.")
		return
	}
	owner, repo := r.PathValue("owner"), r.PathValue("repo")
	baseSHA, err := s.Forgejo.ResolveRef(r.Context(), token, owner, repo, base)
	if err != nil {
		providerError(w, err)
		return
	}
	if !commitHash.MatchString(baseSHA) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	headSHA := baseSHA
	if head != base {
		headSHA, err = s.Forgejo.ResolveRef(r.Context(), token, owner, repo, head)
		if err != nil {
			providerError(w, err)
			return
		}
	}
	if !commitHash.MatchString(headSHA) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	result, err := s.Forgejo.Compare(r.Context(), token, owner, repo, baseSHA, headSHA)
	if err != nil {
		providerError(w, err)
		return
	}
	// The selected native endpoint returns the whole commit list (no paging),
	// and concatenates each commit's files. This is NOT a net changed-file diff.
	if result.Commits == nil || result.Files == nil || result.Total != int64(len(result.Commits)) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	for _, commit := range result.Commits {
		if !commitHash.MatchString(commit.SHA) {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
	}
	jsonResponse(w, 200, struct {
		Commits []forgejo.Commit     `json:"commits"`
		Files   []forgejo.CommitFile `json:"files"`
		Total   string               `json:"total_commits"`
		BaseSHA string               `json:"base_sha"`
		HeadSHA string               `json:"head_sha"`
	}{result.Commits, result.Files, strconv.FormatInt(result.Total, 10), baseSHA, headSHA})
}
func (s *Server) apiWriteFile(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	ref, file, ok := repositoryContentQuery(w, r)
	if !ok {
		return
	}
	if !validRef(ref) || file == "" {
		jsonError(w, 400, "invalid_path", "Provide an explicit branch and file path.")
		return
	}
	var input struct {
		Content string `json:"content"`
		Message string `json:"message"`
		SHA     string `json:"sha"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	data, err := base64.StdEncoding.Strict().DecodeString(input.Content)
	if err != nil || len(data) > 32768 || strings.TrimSpace(input.Message) == "" || len(input.Message) > 8192 {
		jsonError(w, 422, "invalid_file", "Provide base64 content up to 32 KiB and a commit message up to 8 KiB.")
		return
	}
	update := r.Method == "PUT"
	if (update && !commitHash.MatchString(input.SHA)) || (!update && input.SHA != "") {
		jsonError(w, 422, "file_precondition_required", "Updates require the exact loaded file SHA; creation must not supply one.")
		return
	}
	result, err := s.Forgejo.WriteFile(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), file, forgejo.WriteFile{Branch: ref, Content: input.Content, Message: input.Message, SHA: input.SHA}, update)
	if err != nil {
		providerError(w, err)
		return
	}
	if !commitHash.MatchString(result.Commit.SHA) || !commitHash.MatchString(result.Content.SHA) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 200, struct {
		CommitSHA  string `json:"commit_sha"`
		ContentSHA string `json:"content_sha"`
	}{result.Commit.SHA, result.Content.SHA})
}
func (s *Server) apiFork(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if len(input.Name) > 100 {
		jsonError(w, 422, "invalid_name", "Repository name exceeds 100 bytes.")
		return
	}
	result, err := s.Forgejo.Fork(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input.Name)
	if err != nil {
		providerError(w, err)
		return
	}
	if result.ID <= 0 || result.Owner.ID != v.User.ID || !validRepositoryPart(result.Owner.Login) || !validRepositoryPart(result.Name) || !result.Fork || result.Mirror {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 201, repositoryDTO(result))
}
func (s *Server) apiImportRepository(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	var input struct {
		URL      string `json:"url"`
		Name     string `json:"name"`
		Private  bool   `json:"private"`
		Username string `json:"username"`
		Password string `json:"password"`
		Token    string `json:"token"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	target, err := url.Parse(input.URL)
	if err != nil || target.Scheme != "https" || target.Host == "" || target.User != nil || target.RawQuery != "" || target.Fragment != "" || len(input.URL) > 4096 || input.Name == "" || len(input.Name) > 100 || len(input.Username) > 255 || len(input.Password) > 4096 || len(input.Token) > 4096 {
		jsonError(w, 422, "invalid_import", "Provide a credential-free HTTPS Git URL, repository name and optional separate credentials. Forgejo applies its native import/network policy.")
		return
	}
	result, err := s.Forgejo.ImportRepository(r.Context(), token, forgejo.ImportRepository{CloneURL: input.URL, Name: input.Name, Owner: v.User.Login, Private: input.Private, Service: "git", Username: input.Username, Password: input.Password, Token: input.Token})
	input.Password = ""
	input.Token = ""
	if err != nil {
		providerError(w, err)
		return
	}
	if result.ID <= 0 || result.Owner.ID != v.User.ID || !validRepositoryPart(result.Owner.Login) || !validRepositoryPart(result.Name) || result.Mirror {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 201, repositoryDTO(result))
}
