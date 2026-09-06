package web

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) issueActivityRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}/reactions", s.apiProvider(s.apiIssueReactions, "write:issue", "GET", "POST", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}/subscription", s.apiProvider(s.apiIssueSubscription, "write:issue", "GET", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/issues/{index}/attachments", s.apiProvider(s.apiIssueAttachments, "write:issue", "GET", "POST"))
}
func (s *Server) apiIssueReactions(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	if r.Method != "GET" {
		var input struct {
			Content string `json:"content"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if input.Content == "" || len(input.Content) > 64 {
			jsonError(w, 422, "invalid_reaction", "Provide a native configured reaction name.")
			return
		}
		if err := s.Forgejo.ReactToIssue(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, input.Content, r.Method == "DELETE"); err != nil {
			providerError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	reactions, err := s.Forgejo.IssueReactions(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return
	}
	type reactionView struct {
		Content string           `json:"content"`
		User    providerUserView `json:"user"`
	}
	items := make([]reactionView, 0, len(reactions))
	for _, reaction := range reactions {
		items = append(items, reactionView{reaction.Content, providerUser(reaction.User)})
	}
	jsonResponse(w, 200, struct {
		Items []reactionView `json:"items"`
	}{items})
}
func (s *Server) apiIssueSubscription(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	if r.Method != "GET" {
		if !decodeAPIObject(w, r, &struct{}{}) {
			return
		}
		if err := s.Forgejo.SetIssueSubscription(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, v.User.Login, r.Method == "PUT"); err != nil {
			providerError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	result, err := s.Forgejo.IssueSubscription(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, result)
}

var attachmentUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *Server) apiIssueAttachments(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	index, ok := apiID(w, r.PathValue("index"))
	if !ok {
		return
	}
	if r.Method == "POST" {
		var input struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		data, err := base64.StdEncoding.Strict().DecodeString(input.Content)
		if err != nil || len(data) > 32768 || input.Name == "" || input.Name == "." || input.Name == ".." || len(input.Name) > 255 || strings.ContainsAny(input.Name, "/\\\r\n\x00") {
			jsonError(w, 422, "invalid_attachment", "Provide a simple filename and base64 content up to 32 KiB.")
			return
		}
		result, err := s.Forgejo.AddIssueAttachment(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index, input.Name, data)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{strconv.FormatInt(result.ID, 10), result.Name})
		return
	}
	attachments, err := s.Forgejo.IssueAttachments(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), index)
	if err != nil {
		providerError(w, err)
		return
	}
	type attachmentView struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Size      string `json:"size"`
		NativeURL string `json:"native_url"`
	}
	items := make([]attachmentView, 0, len(attachments))
	for _, attachment := range attachments {
		link := ""
		if attachmentUUID.MatchString(attachment.UUID) {
			link = s.Config.ForgejoURL + "/attachments/" + url.PathEscape(attachment.UUID)
		}
		items = append(items, attachmentView{strconv.FormatInt(attachment.ID, 10), attachment.Name, strconv.FormatInt(attachment.Size, 10), link})
	}
	jsonResponse(w, 200, struct {
		Items []attachmentView `json:"items"`
	}{items})
}
