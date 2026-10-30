package store

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestFactoryArtifactChunksPublishOnlyWhenComplete(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	cap := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	upload := factory.NewID()
	send := func(offset int, content string, final bool) (factory.MCPSubmission, factory.MCPReceipt, error) {
		raw, _ := json.Marshal(factory.ArtifactUpload{UploadID: upload, Name: "sample.txt", MediaType: "text/plain", Offset: offset, ContentBase64: base64.StdEncoding.EncodeToString([]byte(content)), Final: final})
		sub := testMCPSubmissionRow(cap, factory.MCPToolPublishArtifact, raw)
		r, err := s.RecordMCPSubmission(ctx, sub)
		return sub, r, err
	}
	first, r, err := send(0, "hello ", false)
	if err != nil || r.FeedPostID != "" || r.ArtifactID == "" {
		t.Fatalf("first: %+v %v", r, err)
	}
	if _, _, err = s.FeedArtifact(ctx, cap.Repository, r.ArtifactID); err == nil {
		t.Fatal("incomplete upload visible")
	}
	replay, err := s.RecordMCPSubmission(ctx, first)
	if err != nil || replay != r {
		t.Fatal("chunk replay changed result")
	}
	if _, _, err = send(5, "wrong", true); err == nil {
		t.Fatal("wrong offset accepted")
	}
	_, done, err := send(6, "world", true)
	if err != nil || done.FeedPostID == "" {
		t.Fatalf("finish: %+v %v", done, err)
	}
	a, content, err := s.FeedArtifact(ctx, cap.Repository, done.ArtifactID)
	if err != nil || !bytes.Equal(content, []byte("hello world")) || a.MediaType != "text/plain" {
		t.Fatalf("content: %+v %q %v", a, content, err)
	}
	if _, _, err = s.FeedArtifact(ctx, cap.Repository+1, done.ArtifactID); err == nil {
		t.Fatal("foreign repo artifact visible")
	}
	detail, err := s.FeedPost(ctx, done.FeedPostID)
	if err != nil || len(detail.Payload.Communication.Artifacts) != 1 || detail.Payload.Communication.Artifacts[0].Href != a.Href {
		t.Fatalf("artifact feed: %+v %v", detail, err)
	}
	raw, _ := json.Marshal(map[string]any{"title": "Screenshot", "summary": "Attached output", "completed": []string{"preview"}, "artifacts": []map[string]any{{"id": a.ID, "href": "https://untrusted.invalid/a", "name": "forged"}}})
	sub := testMCPSubmissionRow(cap, factory.MCPToolPartial, raw)
	linked, err := s.RecordMCPSubmission(ctx, sub)
	if err != nil {
		t.Fatal(err)
	}
	detail, err = s.FeedPost(ctx, linked.FeedPostID)
	if err != nil || detail.Payload.Communication.Artifacts[0].Href != a.Href || detail.Payload.Communication.Artifacts[0].Name != "sample.txt" {
		t.Fatal("artifact metadata not server-resolved")
	}
}
