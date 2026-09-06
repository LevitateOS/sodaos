package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) workRoutes() {
	s.mux.HandleFunc("/api/forgejo/repository-search", s.apiProvider(s.apiRepositorySearch, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/work", s.apiProvider(s.apiWork, "read:issue", "GET"))
	s.mux.HandleFunc("/api/forgejo/notifications", s.apiProvider(s.apiNotifications, "read:notification", "GET"))
	s.mux.HandleFunc("/api/forgejo/notifications/{notification}", s.apiProvider(s.apiSetNotification, "write:notification", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/users/{login}", s.apiProvider(s.apiUserProfile, "read:user", "GET"))
	s.mux.HandleFunc("/api/forgejo/users/{login}/activity", s.apiProvider(s.apiUserActivity, "read:user", "GET"))
}
func (s *Server) apiRepositorySearch(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	owner := r.URL.Query().Get("owner")
	if len(q) > 1024 || (owner != "" && !validRepositoryPart(owner)) {
		jsonError(w, 400, "invalid_search", "Invalid bounded repository search.")
		return
	}
	var uid int64
	if owner != "" {
		user, err := s.Forgejo.UserProfile(r.Context(), token, owner)
		if err != nil {
			providerError(w, err)
			return
		}
		if user.ID <= 0 {
			providerError(w, forgejo.ErrInvalidResponse)
			return
		}
		uid = user.ID
	}
	result, metadata, err := s.Forgejo.SearchRepositories(r.Context(), token, q, uid, page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]repositoryView, 0, len(result))
	for _, repo := range result {
		items = append(items, repositoryDTO(repo))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}

type workView struct {
	issueView
	Repository struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Owner    string `json:"owner"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Kind string `json:"kind"`
}

func (s *Server) apiWork(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	input := forgejo.WorkQuery{Search: q.Get("q"), State: q.Get("state"), Kind: q.Get("type"), Owner: q.Get("owner"), Labels: q.Get("labels"), Milestones: q.Get("milestones"), Page: page}
	if input.State == "" {
		input.State = "open"
	}
	if (input.State != "open" && input.State != "closed" && input.State != "all") || (input.Kind != "" && input.Kind != "issues" && input.Kind != "pulls") || len(input.Search) > 1024 || len(input.Labels) > 1024 || len(input.Milestones) > 1024 || (input.Owner != "" && !validRepositoryPart(input.Owner)) {
		jsonError(w, 400, "invalid_work_filter", "Invalid bounded native search filters.")
		return
	}
	for _, field := range []struct {
		name   string
		target *bool
	}{{"assigned", &input.Assigned}, {"created", &input.Created}, {"mentioned", &input.Mentioned}, {"review_requested", &input.ReviewRequested}, {"reviewed", &input.Reviewed}} {
		raw := q.Get(field.name)
		if raw != "" && raw != "true" && raw != "false" {
			jsonError(w, 400, "invalid_work_filter", "Native personal filters must be boolean.")
			return
		}
		*field.target = raw == "true"
	}
	result, metadata, err := s.Forgejo.SearchWork(r.Context(), token, input)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]workView, 0, len(result))
	for _, item := range result {
		view := workView{issueView: issueDTO(item.Issue), Kind: "issues"}
		view.Repository.ID = strconv.FormatInt(item.Repository.ID, 10)
		view.Repository.Owner = item.Repository.Owner
		view.Repository.Name = item.Repository.Name
		view.Repository.FullName = item.Repository.FullName
		if item.Pull != nil {
			view.Kind = "pulls"
		}
		items = append(items, view)
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}

type notificationView struct {
	ID         string         `json:"id"`
	Unread     bool           `json:"unread"`
	Pinned     bool           `json:"pinned"`
	Updated    string         `json:"updated_at"`
	Title      string         `json:"title"`
	State      string         `json:"state"`
	Route      string         `json:"route"`
	Repository repositoryView `json:"repository"`
}

// Extract only a native subject number from an exact repository path. Never
// follow, proxy or return the supplied host/query/fragment as a credential target.
func notificationRoute(n forgejo.Notification) string {
	owner, name := n.Repository.Owner.Login, n.Repository.Name
	if !validRepositoryPart(owner) || !validRepositoryPart(name) {
		return ""
	}
	root := "/repositories/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	u, err := url.Parse(n.Subject.HTMLURL)
	if err != nil {
		return root
	}
	for _, kind := range []string{"issues", "pulls"} {
		prefix := "/" + owner + "/" + name + "/" + kind + "/"
		if strings.HasPrefix(u.Path, prefix) {
			raw := strings.TrimPrefix(u.Path, prefix)
			id, err := strconv.ParseInt(raw, 10, 64)
			if err == nil && id > 0 && strconv.FormatInt(id, 10) == raw {
				return root + "/" + kind + "/" + raw
			}
		}
	}
	return root
}
func (s *Server) apiNotifications(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	all := r.URL.Query().Get("all")
	if all != "" && all != "true" && all != "false" {
		jsonError(w, 400, "invalid_filter", "Use a boolean all filter.")
		return
	}
	result, metadata, err := s.Forgejo.Notifications(r.Context(), token, all == "true", page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]notificationView, 0, len(result))
	for _, n := range result {
		items = append(items, notificationView{strconv.FormatInt(n.ID, 10), n.Unread, n.Pinned, n.Updated, n.Subject.Title, n.Subject.State, notificationRoute(n), repositoryDTO(n.Repository)})
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiSetNotification(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("notification"))
	if !ok {
		return
	}
	var input struct {
		State string `json:"state"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if input.State != "read" && input.State != "unread" && input.State != "pinned" {
		jsonError(w, 422, "invalid_notification_state", "Choose read, unread or pinned.")
		return
	}
	if err := s.Forgejo.SetNotification(r.Context(), token, id, input.State); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) apiUserProfile(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	login := r.PathValue("login")
	if !validRepositoryPart(login) {
		jsonError(w, 400, "invalid_user", "Select a native username.")
		return
	}
	result, err := s.Forgejo.UserProfile(r.Context(), token, login)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, struct {
		User        providerUserView `json:"user"`
		Description string           `json:"description"`
		Location    string           `json:"location"`
		Pronouns    string           `json:"pronouns"`
		Created     string           `json:"created"`
	}{providerUser(result.User), result.Description, result.Location, result.Pronouns, result.Created})
}
func (s *Server) apiUserActivity(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	login := r.PathValue("login")
	if !validRepositoryPart(login) {
		jsonError(w, 400, "invalid_user", "Select a native username.")
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	date := r.URL.Query().Get("date")
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			jsonError(w, 400, "invalid_date", "Use a YYYY-MM-DD date.")
			return
		}
	}
	result, metadata, err := s.Forgejo.UserActivity(r.Context(), token, login, date, page)
	if err != nil {
		providerError(w, err)
		return
	}
	type activityView struct {
		ID         string           `json:"id"`
		Actor      providerUserView `json:"actor"`
		Repository *repositoryView  `json:"repository"`
		Operation  string           `json:"operation"`
		Created    string           `json:"created"`
		Content    string           `json:"content"`
		Ref        string           `json:"ref"`
	}
	items := make([]activityView, 0, len(result))
	for _, item := range result {
		var repo *repositoryView
		if item.Repository.ID > 0 && validRepositoryPart(item.Repository.Owner.Login) && validRepositoryPart(item.Repository.Name) {
			value := repositoryDTO(item.Repository)
			repo = &value
		}
		items = append(items, activityView{strconv.FormatInt(item.ID, 10), providerUser(item.Actor), repo, item.Operation, item.Created, item.Content, item.Ref})
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
