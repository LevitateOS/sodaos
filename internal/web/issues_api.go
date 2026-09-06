package web

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) issueRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues", s.apiProvider(s.apiIssues, "write:issue", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}", s.apiProvider(s.apiIssue, "write:issue", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}/comments", s.apiProvider(s.apiIssueComments, "write:issue", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issue-comments/{comment}", s.apiProvider(s.apiEditComment, "write:issue", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}/labels", s.apiProvider(s.apiIssueLabels, "write:issue", "PUT"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/labels", s.apiProvider(s.apiLabels, "write:issue", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/labels/{label}", s.apiProvider(s.apiEditLabel, "write:issue", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/milestones", s.apiProvider(s.apiMilestones, "write:issue", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/milestones/{milestone}", s.apiProvider(s.apiEditMilestone, "write:issue", "PATCH"))
	s.issueActivityRoutes()
}
func apiID(w http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		jsonError(w, 400, "invalid_id", "Provide a positive decimal native identifier.")
		return 0, false
	}
	return id, true
}

type labelView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func labelDTO(label forgejo.Label) labelView {
	return labelView{strconv.FormatInt(label.ID, 10), label.Name, label.Color, label.Description}
}

type milestoneView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	DueOn       string `json:"due_on"`
}

func milestoneDTO(m forgejo.Milestone) milestoneView {
	return milestoneView{strconv.FormatInt(m.ID, 10), m.Title, m.Description, m.State, m.DueOn}
}

type issueView struct {
	ID        string             `json:"id"`
	Number    string             `json:"number"`
	Title     string             `json:"title"`
	Body      string             `json:"body"`
	State     string             `json:"state"`
	User      providerUserView   `json:"user"`
	Assignees []providerUserView `json:"assignees"`
	Labels    []labelView        `json:"labels"`
	Milestone *milestoneView     `json:"milestone"`
	Created   string             `json:"created_at"`
	Updated   string             `json:"updated_at"`
}

func issueDTO(issue forgejo.Issue) issueView {
	result := issueView{ID: strconv.FormatInt(issue.ID, 10), Number: strconv.FormatInt(issue.Number, 10), Title: issue.Title, Body: issue.Body, State: issue.State, User: providerUser(issue.User), Assignees: []providerUserView{}, Labels: []labelView{}, Created: issue.Created, Updated: issue.Updated}
	for _, u := range issue.Assignees {
		result.Assignees = append(result.Assignees, providerUser(u))
	}
	for _, label := range issue.Labels {
		result.Labels = append(result.Labels, labelDTO(label))
	}
	if issue.Milestone != nil {
		m := milestoneDTO(*issue.Milestone)
		result.Milestone = &m
	}
	return result
}
func validAssignees(values []string) bool {
	if len(values) > 32 {
		return false
	}
	for _, value := range values {
		if value == "" || len(value) > 255 {
			return false
		}
	}
	return true
}
func (s *Server) apiIssues(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Title     string   `json:"title"`
			Body      string   `json:"body"`
			Assignees []string `json:"assignees"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Title) == "" || len(input.Title) > 255 || len(input.Body) > 32768 || !validAssignees(input.Assignees) {
			jsonError(w, 422, "invalid_issue", "Provide a title up to 255 bytes, body up to 32 KiB and bounded native assignee usernames.")
			return
		}
		issue, err := s.Forgejo.CreateIssue(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), forgejo.CreateIssue{Title: input.Title, Body: input.Body, Assignees: input.Assignees})
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, issueDTO(issue))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	state := q.Get("state")
	if state == "" {
		state = "open"
	}
	if state != "open" && state != "closed" && state != "all" {
		jsonError(w, 400, "invalid_state", "Choose open, closed or all.")
		return
	}
	for _, field := range []string{"q", "labels", "milestones", "assigned_by"} {
		if len(q.Get(field)) > 1024 {
			jsonError(w, 400, "invalid_filter", "Issue filter exceeds supported size.")
			return
		}
	}
	issues, metadata, err := s.Forgejo.Issues(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), forgejo.IssueQuery{Page: page, State: state, Search: q.Get("q"), Labels: q.Get("labels"), Milestones: q.Get("milestones"), Assignee: q.Get("assigned_by")})
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]issueView, 0, len(issues))
	for _, issue := range issues {
		items = append(items, issueDTO(issue))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiIssue(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	var result forgejo.Issue
	var err error
	if r.Method == "GET" {
		result, err = s.Forgejo.Issue(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	} else {
		var input struct {
			Title     *string   `json:"title"`
			Body      *string   `json:"body"`
			State     *string   `json:"state"`
			Assignees *[]string `json:"assignees"`
			Milestone *string   `json:"milestone"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if (input.Title != nil && (strings.TrimSpace(*input.Title) == "" || len(*input.Title) > 255)) || (input.Body != nil && len(*input.Body) > 32768) || (input.State != nil && *input.State != "open" && *input.State != "closed") || (input.Assignees != nil && !validAssignees(*input.Assignees)) {
			jsonError(w, 422, "invalid_issue", "Invalid issue fields or state.")
			return
		}
		var milestone *int64
		if input.Milestone != nil {
			var id int64
			if *input.Milestone != "0" {
				id, ok = apiID(w, *input.Milestone)
				if !ok {
					return
				}
			}
			milestone = &id
		}
		if input.Title == nil && input.Body == nil && input.State == nil && input.Assignees == nil && input.Milestone == nil {
			jsonError(w, 400, "empty_patch", "Provide an issue field to change.")
			return
		}
		result, err = s.Forgejo.EditIssue(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, forgejo.EditIssue{Title: input.Title, Body: input.Body, State: input.State, Assignees: input.Assignees, Milestone: milestone})
	}
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, issueDTO(result))
}

type commentView struct {
	ID      string           `json:"id"`
	Body    string           `json:"body"`
	User    providerUserView `json:"user"`
	Created string           `json:"created_at"`
	Updated string           `json:"updated_at"`
}

func commentDTO(c forgejo.Comment) commentView {
	return commentView{strconv.FormatInt(c.ID, 10), c.Body, providerUser(c.User), c.Created, c.Updated}
}
func (s *Server) apiIssueComments(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	if r.Method == "POST" {
		body, ok := commentBody(w, r)
		if !ok {
			return
		}
		result, err := s.Forgejo.AddIssueComment(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, body)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, commentDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	comments, metadata, err := s.Forgejo.IssueComments(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]commentView, 0, len(comments))
	for _, c := range comments {
		items = append(items, commentDTO(c))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func commentBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		Body string `json:"body"`
	}
	if !decodeAPIObject(w, r, &input) {
		return "", false
	}
	if strings.TrimSpace(input.Body) == "" || len(input.Body) > 32768 {
		jsonError(w, 422, "invalid_comment", "Provide a comment of at most 32 KiB.")
		return "", false
	}
	return input.Body, true
}
func (s *Server) apiEditComment(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("comment"))
	if !ok {
		return
	}
	body, ok := commentBody(w, r)
	if !ok {
		return
	}
	result, err := s.Forgejo.EditIssueComment(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, body)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, commentDTO(result))
}
func (s *Server) apiIssueLabels(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	var input struct {
		Labels *[]string `json:"labels"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if input.Labels == nil || len(*input.Labels) > 100 {
		jsonError(w, 422, "invalid_labels", "Provide an array of at most 100 label IDs.")
		return
	}
	ids := make([]int64, 0, len(*input.Labels))
	for _, value := range *input.Labels {
		id, ok := apiID(w, value)
		if !ok {
			return
		}
		ids = append(ids, id)
	}
	labels, err := s.Forgejo.ReplaceIssueLabels(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, ids)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]labelView, 0, len(labels))
	for _, label := range labels {
		items = append(items, labelDTO(label))
	}
	jsonResponse(w, 200, struct {
		Items []labelView `json:"items"`
	}{items})
}

var labelColor = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

func readLabel(w http.ResponseWriter, r *http.Request) (forgejo.WriteLabel, bool) {
	var input forgejo.WriteLabel
	if !decodeAPIObject(w, r, &input) {
		return input, false
	}
	if strings.TrimSpace(input.Name) == "" || len(input.Name) > 255 || len(input.Description) > 4096 || !labelColor.MatchString(input.Color) {
		jsonError(w, 422, "invalid_label", "Provide a label name, six-digit hex color and bounded description.")
		return input, false
	}
	return input, true
}
func (s *Server) apiLabels(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		input, ok := readLabel(w, r)
		if !ok {
			return
		}
		result, err := s.Forgejo.CreateLabel(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, labelDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	labels, metadata, err := s.Forgejo.Labels(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]labelView, 0, len(labels))
	for _, label := range labels {
		items = append(items, labelDTO(label))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiEditLabel(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("label"))
	if !ok {
		return
	}
	input, ok := readLabel(w, r)
	if !ok {
		return
	}
	result, err := s.Forgejo.EditLabel(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, input)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, labelDTO(result))
}
func readMilestone(w http.ResponseWriter, r *http.Request) (forgejo.WriteMilestone, bool) {
	var input forgejo.WriteMilestone
	if !decodeAPIObject(w, r, &input) {
		return input, false
	}
	if strings.TrimSpace(input.Title) == "" || len(input.Title) > 255 || len(input.Description) > 8192 || (input.State != "open" && input.State != "closed") {
		jsonError(w, 422, "invalid_milestone", "Provide a title, bounded description and open/closed state.")
		return input, false
	}
	if input.DueOn != "" {
		if _, err := time.Parse(time.RFC3339, input.DueOn); err != nil {
			jsonError(w, 422, "invalid_date", "Use an RFC3339 due date.")
			return input, false
		}
	}
	return input, true
}
func (s *Server) apiMilestones(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		input, ok := readMilestone(w, r)
		if !ok {
			return
		}
		result, err := s.Forgejo.CreateMilestone(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, milestoneDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	milestones, metadata, err := s.Forgejo.Milestones(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]milestoneView, 0, len(milestones))
	for _, m := range milestones {
		items = append(items, milestoneDTO(m))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiEditMilestone(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("milestone"))
	if !ok {
		return
	}
	input, ok := readMilestone(w, r)
	if !ok {
		return
	}
	result, err := s.Forgejo.EditMilestone(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, input)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, milestoneDTO(result))
}
