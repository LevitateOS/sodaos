package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) organizationRoutes() {
	s.mux.HandleFunc("/api/forgejo/organizations", s.apiProvider(s.apiOrganizations, "write:organization", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}", s.apiProvider(s.apiOrganization, "write:organization", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/members", s.apiProvider(s.apiOrganizationMembers, "read:organization", "GET"))
	s.mux.HandleFunc("/api/forgejo/organizations/{org}/teams", s.apiProvider(s.apiOrganizationTeams, "write:organization", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/teams/{team}", s.apiProvider(s.apiTeam, "write:organization", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/teams/{team}/members", s.apiProvider(s.apiTeamMembers, "read:organization", "GET"))
	s.mux.HandleFunc("/api/forgejo/teams/{team}/members/{login}", s.apiProvider(s.apiTeamMember, "write:organization", "PUT", "DELETE"))
	s.mux.HandleFunc("/api/forgejo/teams/{team}/repositories", s.apiProvider(s.apiTeamRepositories, "read:organization", "GET"))
	s.mux.HandleFunc("/api/forgejo/teams/{team}/repositories/{owner}/{repo}", s.apiProvider(s.apiTeamRepository, "write:organization", "PUT", "DELETE"))
}

type organizationView struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	FullName            string `json:"full_name"`
	Description         string `json:"description"`
	Email               string `json:"email"`
	Website             string `json:"website"`
	Location            string `json:"location"`
	Visibility          string `json:"visibility"`
	RepoAdminTeamAccess bool   `json:"repo_admin_change_team_access"`
}

func organizationDTO(org forgejo.Organization) organizationView {
	name := org.Name
	if name == "" {
		name = org.Username
	}
	return organizationView{strconv.FormatInt(org.ID, 10), name, org.FullName, org.Description, org.Email, org.Website, org.Location, org.Visibility, org.RepoAdminTeamAccess}
}
func validOrganizationFields(fields forgejo.OrganizationFields) bool {
	if len(fields.FullName) > 255 || len(fields.Description) > 4096 || len(fields.Location) > 255 || (fields.Email != nil && (len(*fields.Email) > 255 || strings.ContainsAny(*fields.Email, "\r\n\x00"))) {
		return false
	}
	if fields.Visibility != "" && fields.Visibility != "public" && fields.Visibility != "limited" && fields.Visibility != "private" {
		return false
	}
	if fields.Website != "" {
		u, err := url.Parse(fields.Website)
		if err != nil || len(fields.Website) > 2048 || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
			return false
		}
	}
	return true
}
func validOrganizationRoute(w http.ResponseWriter, r *http.Request) bool {
	if !validRepositoryPart(r.PathValue("org")) {
		jsonError(w, 400, "invalid_organization", "Select a native organization name.")
		return false
	}
	return true
}
func (s *Server) apiOrganizations(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if r.Method == "POST" {
		var input forgejo.CreateOrganization
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validRepositoryPart(input.Username) || !validOrganizationFields(input.OrganizationFields) {
			jsonError(w, 422, "invalid_organization", "Provide a native organization name and bounded profile fields.")
			return
		}
		result, err := s.Forgejo.CreateOrganization(r.Context(), token, input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, organizationDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	mine := r.URL.Query().Get("mine")
	if mine != "" && mine != "true" && mine != "false" {
		jsonError(w, 400, "invalid_filter", "Use a boolean mine filter.")
		return
	}
	orgs, metadata, err := s.Forgejo.Organizations(r.Context(), token, page, mine == "true")
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]organizationView, 0, len(orgs))
	for _, org := range orgs {
		items = append(items, organizationDTO(org))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiOrganization(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validOrganizationRoute(w, r) {
		return
	}
	var input struct {
		FullName            *string `json:"full_name"`
		Description         *string `json:"description"`
		Email               *string `json:"email"`
		Website             *string `json:"website"`
		Location            *string `json:"location"`
		Visibility          *string `json:"visibility"`
		RepoAdminTeamAccess *bool   `json:"repo_admin_change_team_access"`
	}
	if r.Method == "PATCH" && !decodeAPIObject(w, r, &input) {
		return
	}
	current, err := s.Forgejo.Organization(r.Context(), token, r.PathValue("org"))
	if err != nil {
		providerError(w, err)
		return
	}
	if r.Method == "GET" {
		jsonResponse(w, 200, organizationDTO(current))
		return
	}
	fields := forgejo.OrganizationFields{FullName: current.FullName, Description: current.Description, Website: current.Website, Location: current.Location, Visibility: current.Visibility, Email: input.Email, RepoAdminTeamAccess: input.RepoAdminTeamAccess}
	if input.FullName != nil {
		fields.FullName = *input.FullName
	}
	if input.Description != nil {
		fields.Description = *input.Description
	}
	if input.Website != nil {
		fields.Website = *input.Website
	}
	if input.Location != nil {
		fields.Location = *input.Location
	}
	if input.Visibility != nil {
		fields.Visibility = *input.Visibility
	}
	if !validOrganizationFields(fields) {
		jsonError(w, 422, "invalid_organization", "Invalid bounded native organization profile.")
		return
	}
	result, err := s.Forgejo.EditOrganization(r.Context(), token, r.PathValue("org"), fields)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, organizationDTO(result))
}
func (s *Server) apiOrganizationMembers(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validOrganizationRoute(w, r) {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	users, metadata, err := s.Forgejo.OrganizationMembers(r.Context(), token, r.PathValue("org"), page)
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

type teamView struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Organization   organizationView  `json:"organization"`
	Permission     string            `json:"permission"`
	IncludesAll    bool              `json:"includes_all_repositories"`
	CanCreateRepos bool              `json:"can_create_org_repo"`
	Units          map[string]string `json:"units_map"`
}

func teamDTO(team forgejo.Team) teamView {
	return teamView{strconv.FormatInt(team.ID, 10), team.Name, team.Description, organizationDTO(team.Organization), team.Permission, team.IncludesAll, team.CanCreateRepos, team.Units}
}
func validTeamFields(fields forgejo.TeamFields, create bool) bool {
	if (create && strings.TrimSpace(fields.Name) == "") || len(fields.Name) > 255 || (fields.Description != nil && len(*fields.Description) > 4096) {
		return false
	}
	if fields.Permission != "" && fields.Permission != "read" && fields.Permission != "write" && fields.Permission != "admin" {
		return false
	}
	if create && fields.Permission == "" {
		return false
	}
	if (fields.IncludesAll != nil || fields.Units != nil) && fields.Permission == "" {
		return false
	}
	if fields.Units != nil && (len(fields.Units) == 0 || len(fields.Units) > 10 || fields.Permission == "admin") {
		return false
	}
	for unit, permission := range fields.Units {
		switch unit {
		case "repo.code", "repo.issues", "repo.ext_issues", "repo.pulls", "repo.releases", "repo.wiki", "repo.ext_wiki", "repo.projects", "repo.packages", "repo.actions":
		default:
			return false
		}
		if permission != "none" && permission != "read" && permission != "write" {
			return false
		}
	}
	return true
}
func (s *Server) apiOrganizationTeams(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validOrganizationRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input forgejo.TeamFields
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validTeamFields(input, true) {
			jsonError(w, 422, "invalid_team", "Provide native team name, permission and bounded settings; unit overrides require explicit non-admin permission.")
			return
		}
		result, err := s.Forgejo.CreateTeam(r.Context(), token, r.PathValue("org"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, teamDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	teams, metadata, err := s.Forgejo.OrganizationTeams(r.Context(), token, r.PathValue("org"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]teamView, 0, len(teams))
	for _, team := range teams {
		items = append(items, teamDTO(team))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiTeam(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("team"))
	if !ok {
		return
	}
	if r.Method == "PATCH" {
		var input forgejo.TeamFields
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validTeamFields(input, false) {
			jsonError(w, 422, "invalid_team", "Invalid native team settings; repository/unit policy changes require explicit native permission.")
			return
		}
		result, err := s.Forgejo.EditTeam(r.Context(), token, id, input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 200, teamDTO(result))
		return
	}
	result, err := s.Forgejo.Team(r.Context(), token, id)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, teamDTO(result))
}
func (s *Server) apiTeamMembers(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("team"))
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	users, metadata, err := s.Forgejo.TeamMembers(r.Context(), token, id, page)
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
func (s *Server) apiTeamMember(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("team"))
	if !ok {
		return
	}
	login := r.PathValue("login")
	if !validRepositoryPart(login) {
		jsonError(w, 400, "invalid_user", "Select a native username.")
		return
	}
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	if err := s.Forgejo.SetTeamMember(r.Context(), token, id, login, r.Method == "DELETE"); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) apiTeamRepositories(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("team"))
	if !ok {
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	repos, metadata, err := s.Forgejo.TeamRepositories(r.Context(), token, id, page)
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
func (s *Server) apiTeamRepository(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	id, ok := apiID(w, r.PathValue("team"))
	if !ok || !validRepositoryRoute(w, r) {
		return
	}
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	if err := s.Forgejo.SetTeamRepository(r.Context(), token, id, r.PathValue("owner"), r.PathValue("repo"), r.Method == "DELETE"); err != nil {
		providerError(w, err)
		return
	}
	w.WriteHeader(204)
}
