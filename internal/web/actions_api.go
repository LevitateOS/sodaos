package web

import (
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) actionRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/runs", s.apiProvider(s.apiActionRuns, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/runs/{run}", s.apiProvider(s.apiActionRun, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/tasks", s.apiProvider(s.apiActionTasks, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/workflows", s.apiProvider(s.apiWorkflows, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/workflows/{workflow}/dispatch", s.apiProvider(s.apiDispatchWorkflow, "write:repository", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/secrets", s.apiProvider(s.apiActionSecrets, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/secrets/{name}", s.apiProvider(s.apiActionSecret, "write:repository", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/variables", s.apiProvider(s.apiActionVariables, "read:repository", "GET"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/actions/variables/{name}", s.apiProvider(s.apiActionVariable, "write:repository", "POST", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/actions/secrets", s.apiProvider(s.apiActionSecrets, "read:organization", "GET"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/actions/secrets/{name}", s.apiProvider(s.apiActionSecret, "write:organization", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/actions/variables", s.apiProvider(s.apiActionVariables, "read:organization", "GET"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/actions/variables/{name}", s.apiProvider(s.apiActionVariable, "write:organization", "POST", "PUT", "DELETE"))
}

type actionRunView struct {
	ID           string `json:"id"`
	Number       string `json:"number"`
	Title        string `json:"title"`
	Workflow     string `json:"workflow"`
	SHA          string `json:"sha"`
	Ref          string `json:"ref"`
	Status       string `json:"status"`
	Event        string `json:"event"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
	NeedApproval bool   `json:"need_approval"`
}

func actionRunDTO(run forgejo.ActionRun) actionRunView {
	return actionRunView{strconv.FormatInt(run.ID, 10), strconv.FormatInt(run.Number, 10), run.Title, run.Workflow, run.SHA, run.Ref, run.Status, run.Event, run.Created, run.Updated, run.NeedApproval}
}
func (s *Server) apiActionRuns(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	input := forgejo.ActionRunQuery{Page: page, Ref: q.Get("ref"), Workflow: q.Get("workflow"), Status: q.Get("status")}
	if (input.Ref != "" && !validRef(input.Ref)) || len(input.Workflow) > 4096 {
		jsonError(w, 400, "invalid_run_filter", "Invalid native run filter.")
		return
	}
	switch input.Status {
	case "", "unknown", "waiting", "running", "success", "failure", "cancelled", "skipped", "blocked":
	default:
		jsonError(w, 400, "invalid_run_status", "Select a supported native run status.")
		return
	}
	runs, metadata, err := s.Forgejo.ActionRuns(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]actionRunView, 0, len(runs))
	for _, run := range runs {
		items = append(items, actionRunDTO(run))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiActionRun(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("run"))
	if !ok {
		return
	}
	result, err := s.Forgejo.ActionRun(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id)
	if err != nil {
		providerError(w, err)
		return
	}
	if result.ID != id {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 200, actionRunDTO(result))
}
func (s *Server) apiActionTasks(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	result, metadata, err := s.Forgejo.ActionTasks(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	type taskView struct {
		ID        string `json:"id"`
		RunNumber string `json:"run_number"`
		Name      string `json:"name"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		SHA       string `json:"sha"`
		Ref       string `json:"ref"`
		Workflow  string `json:"workflow"`
	}
	items := make([]taskView, 0, len(result))
	for _, task := range result {
		items = append(items, taskView{strconv.FormatInt(task.ID, 10), strconv.FormatInt(task.RunNumber, 10), task.Name, task.Title, task.Status, task.SHA, task.Ref, task.Workflow})
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiWorkflows(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	ref := r.URL.Query().Get("ref")
	if !validRef(ref) {
		jsonError(w, 400, "invalid_ref", "Choose an explicit native ref.")
		return
	}
	files, err := s.Forgejo.WorkflowFiles(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), ref)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, struct {
		Items []forgejo.WorkflowFile `json:"items"`
	}{files})
}
func (s *Server) apiDispatchWorkflow(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	name := r.PathValue("workflow")
	if !fs.ValidPath(name) || len(name) > 1024 || strings.ContainsAny(name, "\x00\r\n\\") || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
		jsonError(w, 400, "invalid_workflow", "Select a native relative workflow filename.")
		return
	}
	var input struct {
		Ref    string            `json:"ref"`
		Inputs map[string]string `json:"inputs"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if !validRef(input.Ref) || len(input.Inputs) > 100 {
		jsonError(w, 422, "invalid_dispatch", "Provide an explicit ref and bounded native inputs.")
		return
	}
	size := 0
	for key, value := range input.Inputs {
		size += len(key) + len(value)
		if key == "" || len(key) > 255 {
			jsonError(w, 422, "invalid_dispatch", "Invalid native input name.")
			return
		}
	}
	if size > 32768 {
		jsonError(w, 422, "invalid_dispatch", "Inputs exceed supported size.")
		return
	}
	result, err := s.Forgejo.DispatchWorkflow(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), name, input.Ref, input.Inputs)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 201, struct {
		ID     string   `json:"id"`
		Number string   `json:"number"`
		Jobs   []string `json:"jobs"`
	}{strconv.FormatInt(result.ID, 10), strconv.FormatInt(result.Number, 10), result.Jobs})
}
func (s *Server) actionConfig(w http.ResponseWriter, r *http.Request) (forgejo.ActionsConfig, bool) {
	// Namespace comes only from one of the explicit registered route patterns.
	if r.PathValue("org") != "" {
		if !validOrganizationRoute(w, r) {
			return forgejo.ActionsConfig{}, false
		}
		return s.Forgejo.OrganizationActions(r.PathValue("org")), true
	}
	if !validRepositoryRoute(w, r) {
		return forgejo.ActionsConfig{}, false
	}
	return s.Forgejo.RepositoryActions(r.PathValue("owner"), r.PathValue("repo")), true
}

var actionConfigName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,254}$`)

func (s *Server) apiActionSecrets(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	target, ok := s.actionConfig(w, r)
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := target.Secrets(r.Context(), token, page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiActionSecret(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	target, ok := s.actionConfig(w, r)
	if !ok {
		return
	}
	name := r.PathValue("name")
	if !actionConfigName.MatchString(name) {
		jsonError(w, 422, "invalid_secret_name", "Use a bounded native secret name.")
		return
	}
	var input struct {
		Data *string `json:"data"`
	}
	data := ""
	if r.Method == "DELETE" {
		if !decodeAPIObject(w, r, &struct{}{}) {
			return
		}
	} else {
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if input.Data == nil || len(*input.Data) > 32768 {
			jsonError(w, 422, "secret_too_large", "Supply an explicit secret value of at most 32 KiB.")
			return
		}
	}
	if input.Data != nil {
		data = *input.Data
	}
	if err := target.SetSecret(r.Context(), token, name, data, r.Method == "DELETE"); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) apiActionVariables(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	target, ok := s.actionConfig(w, r)
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := target.Variables(r.Context(), token, page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiActionVariable(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	target, ok := s.actionConfig(w, r)
	if !ok {
		return
	}
	name := r.PathValue("name")
	if !actionConfigName.MatchString(name) {
		jsonError(w, 422, "invalid_variable_name", "Use a bounded native variable name.")
		return
	}
	var input struct {
		Value *string `json:"value"`
	}
	value := ""
	if r.Method == "DELETE" {
		if !decodeAPIObject(w, r, &struct{}{}) {
			return
		}
	} else {
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if input.Value == nil || len(*input.Value) > 32768 {
			jsonError(w, 422, "variable_too_large", "Supply an explicit variable value of at most 32 KiB.")
			return
		}
	}
	if input.Value != nil {
		value = *input.Value
	}
	if err := target.SetVariable(r.Context(), token, name, value, r.Method == "POST", r.Method == "DELETE"); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
