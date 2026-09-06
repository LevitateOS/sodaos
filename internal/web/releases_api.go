package web

import (
	"encoding/base64"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) releaseRoutes() {
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/releases", s.apiProvider(s.apiReleases, "write:repository", "GET", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/releases/{release}", s.apiProvider(s.apiRelease, "write:repository", "GET", "PATCH"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/releases/{release}/assets", s.apiProvider(s.apiReleaseUpload, "write:repository", "POST"))
	s.mux.HandleFunc("/api/forgejo/repos/{owner}/{repo}/releases/{release}/assets/{asset}/download", s.apiProvider(s.apiReleaseDownload, "read:repository", "GET"))
}

type releaseAssetView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Size         string `json:"size"`
	Downloadable bool   `json:"downloadable"`
}
type releaseView struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Tag        string             `json:"tag_name"`
	Target     string             `json:"target_commitish"`
	Body       string             `json:"body"`
	Draft      bool               `json:"draft"`
	Prerelease bool               `json:"prerelease"`
	Published  string             `json:"published_at"`
	Assets     []releaseAssetView `json:"assets"`
}

func releaseDTO(value forgejo.Release) releaseView {
	assets := make([]releaseAssetView, 0, len(value.Assets))
	for _, asset := range value.Assets {
		assets = append(assets, releaseAssetView{strconv.FormatInt(asset.ID, 10), asset.Name, strconv.FormatInt(asset.Size, 10), asset.Type == "attachment" && attachmentUUID.MatchString(asset.UUID) && asset.Size >= 0 && asset.Size <= forgejo.DownloadLimit})
	}
	return releaseView{strconv.FormatInt(value.ID, 10), value.Name, value.Tag, value.Target, value.Body, value.Draft, value.Prerelease, value.Published, assets}
}
func validateRelease(w http.ResponseWriter, input forgejo.ReleaseFields, create bool) bool {
	if !create && ((input.Name != nil && *input.Name == "") || (input.Body != nil && *input.Body == "")) {
		jsonError(w, 422, "native_release_clear_gap", "The pinned release PATCH ignores empty title/body replacements; no clear was performed.")
		return false
	}
	if (create && (input.Tag == nil || input.Target == nil || input.Draft == nil)) || (input.Tag != nil && !validRef(*input.Tag)) || (input.Target != nil && !validRef(*input.Target)) || (input.Name != nil && len(*input.Name) > 255) || (input.Body != nil && len(*input.Body) > 32768) {
		jsonError(w, 422, "invalid_release", "Supply a valid tag/ref, bounded title/body, and explicit draft state when creating.")
		return false
	}
	return true
}
func (s *Server) apiReleases(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	if r.Method == "POST" {
		var input forgejo.ReleaseFields
		if !decodeAPIObject(w, r, &input) || !validateRelease(w, input, true) {
			return
		}
		result, err := s.Forgejo.SaveRelease(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), 0, input)
		if err != nil {
			providerError(w, err)
			return
		}
		jsonResponse(w, 201, releaseDTO(result))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	result, metadata, err := s.Forgejo.Releases(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), page)
	if err != nil {
		providerError(w, err)
		return
	}
	items := make([]releaseView, 0, len(result))
	for _, value := range result {
		items = append(items, releaseDTO(value))
	}
	jsonResponse(w, 200, pageResult(items, metadata))
}
func (s *Server) apiRelease(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("release"))
	if !ok {
		return
	}
	var result forgejo.Release
	var err error
	if r.Method == "PATCH" {
		var input forgejo.ReleaseFields
		if !decodeAPIObject(w, r, &input) || !validateRelease(w, input, false) {
			return
		}
		result, err = s.Forgejo.SaveRelease(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, input)
	} else {
		result, err = s.Forgejo.Release(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id)
	}
	if err != nil {
		providerError(w, err)
		return
	}
	if result.ID != id {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	jsonResponse(w, 200, releaseDTO(result))
}
func (s *Server) apiReleaseUpload(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("release"))
	if !ok {
		return
	}
	var input struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	data, err := base64.StdEncoding.Strict().DecodeString(input.Content)
	if err != nil || len(data) > 32768 || input.Name == "" || input.Name == "." || input.Name == ".." || len(input.Name) > 255 || strings.ContainsAny(input.Name, "/\\\r\n\x00") {
		jsonError(w, 422, "invalid_asset", "Supply a simple filename and base64 data of at most 32 KiB.")
		return
	}
	result, err := s.Forgejo.AddReleaseAsset(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, input.Name, data)
	if err != nil {
		providerError(w, err)
		return
	}
	jsonResponse(w, 201, struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{strconv.FormatInt(result.ID, 10), result.Name})
}
func (s *Server) apiReleaseDownload(w http.ResponseWriter, r *http.Request, v store.Session, token string) {
	if !validRepositoryRoute(w, r) {
		return
	}
	id, ok := apiID(w, r.PathValue("release"))
	if !ok {
		return
	}
	asset, ok := apiID(w, r.PathValue("asset"))
	if !ok {
		return
	}
	metadata, err := s.Forgejo.ReleaseAsset(r.Context(), token, r.PathValue("owner"), r.PathValue("repo"), id, asset)
	if err != nil {
		providerError(w, err)
		return
	}
	if metadata.ID != asset || !attachmentUUID.MatchString(metadata.UUID) {
		providerError(w, forgejo.ErrInvalidResponse)
		return
	}
	if metadata.Type != "attachment" {
		jsonError(w, 409, "external_asset", "External assets require native Forgejo navigation; no credentials are forwarded.")
		return
	}
	if metadata.Size < 0 || metadata.Size > forgejo.DownloadLimit {
		providerError(w, forgejo.ErrResponseTooLarge)
		return
	}
	data, err := s.Forgejo.AttachmentData(r.Context(), token, metadata.UUID)
	if err != nil {
		providerError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": metadata.Name}))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
