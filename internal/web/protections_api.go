package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) protectionRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/branch-protections", s.apiProvider(s.apiBranchProtections, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/branch-protections/{rule}", s.apiProvider(s.apiEditBranchProtection, "write:repository", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/tag-protections", s.apiProvider(s.apiTagProtections, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/tag-protections/{protection}", s.apiProvider(s.apiEditTagProtection, "write:repository", "PATCH"))
}
func validProtectionRule(rule string) bool {
	if strings.TrimSpace(rule) == "" || len(rule) > 1024 || strings.ContainsAny(rule, "\x00\r\n\\") {
		return false
	}
	for _, part := range strings.Split(rule, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
func validateProtection(w http.ResponseWriter, input forgejo.ProtectionFields) bool {
	if input.RequiredApprovals != nil && (*input.RequiredApprovals < 0 || *input.RequiredApprovals > 1000) {
		jsonError(w, 422, "invalid_protection", "Required approvals must be between 0 and 1000.")
		return false
	}
	for _, names := range []*[]string{input.PushUsers, input.PushTeams, input.MergeUsers, input.MergeTeams, input.ApprovalUsers, input.ApprovalTeams} {
		if names != nil && !validAssignees(*names) {
			jsonError(w, 422, "invalid_protection", "Provide bounded native user/team names.")
			return false
		}
	}
	for _, patterns := range []*string{input.ProtectedFiles, input.UnprotectedFiles} {
		if patterns != nil && len(*patterns) > 4096 {
			jsonError(w, 422, "invalid_protection", "File patterns exceed supported size.")
			return false
		}
	}
	if input.StatusChecks != nil {
		if len(*input.StatusChecks) > 100 {
			jsonError(w, 422, "invalid_protection", "Too many status contexts.")
			return false
		}
		for _, context := range *input.StatusChecks {
			if context == "" || len(context) > 255 || strings.ContainsAny(context, "\x00\r\n") {
				jsonError(w, 422, "invalid_protection", "Invalid native status context.")
				return false
			}
		}
	}
	// Pinned native PATCH only applies nested push flags inside their enabled
	// parent branch. Reject otherwise-ignored combinations rather than fake success.
	if input.EnablePushWhitelist != nil && (input.EnablePush == nil || !*input.EnablePush) {
		jsonError(w, 422, "invalid_protection_dependency", "Push allowlist edits require explicit enable_push true.")
		return false
	}
	if input.PushWhitelistDeployKeys != nil && (input.EnablePushWhitelist == nil || !*input.EnablePushWhitelist || input.EnablePush == nil || !*input.EnablePush) {
		jsonError(w, 422, "invalid_protection_dependency", "Deploy-key allowlisting requires explicit enabled push and push allowlist.")
		return false
	}
	return true
}
func (s *Server) apiBranchProtections(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input forgejo.BranchProtection
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if !validProtectionRule(input.Name) {
			jsonError(w, 422, "invalid_rule", "Provide a bounded native protection rule.")
			return
		}
		if !validateProtection(w, input.ProtectionFields) {
			return
		}
		result, err := s.Forgejo.CreateBranchProtection(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, result)
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	items, metadata, err := s.Forgejo.BranchProtections(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiEditBranchProtection(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	rule := r.PathValue("rule")
	if !validProtectionRule(rule) {
		jsonError(w, 400, "invalid_rule", "Select a native rule.")
		return
	}
	var input forgejo.ProtectionFields
	if !decodeAPIObject(w, r, &input) || !validateProtection(w, input) {
		return
	}
	result, err := s.Forgejo.EditBranchProtection(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), rule, input)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, result)
}

type tagProtectionView struct {
	ID      string   `json:"id"`
	Pattern string   `json:"name_pattern"`
	Users   []string `json:"whitelist_usernames"`
	Teams   []string `json:"whitelist_teams"`
}

func tagProtectionDTO(p forgejo.TagProtection) tagProtectionView {
	return tagProtectionView{strconv.FormatInt(p.ID, 10), p.Pattern, append([]string{}, p.Users...), append([]string{}, p.Teams...)}
}
func validTagProtection(input forgejo.TagProtectionFields) bool {
	return (input.Pattern == nil || validProtectionRule(*input.Pattern)) && (input.Users == nil || validAssignees(*input.Users)) && (input.Teams == nil || validAssignees(*input.Teams))
}
func (s *Server) apiTagProtections(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input forgejo.TagProtectionFields
		if !decodeAPIObject(w, r, &input) {
			return
		}
		if input.Pattern == nil || !validTagProtection(input) {
			jsonError(w, 422, "invalid_tag_protection", "Provide a native name pattern and bounded allowed user/team names.")
			return
		}
		result, err := s.Forgejo.CreateTagProtection(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, tagProtectionDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	protections, metadata, err := s.Forgejo.TagProtections(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]tagProtectionView, 0, len(protections))
	for _, p := range protections {
		items = append(items, tagProtectionDTO(p))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiEditTagProtection(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("protection"))
	if !ok {
		return
	}
	var input forgejo.TagProtectionFields
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if !validTagProtection(input) {
		jsonError(w, 422, "invalid_tag_protection", "Invalid native protection fields.")
		return
	}
	result, err := s.Forgejo.EditTagProtection(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, input)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 200, tagProtectionDTO(result))
}
