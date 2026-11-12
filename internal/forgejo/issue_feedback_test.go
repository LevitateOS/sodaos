package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func TestIssueFeedbackReconcilesLostReplyWithoutEditingRequirements(t *testing.T) {
	identities := graphIdentityFixture(t, 42, 91, 7)
	db := identities.registry.(*store.Store)
	repository, _ := identities.LocalRepository(t.Context(), 42)
	actor, _ := identities.LocalActor(t.Context(), "91")
	author, _ := identities.LocalActor(t.Context(), "7")
	var saved, mode string
	posts, requests, pages := 0, 0, 0
	var beforeRepository func()
	var origin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "token synthetic-feedback-token" {
			t.Error("missing native actor authorization")
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		comment := func(body string) map[string]any {
			return map[string]any{"id": 12, "body": body, "user": User{ID: 91}, "issue_url": origin + "/owner/demo/issues/55"}
		}
		switch {
		case r.URL.Path == "/api/v1/user":
			json.NewEncoder(w).Encode(User{ID: 91, Login: "worker"})
		case r.URL.Path == "/api/v1/repositories/42":
			if beforeRepository != nil {
				callback := beforeRepository
				beforeRepository = nil
				callback()
			}
			json.NewEncoder(w).Encode(Repository{ID: 42, Name: "demo", Owner: User{Login: "owner"}, Permissions: &RepositoryPermissions{Push: true}})
		case r.URL.Path == "/api/v1/repos/owner/demo/issues/55/comments" && r.Method == "GET":
			pages++
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			if mode == "absent" {
				w.Write([]byte("[]"))
				return
			}
			if mode == "many" && page < 3 {
				items := []map[string]any{}
				for n := 0; n < 50; n++ {
					items = append(items, map[string]any{"id": 100 + page*50 + n, "body": "Unrelated human comment", "user": User{ID: 7}})
				}
				json.NewEncoder(w).Encode(items)
				return
			}
			json.NewEncoder(w).Encode([]map[string]any{comment(saved)})
		case r.URL.Path == "/api/v1/repos/owner/demo/issues/comments/12" && r.Method == "GET":
			json.NewEncoder(w).Encode(comment(saved))
		case r.URL.Path == "/api/v1/repos/owner/demo/issues/55/comments" && r.Method == "POST":
			posts++
			var in struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			saved = in.Body
			w.WriteHeader(503)
		default:
			t.Errorf("unexpected native mutation or route: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	origin = server.URL
	credential := filepath.Join(t.TempDir(), "actor")
	if err := os.WriteFile(credential, []byte("synthetic-feedback-token"), 0600); err != nil {
		t.Fatal(err)
	}
	writer := NewIssueGraphWriter(&graphClientFixture{revision: 12, allowed: true}, New(server.URL), credential, identities)
	effects := &factoryservice.PostIssueEffects{Store: db, Graph: writer, Actor: int64(actor)}
	makePost := func(key string) store.FeedPostDetail {
		id, revision, err := db.SaveFeedPost(t.Context(), factory.PostRequest{Repository: int64(repository), Body: "Please fix the filter.", IdempotencyKey: key}, strconv.FormatInt(int64(author), 10), "Human", time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		if err = db.RecordPostIntake(t.Context(), id, revision); err != nil {
			t.Fatal(err)
		}
		pipe, err := db.PostIntake(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		next := pipe
		next.Judgment = factory.PostJudgmentApplied
		next.Route = factory.PostRouteExistingWork
		next.Issue = 55
		if err = db.AdvancePostIntake(t.Context(), pipe, next); err != nil {
			t.Fatal(err)
		}
		detail, err := db.FeedPost(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		return detail
	}
	detail := makePost("feedback-lost")
	if err := effects.DeliverPostFeedback(t.Context(), detail, 55); !errors.Is(err, factoryservice.ErrPostFeedbackPending) {
		t.Fatalf("lost reply was not retained pending: %v", err)
	}
	if posts != 1 || pages != 0 {
		t.Fatal("new delivery searched history or repeated send", posts, pages)
	}
	before := requests
	if err := effects.DeliverPostFeedback(t.Context(), detail, 55); !errors.Is(err, factoryservice.ErrPostFeedbackPending) || requests != before {
		t.Fatal("early recovery read provider or renewed dispatch", err, requests, before)
	}
	recorded, err := db.PostFeedback(t.Context(), detail.Record.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	due := recorded
	due.NotBefore = 0
	if err = db.SavePostFeedback(t.Context(), recorded, due); err != nil {
		t.Fatal(err)
	}
	// A new service instance has no in-memory send knowledge.
	effects = &factoryservice.PostIssueEffects{Store: db, Graph: writer, Actor: int64(actor)}
	if err = effects.DeliverPostFeedback(t.Context(), detail, 55); err != nil {
		t.Fatal(err)
	}
	receipt, err := db.PostFeedback(t.Context(), detail.Record.ID, 0)
	if err != nil || receipt.Receipt != "12" || posts != 1 {
		t.Fatal("recovery did not adopt original comment", receipt, err, posts)
	}
	before = requests
	if err = effects.DeliverPostFeedback(t.Context(), detail, 55); err != nil || requests != before {
		t.Fatal("known receipt recollected history", err)
	}
	marker := "<!-- soda-feed:" + detail.Record.ID + ":r0 -->"
	expected := marker + "\nFeedback from @Human in the project feed:\n\nPlease fix the filter."
	if saved != expected {
		t.Fatal("feedback text changed")
	}
	legacy := receipt
	legacy.Legacy = true
	legacy.Receipt = ""
	legacy.Complete = false
	legacy.NotBefore = 0
	legacy.Cursor = 1
	mode = "many"
	pages = 0
	first, err := writer.DeliverIssueFeedback(context.Background(), legacy, false, func(context.Context) error { return nil })
	if err != nil || first.Cursor != 3 || first.Complete || pages != 2 {
		t.Fatal("legacy lookup was unbounded or lost cursor", first, err, pages)
	}
	second, err := writer.DeliverIssueFeedback(context.Background(), first, false, func(context.Context) error { return nil })
	if err != nil || second.Receipt != "12" || pages != 3 || posts != 1 {
		t.Fatal("remaining legacy page did not recover exact comment", second, err)
	}
	mode = "absent"
	absent, err := writer.DeliverIssueFeedback(context.Background(), legacy, false, func(context.Context) error { return nil })
	if err != nil || absent.Attention != "not_found" || posts != 1 {
		t.Fatal("absence renewed dispatch", absent, err)
	}
	mode = ""
	saved = marker + "\nHuman edited this comment"
	edited, err := writer.DeliverIssueFeedback(context.Background(), legacy, false, func(context.Context) error { return nil })
	if err == nil || edited.Attention != "content_changed" || edited.Validate() != nil || saved != marker+"\nHuman edited this comment" {
		t.Fatal("edited text was adopted or changed", edited, err)
	}
	saved = expected
	point, err := writer.DeliverIssueFeedback(context.Background(), receipt, false, func(context.Context) error { return nil })
	if err != nil || point.Receipt != "12" {
		t.Fatal("point receipt lookup failed", err)
	}
	withdrawn := makePost("feedback-withdrawn")
	beforeRepository = func() {
		if _, err := db.EditFeedPost(t.Context(), withdrawn.Record.ID, strconv.FormatInt(int64(author), 10), "Concurrent human edit", 0, time.Now().Unix()); err != nil {
			t.Error(err)
		}
	}
	if err = effects.DeliverPostFeedback(t.Context(), withdrawn, 55); err == nil || posts != 1 {
		t.Fatal("source withdrawal during I/O reached POST", err, posts)
	}
	t.Log("PASS: realPG post journal + native HTTP: one lost-reply send, local due hold, restarted-service receipt adoption, bounded legacy paging, absence/mismatched human-text refusal, direct receipt lookup and source-edit-before-send refusal")
}
