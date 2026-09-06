package web

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) pullRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls", s.apiProvider(s.apiPulls, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}", s.apiProvider(s.apiPull, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/commits", s.apiProvider(s.apiPullCommits, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/files", s.apiProvider(s.apiPullFiles, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/diff", s.apiProvider(s.apiPullDiff, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/reviews", s.apiProvider(s.apiPullReviews, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/reviewers", s.apiProvider(s.apiPullReviewers, "write:repository", "POST", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/pulls/{index}/merge", s.apiProvider(s.apiMergePull, "write:repository", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/commit-status/{sha}", s.apiProvider(s.apiCommitStatus, "read:repository", "GET"))
}

type pullBranchView struct {
	Ref   string `json:"ref"`
	SHA   string `json:"sha"`
	Label string `json:"label"`
}
type pullView struct {
	issueView
	Head         pullBranchView     `json:"head"`
	Base         pullBranchView     `json:"base"`
	MergeBase    string             `json:"merge_base"`
	Merged       bool               `json:"merged"`
	Mergeable    bool               `json:"mergeable"`
	Draft        bool               `json:"draft"`
	Reviewers    []providerUserView `json:"requested_reviewers"`
	MergeMethods []string           `json:"merge_methods"`
}

func pullDTO(p forgejo.Pull) pullView {
	methods := []string{}
	repo := p.Base.Repo
	if repo.AllowMerge {
		methods = append(methods, "merge")
	}
	if repo.AllowSquash {
		methods = append(methods, "squash")
	}
	if repo.AllowRebase {
		methods = append(methods, "rebase")
	}
	if repo.AllowRebaseExplicit {
		methods = append(methods, "rebase-merge")
	}
	if repo.AllowFastForward {
		methods = append(methods, "fast-forward-only")
	}
	result := pullView{issueView: issueDTO(p.Issue), Head: pullBranchView{p.Head.Ref, p.Head.SHA, p.Head.Label}, Base: pullBranchView{p.Base.Ref, p.Base.SHA, p.Base.Label}, MergeBase: p.MergeBase, Merged: p.Merged, Mergeable: p.Mergeable, Draft: p.Draft, MergeMethods: methods, Reviewers: []providerUserView{}}
	for _, u := range p.RequestedReviewers {
		result.Reviewers = append(result.Reviewers, providerUser(u))
	}
	return result
}
func (s *Server) apiPulls(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input forgejo.CreatePull
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Title) == "" || len(input.Title) > 255 || len(input.Body) > 32768 || !validRef(input.Base) || !validRef(input.Head) {
			jsonError(w, 422, "invalid_pull", "Provide title, bounded body, base and head refs.")
			return
		}
		result, err := s.Forgejo.CreatePull(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, pullDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "open"
	}
	if state != "open" && state != "closed" && state != "all" {
		jsonError(w, 400, "invalid_state", "Choose open, closed or all.")
		return
	}
	pulls, metadata, err := s.Forgejo.Pulls(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), state, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]pullView, 0, len(pulls))
	for _, p := range pulls {
		items = append(items, pullDTO(p))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiPull(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	result, err := s.Forgejo.Pull(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pullDTO(result))
}

// The reviewed head is explicit even if upstream advances after this preflight.
// Native reviews receive commit_id and merge receives head_commit_id; neither
// operation is allowed to choose an implicit latest revision or force a merge.
func (s *Server) checkedPull(w http.ResponseWriter, r *http.Request, token string, index int64, head, base, mergeBase string) (forgejo.Pull, bool) {
	if !commitHash.MatchString(head) || !commitHash.MatchString(base) || !commitHash.MatchString(mergeBase) {
		jsonError(w, 422, "pull_snapshot_required", "Provide the full displayed head, base and merge-base SHAs.")
		return forgejo.Pull{}, false
	}
	p, err := s.Forgejo.Pull(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return p, false
	}
	if p.Head.SHA != head || p.Base.SHA != base || p.MergeBase != mergeBase {
		jsonError(w, 409, "pull_changed", "Pull request revisions changed. Reload before reviewing or merging.")
		return p, false
	}
	return p, true
}
func (s *Server) apiPullCommits(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := s.Forgejo.PullCommits(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiPullFiles(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	if _, ok = s.checkedPull(w, r, token, index, q.Get("head"), q.Get("base"), q.Get("merge_base")); !ok {
		return
	}
	files, metadata, err := s.Forgejo.PullFiles(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, page)
	if err != nil {
		providerError(w, err)
		return
	}
	if _, ok = s.checkedPull(w, r, token, index, q.Get("head"), q.Get("base"), q.Get("merge_base")); !ok {
		return
	}
	type changedFileView struct {
		Name         string `json:"filename"`
		PreviousName string `json:"previous_filename"`
		Status       string `json:"status"`
		Additions    string `json:"additions"`
		Deletions    string `json:"deletions"`
	}
	items := make([]changedFileView, 0, len(files))
	for _, f := range files {
		items = append(items, changedFileView{f.Name, f.PreviousName, f.Status, strconv.FormatInt(f.Additions, 10), strconv.FormatInt(f.Deletions, 10)})
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiPullDiff(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	q := r.URL.Query()
	if _, ok = s.checkedPull(w, r, token, index, q.Get("head"), q.Get("base"), q.Get("merge_base")); !ok {
		return
	}
	diff, err := s.Forgejo.PullDiff(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return
	}
	if _, ok = s.checkedPull(w, r, token, index, q.Get("head"), q.Get("base"), q.Get("merge_base")); !ok {
		return
	}
	if !utf8.Valid(diff) {
		jsonError(w, 422, "unsupported_diff", "Use native Git for this diff encoding.")
		return
	}
	jsonResponse(w, 200, struct {
		Text      string `json:"text"`
		Head      string `json:"head"`
		Base      string `json:"base"`
		MergeBase string `json:"merge_base"`
	}{string(diff), q.Get("head"), q.Get("base"), q.Get("merge_base")})
}

type reviewView struct {
	ID        string           `json:"id"`
	User      providerUserView `json:"user"`
	Body      string           `json:"body"`
	Commit    string           `json:"commit_id"`
	State     string           `json:"state"`
	Stale     bool             `json:"stale"`
	Dismissed bool             `json:"dismissed"`
	Submitted string           `json:"submitted_at"`
}

func reviewDTO(review forgejo.Review) reviewView {
	return reviewView{strconv.FormatInt(review.ID, 10), providerUser(review.User), review.Body, review.Commit, review.State, review.Stale, review.Dismissed, review.Submitted}
}
func (s *Server) apiPullReviews(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Head      string `json:"head"`
			Base      string `json:"base"`
			MergeBase string `json:"merge_base"`
			Event     string `json:"event"`
			Body      string `json:"body"`
			Comments  []struct {
				Path string `json:"path"`
				Line int64  `json:"line"`
				Body string `json:"body"`
			} `json:"comments"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if len(input.Body) > 32768 || len(input.Comments) > 20 || (input.Event != "APPROVED" && input.Event != "REQUEST_CHANGES" && input.Event != "COMMENT") {
			jsonError(w, 422, "invalid_review", "Choose a supported review event and bounded content.")
			return
		}
		comments := make([]forgejo.ReviewComment, 0, len(input.Comments))
		for _, comment := range input.Comments {
			if comment.Path == "" || strings.HasPrefix(comment.Path, "/") || strings.ContainsAny(comment.Path, "\x00\r\n\\") || len(comment.Path) > 4096 || comment.Line <= 0 || comment.Line > 2147483647 || strings.TrimSpace(comment.Body) == "" || len(comment.Body) > 8192 {
				jsonError(w, 422, "invalid_review_comment", "Provide a relative file path, positive new-side line number and bounded comment.")
				return
			}
			for _, part := range strings.Split(comment.Path, "/") {
				if part == "" || part == "." || part == ".." {
					jsonError(w, 422, "invalid_review_comment", "Invalid review file path.")
					return
				}
			}
			comments = append(comments, forgejo.ReviewComment{Path: comment.Path, NewLine: comment.Line, Body: comment.Body})
		}
		if _, ok = s.checkedPull(w, r, token, index, input.Head, input.Base, input.MergeBase); !ok {
			return
		}
		result, err := s.Forgejo.CreatePullReview(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, forgejo.CreateReview{Body: input.Body, Commit: input.Head, Event: input.Event, Comments: comments})
		if err != nil {
			providerError(w, err)
			return
		}
		if result.Commit != input.Head {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
		jsonResponse(w, 201, reviewDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	reviews, metadata, err := s.Forgejo.PullReviews(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]reviewView, 0, len(reviews))
	for _, review := range reviews {
		items = append(items, reviewDTO(review))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiPullReviewers(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	var input struct {
		Reviewers []string `json:"reviewers"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if len(input.Reviewers) == 0 || !validAssignees(input.Reviewers) {
		jsonError(w, 422, "invalid_reviewers", "Provide bounded native reviewer usernames.")
		return
	}
	if err := s.Forgejo.RequestPullReviewers(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, input.Reviewers, r.Method == "DELETE"); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) apiMergePull(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	var input struct {
		Head      string `json:"head"`
		Base      string `json:"base"`
		MergeBase string `json:"merge_base"`
		Strategy  string `json:"strategy"`
		Title     string `json:"title"`
		Message   string `json:"message"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	switch input.Strategy {
	case "merge", "squash", "rebase", "rebase-merge", "fast-forward-only":
	default:
		jsonError(w, 422, "invalid_merge_strategy", "Choose a supported native merge strategy.")
		return
	}
	if len(input.Title) > 255 || len(input.Message) > 8192 {
		jsonError(w, 422, "invalid_merge_message", "Merge message exceeds supported size.")
		return
	}
	if _, ok = s.checkedPull(w, r, token, index, input.Head, input.Base, input.MergeBase); !ok {
		return
	}
	if err := s.Forgejo.MergePull(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, input.Head, input.Strategy, input.Title, input.Message); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) apiCommitStatus(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	sha := r.PathValue("sha")
	if !commitHash.MatchString(sha) {
		jsonError(w, 400, "invalid_commit", "Select a full commit SHA.")
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	result, metadata, err := s.Forgejo.Status(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), sha, page)
	if err != nil {
		providerError(w, err)
		return
	}
	type statusView struct {
		ID          string `json:"id"`
		Context     string `json:"context"`
		Description string `json:"description"`
		State       string `json:"state"`
	}
	items := make([]statusView, 0, len(result.Statuses))
	for _, status := range result.Statuses {
		items = append(items, statusView{strconv.FormatInt(status.ID, 10), status.Context, status.Description, status.State})
	}
	jsonResponse(w, 200, struct {
		pageView[statusView]
		SHA        string `json:"sha"`
		State      string `json:"state"`
		TotalCount string `json:"total_count"`
	}{pageResult(items, metadata), result.SHA, result.State, strconv.FormatInt(result.Total, 10)})
}
