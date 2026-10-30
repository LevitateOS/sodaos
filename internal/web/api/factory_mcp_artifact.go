package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"mime"
	"net/http"
	"strconv"
	"time"
)

func artifactPayloadSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"upload_id": {Type: "string", Description: "Your stable 32-hex upload ID. Reuse across chunks and advance offset using next_offset."}, "name": {Type: "string"}, "media_type": {Type: "string"}, "offset": {Type: "integer", Description: "Bytes already uploaded; starts at zero."}, "content_base64": {Type: "string", Description: "Base64 of at most 32768 bytes. Total file at most 8 MiB."}, "final": {Type: "boolean", Description: "True for last chunk: publish the artifact and its feed card."}}, Required: []string{"upload_id", "name", "media_type", "offset", "content_base64", "final"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

type factoryMCPArtifactWriter interface {
	UploadWorkerArtifact(context.Context, factory.MCPCapability, []byte) (factory.FeedArtifact, string, error)
}

func (s *factoryCatalogServer) serveArtifact(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, factory.MCPToolPublishArtifact, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	writer, ok := s.deps.Submissions.(factoryMCPArtifactWriter)
	if !ok {
		return nil, nil, errors.New("artifact upload is unavailable")
	}
	raw, err := json.Marshal(args.Payload)
	if err != nil {
		return nil, nil, err
	}
	artifact, postID, err := writer.UploadWorkerArtifact(ctx, cap, raw)
	if err != nil {
		return nil, nil, err
	}
	out := map[string]any{"artifact_id": artifact.ID, "next_offset": artifact.Size, "complete": postID != "", "post_id": postID}
	if postID != "" {
		out["digest"] = artifact.Digest
		out["artifact"] = artifact
	}
	return mcpJSONResult(out)
}
func (s *API) apiFactoryArtifact(w http.ResponseWriter, r *http.Request, v store.Session) {
	repo, err := strconv.ParseInt(r.URL.Query().Get("repository"), 10, 64)
	if err != nil || repo <= 0 || !factory.ValidID(r.PathValue("id")) {
		auth.JSONError(w, 400, "invalid_request", "Artifact identity is invalid.")
		return
	}
	if _, err = s.visibleRepository(r, v, repo); err != nil {
		auth.JSONError(w, 404, "not_found", "Artifact is unavailable.")
		return
	}
	a, data, err := s.Store.FeedArtifact(r.Context(), repo, r.PathValue("id"))
	if err != nil {
		code := 503
		if errors.Is(err, store.ErrNotFound) {
			code = 404
		}
		auth.JSONError(w, code, "unavailable", "Artifact is unavailable.")
		return
	}
	w.Header().Set("Content-Type", a.MediaType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Name}))
	http.ServeContent(w, r, a.Name, time.Time{}, bytes.NewReader(data))
}
